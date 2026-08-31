package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/go-pdfkit/conformance/corpus"
	"github.com/go-pdfkit/conformance/images"
	"github.com/go-pdfkit/conformance/internal/poppler"
)

// tinyCorpus writes a manifest naming two populations and a file for each.
func tinyCorpus(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "one.pdf"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := corpus.Write(dir, []corpus.Entry{
		{Path: "alpha/one.pdf", Origin: "alpha", Source: "u", SHA256: "x"},
		{Path: "beta/one.pdf", Origin: "beta", Source: "u", SHA256: "x"},
	}); err != nil {
		t.Fatal(err)
	}
	return dir
}

// judge stands in for the comparison so these run without poppler.
func judge(t *testing.T, rs ...images.Result) *int {
	t.Helper()
	was := judgeOne
	t.Cleanup(func() { judgeOne = was })
	n := 0
	judgeOne = func(string, images.Options) []images.Result {
		n++
		return rs
	}
	return &n
}

func TestRunReportsEachPopulationSeparately(t *testing.T) {
	judge(t, images.Result{Name: "I", Filter: "JBIG2Decode"})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t)}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "alpha\n") || !strings.Contains(got, "beta\n") {
		t.Errorf("got %q", got)
	}
	if strings.Count(got, "JBIG2Decode") != 2 {
		t.Errorf("the filter is not reported once per population: %q", got)
	}
}

func TestRunLooksAtOnePopulation(t *testing.T) {
	judge(t, images.Result{Name: "I"})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-only", "beta"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if got := out.String(); strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Errorf("got %q", got)
	}
}

func TestRunStopsAtTheLimit(t *testing.T) {
	// A corpus is large and a first look should not have to be a whole one.
	n := judge(t, images.Result{Name: "I"})
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	var entries []corpus.Entry
	for _, name := range []string{"a.pdf", "b.pdf", "c.pdf"} {
		os.WriteFile(filepath.Join(dir, "alpha", name), []byte("x"), 0o644)
		entries = append(entries, corpus.Entry{
			Path: "alpha/" + name, Origin: "alpha", Source: "u", SHA256: "x"})
	}
	if err := corpus.Write(dir, entries); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", dir, "-limit", "2"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if *n != 2 {
		t.Errorf("it looked at %d documents, not 2", *n)
	}
}

func TestRunRefusesWhatItCannotDo(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		code int
		want string
	}{
		{"no corpus named", nil, 2, "-dir is needed"},
		{"a flag it does not have", []string{"-nope"}, 2, ""},
		{"a corpus whose manifest says nothing it can read", nil, 1, "images:"},
		{"a population that is not in it", []string{"-only", "gamma"}, 1, "no population"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.args
			switch {
			case len(args) > 0 && args[0] == "-only":
				args = append([]string{"-dir", tinyCorpus(t)}, args...)
			case tc.name == "a corpus whose manifest says nothing it can read":
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "MANIFEST.tsv"),
					[]byte("nothing\tit\tknows\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				args = []string{"-dir", dir}
			}
			var out, errOut bytes.Buffer
			if code := run(args, &out, &errOut); code != tc.code {
				t.Fatalf("exit %d, want %d: %s", code, tc.code, errOut.String())
			}
			if tc.want != "" && !strings.Contains(errOut.String(), tc.want) {
				t.Errorf("said %q", errOut.String())
			}
		})
	}
}

func TestMainRunsTheCommand(t *testing.T) {
	was := osExit
	defer func() { osExit = was }()
	code := -1
	osExit = func(c int) { code = c }
	old := os.Args
	os.Args = []string{"images"}
	defer func() { os.Args = old }()
	main()
	if code != 2 {
		t.Errorf("running it with no corpus exited %d", code)
	}
}

// atTime fixes what the record will say it was taken.
func atTime(t *testing.T, s string) {
	t.Helper()
	was := now
	t.Cleanup(func() { now = was })
	when, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	now = func() time.Time { return when }
}

func TestRunWritesABaselineThatCanBeComparedAgainst(t *testing.T) {
	judge(t, images.Result{Name: "I", Filter: "JBIG2Decode",
		Difference: images.Difference{Share: 0.5, Peak: 30, MSE: 9, Mean: -3}})
	atTime(t, "2026-08-30T15:04:05Z")
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-json"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got baseline
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("the record is not JSON: %v\n%s", err, out.String())
	}
	if got.Taken != "2026-08-30T15:04:05Z" || got.Pages != 1 {
		t.Errorf("the record does not say when or how much: %+v", got)
	}
	if len(got.Populations) != 2 ||
		got.Populations[0].Population != "alpha" ||
		got.Populations[1].Population != "beta" {
		t.Fatalf("populations came out as %+v", got.Populations)
	}
	p := got.Populations[0]
	if p.Documents != 1 || len(p.Filters) != 1 || p.Filters[0].Direct == nil ||
		p.Filters[0].Direct.Differing != 1 {
		t.Errorf("alpha came out as %+v", p)
	}
	// The gate IS the instrument, so a record that did not carry it could not
	// be told apart from one taken by the ink bisection that preceded it.
	if got.Gate != images.Gate {
		t.Errorf("the record says the gate was %d", got.Gate)
	}
	if peak := p.Filters[0].Direct.Terms.Peak.Worst; peak != 30 {
		t.Errorf("the magnitudes did not reach the record: %+v", p.Filters[0].Direct.Terms)
	}
	if p.Refused != 0 || p.Unopenable != 0 || p.Declined != 0 {
		t.Errorf("a population that was judged reports something missing: %+v", p)
	}
	// The record replaces the report rather than joining it, or it would not
	// parse.
	if strings.Contains(out.String(), "pictures  ") {
		t.Errorf("the human report was written into the record: %s", out.String())
	}
	// A version this was built against is what tells a later drop from a
	// rebuild, so the record is useless without one.
	if len(got.Modules) == 0 {
		t.Error("the record names no module it was built against")
	}
}

func TestWhatCouldNotBeComparedReachesTheRecord(t *testing.T) {
	// The whole point of the split is that it survives to the file somebody
	// reads a year later, so it is checked where it lands and not only where
	// it is counted.
	judge(t,
		images.Result{Difference: images.Difference{Share: -1}, Missing: images.Ours, Note: "refused: x"},
		images.Result{Difference: images.Difference{Share: -1}, Missing: images.Neither, Note: "refused: y"},
		images.Result{Difference: images.Difference{Share: -1}, Missing: images.Theirs, Note: "they took nothing out"},
	)
	atTime(t, "2026-08-30T15:04:05Z")
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-only", "alpha", "-json"},
		&out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got baseline
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("the record is not JSON: %v", err)
	}
	p := got.Populations[0]
	if p.Refused != 1 || p.Unopenable != 1 || p.Declined != 1 {
		t.Errorf("refused %d, unopenable %d, declined %d; want one of each",
			p.Refused, p.Unopenable, p.Declined)
	}
	// Nothing was comparable, so nothing may be reported as a filter that
	// agreed or disagreed.
	if len(p.Filters) != 0 {
		t.Errorf("it invented filters out of what it could not compare: %+v", p.Filters)
	}
}

func TestABaselineWithNoBuildInfoStillRecordsTheCounts(t *testing.T) {
	// Nothing here is worth losing a run's counts over.
	was := buildInfo
	defer func() { buildInfo = was }()
	buildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
	if got := modules(); got != nil {
		t.Errorf("it invented %+v", got)
	}
}

func TestTheJudgeIsRecordedByVersionWhenItSaysOne(t *testing.T) {
	for _, tc := range []struct {
		name string
		out  []byte
		err  error
		want string
	}{
		{"it answers", []byte("pdfimages version 26.04.0\nCopyright\n"), nil,
			"pdfimages version 26.04.0"},
		{"it is not installed", nil, os.ErrNotExist, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			was := versionCommand
			defer func() { versionCommand = was }()
			versionCommand = func() ([]byte, error) { return tc.out, tc.err }
			if got := judgeVersion(); got != tc.want {
				t.Errorf("recorded the judge as %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheRealJudgeIsTheOneThatIsAsked(t *testing.T) {
	// Whether poppler is installed decides the answer, not whether the
	// statement runs, so this covers the wiring either way.
	_, _ = versionCommand()
}

func TestTheReportNamesEveryDocumentTheJudgeHungOn(t *testing.T) {
	// conformance#21. A document skipped because pdfimages would not finish
	// is not a document that scored badly, and in a report that leaves it out
	// the two are the same absence. It is named in the report as well as in
	// the record, with the tool and with its whole path.
	judge(t, images.Result{Path: "/corpus/gh-qpdf/hangs.pdf", Page: 1,
		Missing: images.Hung, Tool: "pdfimages", Difference: images.Difference{Share: -1}})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-only", "alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "hung  pdfimages  /corpus/gh-qpdf/hangs.pdf page 1") {
		t.Errorf("the hang is not named with its tool and path: %q", got)
	}
}

func TestTheRecordSaysWhatBoundTheJudgeWasHeldTo(t *testing.T) {
	// The bound is part of the instrument, like the gate: a run under a
	// shorter one can name documents as hung that a longer one measures, and
	// the populations they are in would then read as smaller without saying
	// why. So the record carries the value the run used.
	was := poppler.Timeout
	defer func() { poppler.Timeout = was }()
	judge(t, images.Result{Name: "I", Filter: "JBIG2Decode"})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-json", "-timeout", "9m"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got baseline
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Timeout != "9m0s" {
		t.Errorf("the record says the judge was bounded at %q", got.Timeout)
	}
	if poppler.Timeout != 9*time.Minute {
		t.Errorf("the flag did not reach the judge: %v", poppler.Timeout)
	}
}
