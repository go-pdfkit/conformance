package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"

	"github.com/go-pdfkit/conformance/corpus"
)

// harvest is a variable so a test can watch what run does with what it returns
// without going near the network.
var harvest = corpus.Harvest

// run parses the arguments, harvests, and prints how big each population now
// is — which is the number any prevalence taken from this corpus is divided by.
func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("harvest", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dir := fs.String("dir", "", "the corpus directory")
	origin := fs.String("origin", "", "the population these documents belong to")
	query := fs.String("query", "", "what to ask the archive for")
	want := fs.Int("want", 100, "how many documents this origin should end up with")
	maxBytes := fs.Int64("max-bytes", 40<<20, "refuse a document larger than this")
	workers := fs.Int("workers", 4, "how many to fetch at once")
	check := fs.Bool("check", false,
		"fetch nothing: report every way the corpus and its manifest disagree")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *check {
		return checkCorpus(*dir, out, errOut)
	}
	if *dir == "" || *origin == "" || *query == "" {
		fmt.Fprintln(errOut, "harvest: -dir, -origin and -query are all needed")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	entries, err := harvest(ctx, &corpus.Archive{}, corpus.Plan{
		Dir: *dir, Origin: *origin, Query: *query,
		Want: *want, MaxBytes: *maxBytes, Workers: *workers,
		Log: func(f string, a ...any) { fmt.Fprintf(errOut, f+"\n", a...) },
	})
	if err != nil {
		fmt.Fprintf(errOut, "harvest: %v\n", err)
		return 1
	}
	counts := corpus.Origins(entries)
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(out, "%s\t%d\n", name, counts[name])
	}
	return 0
}

// checkCorpus reports where the corpus and the manifest disagree, and is in
// harvest because harvest is what writes the manifest.
//
// The package comment of corpus says the manifest is what makes a corpus a
// measurement rather than a pile, "so a figure quoted from it can be reproduced
// and a file that changed underneath can be noticed". Nothing noticed: the hash
// was written and never read back. Four files of the scans corpus are on disk
// and in no row, three of them documents our reader refuses, which is why one
// instrument counts 63 refusals and another 66.
//
// It exits NON-ZERO when a corpus disagrees with its manifest, so it can be the
// first line of a measuring script rather than something to remember to run.
func checkCorpus(dir string, out, errOut io.Writer) int {
	if dir == "" {
		fmt.Fprintln(errOut, "harvest: -check needs -dir")
		return 2
	}
	problems, err := corpus.Check(dir)
	if err != nil {
		fmt.Fprintf(errOut, "harvest: %v\n", err)
		return 1
	}
	for _, p := range problems {
		fmt.Fprintf(out, "%s\n", p)
	}
	if len(problems) == 0 {
		fmt.Fprintf(out, "the corpus and its manifest agree\n")
		return 0
	}
	fmt.Fprintf(out, "%d disagreement(s)\n", len(problems))
	return 1
}
