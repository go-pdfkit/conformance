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
	limit := fs.Int("limit", 0, "judge no more than this many documents per population")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dir == "" {
		fmt.Fprintln(errOut, "compare: -dir is needed")
		return 2
	}
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
}
