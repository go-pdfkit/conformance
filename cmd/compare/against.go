package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// previous is one row of an earlier run's -timings file: how long we took on one
// page, and whether that page was comparable at all.
type previous struct {
	ours  time.Duration
	share float64
}

// readTimings reads a -timings file written by an earlier run.
//
// It exists because a byte-identity sweep cannot see a slowdown. A refactor that
// keeps every pixel can multiply the work behind them, and one did: a picture of
// one component with a soft mask lost a 256-entry memo and converted 3.93 million
// pixels one at a time, on 86 pages of this corpus, at two to twelve times slower,
// through three byte-identity sweeps of 3 215 documents that all came back clean.
//
// So the pixels and the time need two instruments, and this is the second one.
func readTimings(path string) (map[string]previous, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]previous{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for line := 0; sc.Scan(); line++ {
		fields := strings.Split(sc.Text(), "\t")
		if line == 0 || len(fields) < 7 {
			continue
		}
		ns, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			continue
		}
		share, err := strconv.ParseFloat(fields[5], 64)
		if err != nil {
			continue
		}
		out[key(fields[0], fields[1], fields[2])] = previous{time.Duration(ns), share}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// key names one page the same way in both runs.
func key(population, document, page string) string {
	return population + "\t" + document + "\t" + page
}

// slower is one page that took materially longer than it did before.
type slower struct {
	key     string
	was, is time.Duration
	factor  float64
}

// regressions reports the pages that got slower by at least factor, ignoring the
// ones whose comparability changed.
//
// A page that was blank before and draws now is SLOWER and is not a regression --
// it is the point of the change that admitted it. Share tells those apart: a page
// that could not be compared before, or that disagreed by 10% or more, is left
// out rather than reported as a loss.
//
// The thresholds are deliberately coarse. One timing per page is noisy: in the run
// that found the memo defect, three pages moved by more than 7x for no reason a
// re-measurement could reproduce. These are CANDIDATES, and the report says so.
func regressions(was map[string]previous, now []timedPage, factor float64, atLeast time.Duration) []slower {
	var out []slower
	for _, p := range now {
		b, ok := was[p.key]
		if !ok || b.ours <= 0 || p.ours <= 0 {
			continue
		}
		// A factor on a page that takes no time is not a finding. Comparing two
		// runs of this corpus after a real defect had been fixed, four of the six
		// largest factors left were pages under two milliseconds -- 0.3 ms to
		// 1.5 ms reads as five times slower and says nothing at all.
		if p.ours < atLeast {
			continue
		}
		if b.share < 0 || b.share >= 0.10 || p.share < 0 || p.share >= 0.10 {
			continue
		}
		if f := float64(p.ours) / float64(b.ours); f >= factor {
			out = append(out, slower{p.key, b.ours, p.ours, f})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].factor != out[j].factor {
			return out[i].factor > out[j].factor
		}
		return out[i].key < out[j].key
	})
	return out
}

// timedPage is one page of the run in progress.
type timedPage struct {
	key   string
	ours  time.Duration
	share float64
}

// reportRegressions prints what got slower, and says what the reader may conclude.
func reportRegressions(out io.Writer, rs []slower, factor float64) {
	if len(rs) == 0 {
		fmt.Fprintf(out, "\nno page is %.1fx slower than the run compared against\n", factor)
		return
	}
	fmt.Fprintf(out, "\n%d page(s) at least %.1fx slower than the run compared against.\n", len(rs), factor)
	fmt.Fprintln(out, "These are CANDIDATES: one timing per page is noisy, and a page whose")
	fmt.Fprintln(out, "comparability changed is excluded rather than reported. Re-measure before")
	fmt.Fprintln(out, "believing any single row.")
	for _, r := range rs {
		f := strings.SplitN(r.key, "\t", 3)
		fmt.Fprintf(out, "  %6.2fx  %10v -> %-10v  %s  %s page %s\n",
			r.factor, r.was.Round(time.Millisecond), r.is.Round(time.Millisecond), f[0], f[1], f[2])
	}
}
