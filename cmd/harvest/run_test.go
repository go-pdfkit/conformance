package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/go-pdfkit/conformance/corpus"
)

func TestRunPrintsHowBigEachPopulationIs(t *testing.T) {
	// The count per origin is the number any prevalence taken from the corpus
	// is divided by, so it is what the command reports.
	harvest = func(context.Context, *corpus.Archive, corpus.Plan) ([]corpus.Entry, error) {
		return []corpus.Entry{
			{Origin: "scans"}, {Origin: "scans"}, {Origin: "forms"},
		}, nil
	}
	defer func() { harvest = corpus.Harvest }()

	var out, errOut bytes.Buffer
	code := run([]string{"-dir", "d", "-origin", "scans", "-query", "q"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	// Sorted, so two runs print the same thing.
	if out.String() != "forms\t1\nscans\t2\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestRunSaysWhatIsMissing(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no directory", []string{"-origin", "o", "-query", "q"}},
		{"no origin", []string{"-dir", "d", "-query", "q"}},
		{"no query", []string{"-dir", "d", "-origin", "o"}},
		{"a flag that is not one", []string{"-nonsense"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run(tc.args, &out, &errOut); code != 2 {
				t.Errorf("exit %d, want 2", code)
			}
		})
	}
}

func TestRunReportsAHarvestThatFailed(t *testing.T) {
	harvest = func(context.Context, *corpus.Archive, corpus.Plan) ([]corpus.Entry, error) {
		return nil, errors.New("the archive said no")
	}
	defer func() { harvest = corpus.Harvest }()

	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", "d", "-origin", "o", "-query", "q"}, &out, &errOut); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "the archive said no") {
		t.Errorf("the reason is not in %q", errOut.String())
	}
}

func TestTheLogReachesTheCaller(t *testing.T) {
	harvest = func(_ context.Context, _ *corpus.Archive, p corpus.Plan) ([]corpus.Entry, error) {
		p.Log("fetched %d", 7)
		return nil, nil
	}
	defer func() { harvest = corpus.Harvest }()

	var out, errOut bytes.Buffer
	run([]string{"-dir", "d", "-origin", "o", "-query", "q"}, &out, &errOut)
	if !strings.Contains(errOut.String(), "fetched 7") {
		t.Errorf("the log did not reach the caller: %q", errOut.String())
	}
}

func TestMainCallsRun(t *testing.T) {
	oldExit, oldArgs := osExit, os.Args
	defer func() { osExit, os.Args = oldExit, oldArgs }()
	got := -1
	osExit = func(code int) { got = code }
	os.Args = []string{"harvest"} // no flags, so run refuses
	main()
	if got != 2 {
		t.Errorf("main exited %d, want 2", got)
	}
}
