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
	if err := fs.Parse(args); err != nil {
		return 2
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
