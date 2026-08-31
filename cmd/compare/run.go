package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-pdfkit/conformance/compare"
	"github.com/go-pdfkit/conformance/corpus"
	"github.com/go-pdfkit/conformance/internal/poppler"
)

// compareOne is a variable so a test can judge without poppler installed.
var compareOne = compare.Compare

// run judges a corpus, one population at a time.
func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dir := fs.String("dir", "", "the corpus directory")
	only := fs.String("only", "", "judge just this population")
	pages := fs.Int("pages", 1, "pages of each document to judge")
	dpi := fs.Float64("dpi", 72, "what to ask both renderers for")
	budget := fs.Duration("budget", 20*time.Second, "how long a page may be drawn for")
	slow := fs.Duration("slow", 20*time.Second, "report pages we took longer than this on")
	judgeTimeout := fs.Duration("timeout", poppler.Timeout, "how long the judge may take on one page before it is called a hang")
	limit := fs.Int("limit", 0, "judge no more than this many documents per population")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dir == "" {
		fmt.Fprintln(errOut, "compare: -dir is needed")
		return 2
	}
	// The judge is bounded rather than left to the caller's patience, because
	// pdftoppm hangs on at least one document of this corpus and a hang looks
	// exactly like a slow run; see conformance#21.
	poppler.Timeout = *judgeTimeout
	entries, err := corpus.Read(*dir)
	if err != nil {
		fmt.Fprintf(errOut, "compare: %v\n", err)
		return 1
	}
	groups := map[string][]string{}
	for _, e := range entries {
		if *only == "" || e.Origin == *only {
			groups[e.Origin] = append(groups[e.Origin], filepath.Join(*dir, e.Path))
		}
	}
	if len(groups) == 0 {
		fmt.Fprintf(errOut, "compare: %s holds no population named %q\n", *dir, *only)
		return 1
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		paths := groups[name]
		if *limit > 0 && len(paths) > *limit {
			paths = paths[:*limit]
		}
		var rs []compare.Result
		for _, p := range paths {
			rs = append(rs, compareOne(p, compare.Options{
				DPI: *dpi, MaxDuration: *budget, Pages: *pages})...)
		}
		report(out, name, compare.Summarise(rs, *slow))
	}
	return 0
}

// report prints one population's verdict.
func report(out io.Writer, name string, s compare.Summary) {
	fmt.Fprintf(out, "%s\t%d compared\t%d not\n", name, s.Compared, s.NotCompared)
	for reason, n := range s.Notes {
		fmt.Fprintf(out, "\t%5d  %s\n", n, reason)
	}
	if s.Compared > 0 {
		fmt.Fprintf(out, "\tpixels differing: median %.4f  p90 %.4f  p99 %.4f  worst %.4f\n",
			s.Median, s.P90, s.P99, s.Max)
		fmt.Fprintf(out, "\tunder 1%% %d  under 2%% %d  under 5%% %d  under 10%% %d\n",
			s.Under[0.01], s.Under[0.02], s.Under[0.05], s.Under[0.10])
	}
	fmt.Fprintf(out, "\tslowest page %v, %d over the threshold\n", s.Slowest.Round(time.Millisecond), s.Over)
	for _, r := range s.Slow {
		fmt.Fprintf(out, "\t%12v  %s page %d\n",
			r.Ours.Round(time.Millisecond), filepath.Base(r.Path), r.Page)
	}
	// Every hang is named, and none is summarised away: a page the judge
	// would not answer about is not a page that disagreed, and a reader who
	// cannot tell those apart has a report that misleads.
	for _, r := range s.Hung {
		fmt.Fprintf(out, "\thung  %s  %s page %d\n", r.Tool, r.Path, r.Page)
	}
}
