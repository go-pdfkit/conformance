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
	// confirmed says the page was drawn again and was still slower; dropped says
	// it was drawn again and was not. Neither is set when nothing re-measured it.
	confirmed, dropped bool
	path               string
	page               int
}

// regressions reports the pages that got slower by at least factor, ignoring the
// ones whose comparability changed.
//
// A page that was blank before and draws now is SLOWER and is not a regression --
// it is the point of the change that admitted it. Share tells those apart: a page
// that could not be compared before, or that disagreed by 10% or more, is left
// out rather than reported as a loss.
//
// The thresholds are deliberately coarse, and there are three of them because one
// timing per page is noisy in two different ways: a page can be slow by accident
// (so a minimum increase), and a fast page divides a fixed accident into a huge
// ratio (so a minimum duration). Even with all three these are CANDIDATES, and the
// report says so.
func regressions(was map[string]previous, now []timedPage, factor float64, atLeast, slowerBy time.Duration) []slower {
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
		// BOTH a factor and an absolute increase, because a factor alone is not
		// a signal on this measurement. Run against a reference taken at a
		// different machine load, `-factor 2 -atleast 10ms` produced THIRTEEN
		// candidates and every one of them was noise: re-measured in one
		// interleaved run they came out at 0.92x to 1.01x. Their increases were
		// at most +88ms. The one real regression this check exists for was
		// +337ms, on a page that went from 32ms to 369ms.
		//
		// So the discriminator is the increase, not the ratio: scheduling noise
		// is roughly additive and a small page divides it into a large factor.
		if f := float64(p.ours) / float64(b.ours); f >= factor && p.ours-b.ours >= slowerBy {
			out = append(out, slower{key: p.key, was: b.ours, is: p.ours, factor: f,
				path: p.path, page: p.page})
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
	// path and page say where to find this page again, because a candidate is
	// only believable once something has drawn it a second time.
	path string
	page int
}

// reportRegressions prints what got slower, and says what the reader may conclude.
func reportRegressions(out io.Writer, rs []slower, factor float64) {
	if len(rs) == 0 {
		fmt.Fprintf(out, "\nno page is %.1fx slower than the run compared against\n", factor)
		return
	}
	kept := 0
	for _, r := range rs {
		if !r.dropped {
			kept++
		}
	}
	fmt.Fprintf(out, "\n%d page(s) at least %.1fx slower than the run compared against.\n", kept, factor)
	fmt.Fprintln(out, "These are CANDIDATES: one timing per page is noisy, and a page whose")
	fmt.Fprintln(out, "comparability changed is excluded rather than reported. Re-measure before")
	fmt.Fprintln(out, "believing any single row.")
	for _, r := range rs {
		f := strings.SplitN(r.key, "\t", 3)
		mark := "  "
		switch {
		case r.confirmed:
			mark = "! " // drawn again and still slower
		case r.dropped:
			mark = "~ " // drawn again and not slower: noise
		}
		fmt.Fprintf(out, "%s%6.2fx  %10v -> %-10v  %s  %s page %s\n",
			mark, r.factor, r.was.Round(time.Millisecond), r.is.Round(time.Millisecond), f[0], f[1], f[2])
	}
	if kept != len(rs) {
		fmt.Fprintf(out, "  %d dropped by re-measurement (~), %d confirmed (!)\n", len(rs)-kept, kept)
	}
}

// confirm draws each candidate again and keeps the ones that are still slower.
//
// This exists because no threshold separates a regression from an accident on a
// single timing, and that is measured rather than feared. The increase floor above
// removes the small-page accidents, whose increases are bounded by how long the
// page takes at all. It cannot remove the large-page ones: with a factor of 2, a
// 10ms floor and a 100ms increase floor, a run against a reference taken under a
// different machine load still offered three candidates on pages of 150ms and up,
// and re-measuring those three by hand gave 0.98x, 0.98x and 1.01x. Their
// increases cleared every threshold the real regression cleared. Nothing computed
// from one pair of numbers can tell the two apart; drawing the page again can.
//
// draw is a seam so a test need not have poppler. It returns how long we took, and
// the minimum over tries is what is compared, because a minimum is the estimator
// that noise cannot inflate: an accident can only ever make a page look slower.
//
// A page nothing could draw again is kept rather than dropped. An unread answer
// must not read as a negative.
// factor and slowerBy are the SAME bars that made these rows candidates, and a
// re-measurement has to clear them again. Asking only whether the page is still a
// little slower is not enough, and this is not hypothetical: at render v0.58.0 the
// check drew a page again, got 358 ms against the reference's 350 ms, and marked it
// CONFIRMED -- 1.02x, reported under a headline that said "at least 2.0x slower".
// A row that comes back at 1.02x is noise by the same rule that selected it.
func confirm(rs []slower, tries int, factor float64, slowerBy time.Duration, draw func(path string, page int) (time.Duration, bool)) []slower {
	// No guard on tries: at zero the loop below does not run, the row keeps the
	// reading it came in with, and nothing is drawn. An early return here would
	// be a branch no test could tell from its absence, which is how a guard
	// comes to be trusted without ever having been exercised.
	out := make([]slower, 0, len(rs))
	for _, r := range rs {
		best := time.Duration(0)
		any := false
		for i := 0; i < tries; i++ {
			d, ok := draw(r.path, r.page)
			if !ok || d <= 0 {
				continue
			}
			if !any || d < best {
				best, any = d, true
			}
		}
		if !any {
			out = append(out, r)
			continue
		}
		r.is = best
		r.factor = float64(best) / float64(r.was)
		r.confirmed = r.factor >= factor && best-r.was >= slowerBy
		r.dropped = !r.confirmed
		out = append(out, r)
	}
	return out
}
