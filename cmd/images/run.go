package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/go-pdfkit/conformance/corpus"
	"github.com/go-pdfkit/conformance/images"
)

// judgeOne is a variable so a test can judge without poppler installed.
var judgeOne = images.Judge

// run judges a corpus, one population at a time.
func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("images", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dir := fs.String("dir", "", "the corpus directory")
	only := fs.String("only", "", "judge just this population")
	pages := fs.Int("pages", 1, "pages of each document to take pictures from")
	limit := fs.Int("limit", 0, "judge no more than this many documents per population")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dir == "" {
		fmt.Fprintln(errOut, "images: -dir is needed")
		return 2
	}
	entries, err := corpus.Read(*dir)
	if err != nil {
		fmt.Fprintf(errOut, "images: %v\n", err)
		return 1
	}
	groups := map[string][]string{}
	for _, e := range entries {
		if *only == "" || e.Origin == *only {
			groups[e.Origin] = append(groups[e.Origin], filepath.Join(*dir, e.Path))
		}
	}
	if len(groups) == 0 {
		fmt.Fprintf(errOut, "images: %s holds no population named %q\n", *dir, *only)
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
		var rs []images.Result
		for _, p := range paths {
			rs = append(rs, judgeOne(p, images.Options{Pages: *pages})...)
		}
		fmt.Fprintf(out, "%s\n", name)
		fmt.Fprint(out, images.Report(images.Tally(rs)))
	}
	return 0
}
