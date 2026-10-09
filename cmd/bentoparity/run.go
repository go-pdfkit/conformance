package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/go-pdfkit/conformance/parity"
)

// table is the mapping being measured. It is a variable so a test can measure
// a world small enough to verify end to end, rather than needing the whole
// fleet checked out.
var table = parity.BentoPDF

// smallTable is that world: one claim that holds, over a two-tool list.
var smallTable = []parity.Claim{{Tool: "merge-pdf", Claim: "verb:merge"}}

// helpOf runs a command's help and hands back what it printed. It is a
// variable so a test can measure without a binary on disk.
var helpOf = func(bin string) (io.Reader, error) {
	out, err := exec.Command(bin, "--help").CombinedOutput()
	// ⛔ The error is DELIBERATELY not returned. A command that prints its
	// usage and exits non-zero is the normal shape of "--help", and refusing
	// it would refuse every tool written that way. What matters is whether
	// verbs came back, and parity.Verbs says so.
	_ = err
	if len(out) == 0 {
		return nil, fmt.Errorf("%s printed nothing at all", bin)
	}
	return bytes.NewReader(out), nil
}

func run(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("bentoparity", flag.ContinueOnError)
	fs.SetOutput(errw)
	tools := fs.String("tools", "", "BentoPDF's docs/tools directory")
	bin := fs.String("pdfops", "", "a compiled pdfops binary, for the verbs")
	src := fs.String("src", "", "a directory holding the repositories to check claims against")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	for _, f := range []struct{ name, v string }{
		{"-tools", *tools}, {"-pdfops", *bin}, {"-src", *src},
	} {
		if f.v == "" {
			fmt.Fprintf(errw, "bentoparity: %s is required\n", f.name)
			return 2
		}
	}

	// "index" is BentoPDF's own contents page, not a tool.
	list, err := parity.ReadTools(os.DirFS(*tools), "index")
	if err != nil {
		fmt.Fprintln(errw, "bentoparity:", err)
		return 1
	}
	help, err := helpOf(*bin)
	if err != nil {
		fmt.Fprintln(errw, "bentoparity:", err)
		return 1
	}
	verbs, err := parity.Verbs(help)
	if err != nil {
		fmt.Fprintln(errw, "bentoparity:", err)
		return 1
	}
	rep, err := parity.Measure(list, table, verbs, os.DirFS(*src))
	if err != nil {
		fmt.Fprintln(errw, "bentoparity:", err)
		fmt.Fprintln(errw, "no figure reported: an unverified claim is not a capability")
		return 1
	}

	fmt.Fprintf(out, "BentoPDF documents %d tools; pdfops carries %d verbs\n\n", len(rep.Tools), len(verbs))
	fmt.Fprintln(out, "not covered:")
	for i, t := range rep.Uncovered {
		fmt.Fprintf(out, "  %-26s", t)
		if i%3 == 2 {
			fmt.Fprintln(out)
		}
	}
	fmt.Fprintf(out, "\n\ncovered      %3d\nnot covered  %3d\nparity       %.1f%% (%d/%d), every claim verified\n",
		len(rep.Covered), len(rep.Uncovered), rep.Percent(), len(rep.Covered), len(rep.Tools))
	return 0
}
