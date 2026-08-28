package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-pdfkit/conformance/corpus"
	"github.com/go-pdfkit/conformance/survey"
)

// run groups a corpus by population and prints what each is made of.
func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("survey", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dir := fs.String("dir", "", "the corpus directory")
	only := fs.String("only", "", "survey just this population")
	pages := fs.Int("pages", 0, "look at no more than this many pages of each document (0 = all)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dir == "" {
		fmt.Fprintln(errOut, "survey: -dir is needed")
		return 2
	}

	groups, err := populations(*dir)
	if err != nil {
		fmt.Fprintf(errOut, "survey: %v\n", err)
		return 1
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		if *only == "" || name == *only {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		fmt.Fprintf(errOut, "survey: %s holds no documents\n", *dir)
		return 1
	}
	for _, name := range names {
		c := survey.Survey(groups[name], *pages)
		report(out, name, c)
	}
	return 0
}

// report prints one population's counts.
func report(out io.Writer, name string, c survey.Counts) {
	fmt.Fprintf(out, "%s\t%d files\t%d opened\t%d pages\n", name, c.Files, c.Documents, c.Pages)
	for _, r := range c.Reasons() {
		fmt.Fprintf(out, "\trefused: %5d  %s\n", c.Refused[r], r)
	}
	for _, f := range c.Filters() {
		share := 0.0
		if c.Documents > 0 {
			share = 100 * float64(c.UsedBy[f]) / float64(c.Documents)
		}
		fmt.Fprintf(out, "\t%-16s %5d documents (%5.1f%%)  %7d images  %5d pages blank without it\n",
			f, c.UsedBy[f], share, c.Images[f], c.BlankWithout[f])
	}
}

// populations groups a corpus's documents. The manifest is the authority; a
// directory without one is grouped by its top-level directories, so a corpus
// somebody assembled by hand can still be surveyed.
func populations(dir string) (map[string][]string, error) {
	out := map[string][]string{}
	entries, err := corpus.Read(dir)
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 {
		for _, e := range entries {
			out[e.Origin] = append(out[e.Origin], filepath.Join(dir, e.Path))
		}
		return out, nil
	}
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			return err
		}
		// Rel cannot fail here: WalkDir hands back paths under the directory
		// it was given, which is the one being subtracted.
		rel, _ := filepath.Rel(dir, path)
		group := filepath.Dir(rel)
		if group == "." {
			group = filepath.Base(dir)
		}
		out[group] = append(out[group], path)
		return nil
	})
	return out, err
}
