package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-pdfkit/conformance/compare"
	"github.com/go-pdfkit/conformance/corpus"
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
func judge(t *testing.T, rs ...compare.Result) {
	t.Helper()
	was := compareOne
	t.Cleanup(func() { compareOne = was })
	compareOne = func(string, compare.Options) []compare.Result { return rs }
}

func TestRunJudgesEachPopulationSeparately(t *testing.T) {
	judge(t, compare.Result{Share: 0.004, Ours: time.Second})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t)}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "alpha\t1 compared") || !strings.Contains(got, "beta\t1 compared") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "median") || !strings.Contains(got, "slowest page") {
		t.Errorf("the distribution is not reported: %q", got)
	}
}

func TestRunReportsWhatCouldNotBeJudged(t *testing.T) {
	judge(t, compare.Result{Share: -1, Note: "we drew nothing"})
	var out, errOut bytes.Buffer
	run([]string{"-dir", tinyCorpus(t), "-only", "alpha"}, &out, &errOut)
	if !strings.Contains(out.String(), "we drew nothing") {
		t.Errorf("got %q", out.String())
	}
	// And with nothing compared, no distribution is invented.
	if strings.Contains(out.String(), "median") {
		t.Errorf("a distribution was printed for nothing: %q", out.String())
	}
}

func TestRunSaysWhatIsWrong(t *testing.T) {
	for _, tc := range []struct {
		name string
		args func(t *testing.T) []string
		want int
	}{
		{"no directory", func(*testing.T) []string { return nil }, 2},
		{"a flag that is not one", func(*testing.T) []string { return []string{"-nonsense"} }, 2},
		{"a population that is not in it", func(t *testing.T) []string {
			return []string{"-dir", tinyCorpus(t), "-only", "gamma"}
		}, 1},
		{"a manifest that will not parse", func(t *testing.T) []string {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, corpus.ManifestName),
				[]byte("nothing\tuseful\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return []string{"-dir", dir}
		}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			judge(t)
			var out, errOut bytes.Buffer
			if code := run(tc.args(t), &out, &errOut); code != tc.want {
				t.Errorf("exit %d, want %d", code, tc.want)
			}
		})
	}
}

func TestOnlySoManyDocumentsCanBeAskedFor(t *testing.T) {
	// A population of two, judged with a limit of one: a corpus of a hundred
	// thousand is surveyed by sampling it, not by waiting for it.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	var entries []corpus.Entry
	for _, name := range []string{"one.pdf", "two.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, "alpha", name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, corpus.Entry{
			// A slash, not filepath.Join: this is what a manifest holds, and a
			// manifest holds the same text on every machine. The line above is
			// a real path on this disk and rightly uses filepath.
			Path: "alpha/" + name, Origin: "alpha", Source: "u", SHA256: "x"})
	}
	if err := corpus.Write(dir, entries); err != nil {
		t.Fatal(err)
	}
	seen := 0
	was := compareOne
	defer func() { compareOne = was }()
	compareOne = func(string, compare.Options) []compare.Result {
		seen++
		return []compare.Result{{Share: 0.1}}
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", dir, "-limit", "1"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if seen != 1 {
		t.Errorf("judged %d documents with a limit of 1", seen)
	}
}

func TestMainCallsRun(t *testing.T) {
	oldExit, oldArgs := osExit, os.Args
	defer func() { osExit, os.Args = oldExit, oldArgs }()
	got := -1
	osExit = func(code int) { got = code }
	os.Args = []string{"compare"}
	main()
	if got != 2 {
		t.Errorf("main exited %d, want 2", got)
	}
}

func TestTheReportNamesTheSlowPages(t *testing.T) {
	judge(t, compare.Result{Path: "/corpus/slow.pdf", Page: 3,
		Share: 0.004, Ours: 30 * time.Second})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-slow", "1s"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "slow.pdf page 3") {
		t.Errorf("the slow page is not named: %q", got)
	}
	if strings.Contains(got, "/corpus/") {
		t.Errorf("the whole path is in the way: %q", got)
	}
}

func TestTheReportNamesEveryPageTheJudgeHungOn(t *testing.T) {
	// conformance#21. A page the judge would not answer about is not a page
	// that disagreed, and the two are the same absence in a count — so the
	// document is named, with the tool, and with its whole path, because the
	// point of naming it is to go and look at it.
	judge(t, compare.Result{Path: "/corpus/gh-qpdf/hangs.pdf", Page: 1,
		Share: -1, Tool: "pdftoppm", Note: "hung: pdftoppm did not finish within 2m0s"})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t)}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "hung  pdftoppm  /corpus/gh-qpdf/hangs.pdf page 1") {
		t.Errorf("the hang is not named with its tool and path: %q", got)
	}
}

func TestTheBoundOnTheJudgeCanBeSaid(t *testing.T) {
	// A corpus of larger pages may want a larger bound, and a run that used
	// one has to be able to say so.
	was := poppler.Timeout
	defer func() { poppler.Timeout = was }()
	judge(t, compare.Result{Path: "a.pdf", Page: 1, Share: 0})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-timeout", "9m"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if poppler.Timeout != 9*time.Minute {
		t.Errorf("the judge is bounded at %v", poppler.Timeout)
	}
}

// TestTheReportSaysHowTheTwoRenderersTimed covers the block that prints the
// speed half of the comparison. It is printed at all because Result.Theirs was
// being measured and never reported: a claim about speed that this instrument
// cannot be run to check is not a claim anyone can act on.
func TestTheReportSaysHowTheTwoRenderersTimed(t *testing.T) {
	var out bytes.Buffer
	report(&out, "pop", compare.Summarise([]compare.Result{
		{Path: "/c/fast.pdf", Page: 1, Share: 0, Ours: ms(10), Theirs: ms(100)},
		{Path: "/c/slow.pdf", Page: 1, Share: 0, Ours: ms(300), Theirs: ms(100)},
	}, 0))
	got := out.String()
	for _, want := range []string{
		"2 pages both were timed on",
		"ours 310ms",
		"theirs 200ms",
		"faster on 1 of 2",
		"worst   3.00x SLOWER",
		"slow.pdf page 1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not say %q:\n%s", want, got)
		}
	}
}

// TestTheReportIsSilentWithNoTimings keeps a run that could time nothing from
// printing a ratio of zero over zero.
func TestTheReportIsSilentWithNoTimings(t *testing.T) {
	var out bytes.Buffer
	report(&out, "pop", compare.Summarise([]compare.Result{
		{Path: "/c/x.pdf", Page: 1, Share: -1, Note: "we drew nothing"},
	}, 0))
	if strings.Contains(out.String(), "both were timed on") {
		t.Errorf("it spoke about timings it does not have:\n%s", out.String())
	}
}

// TestPerPageTimingsAreWritten covers the file that makes the speed claim
// auditable. The summary gives medians; only the per-page rows let a reader
// correct for the judge being a subprocess, or check the medians at all.
func TestPerPageTimingsAreWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timings.tsv")
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-timings", path}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(b), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	if len(lines) < 2 {
		t.Fatalf("the file holds only %d line(s):\n%s", len(lines), b)
	}
	if lines[0] != "population\tdocument\tpage\tours_ns\ttheirs_ns\tshare\thung" {
		t.Errorf("header %q", lines[0])
	}
	for _, l := range lines[1:] {
		if n := len(strings.Split(l, "\t")); n != 7 {
			t.Errorf("row has %d fields, want 7: %q", n, l)
		}
	}
}

// TestATimingsFileThatCannotBeWrittenIsReported: a run that silently loses its
// own measurements is worse than one that stops.
func TestATimingsFileThatCannotBeWrittenIsReported(t *testing.T) {
	var out, errOut bytes.Buffer
	// A directory where the file has to go, which no operating system will let
	// os.Create open for writing.
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-dir", tinyCorpus(t), "-timings", blocked}, &out, &errOut); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "compare:") {
		t.Errorf("it did not say why: %q", errOut.String())
	}
}

// TestARunCanBeComparedWithAnEarlierOne covers the -against path end to end: a
// run writes its timings, and a second run reads them back and says nothing got
// slower. It is the second instrument the memo defect needed and did not have.
func TestARunCanBeComparedWithAnEarlierOne(t *testing.T) {
	dir := tinyCorpus(t)
	first := filepath.Join(t.TempDir(), "first.tsv")
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", dir, "-timings", first}, &out, &errOut); code != 0 {
		t.Fatalf("the first run exited %d: %s", code, errOut.String())
	}
	out.Reset()
	if code := run([]string{"-dir", dir, "-against", first}, &out, &errOut); code != 0 {
		t.Fatalf("the second run exited %d: %s", code, errOut.String())
	}
	// The same corpus against itself: the only honest verdict is that nothing
	// crossed the threshold.
	if !strings.Contains(out.String(), "no page is 2.0x slower") {
		t.Errorf("the comparison said:\n%s", out.String())
	}

	// A file that cannot be read stops the run rather than comparing against
	// nothing.
	out.Reset()
	errOut.Reset()
	if code := run([]string{"-dir", dir, "-against", filepath.Join(t.TempDir(), "absent.tsv")}, &out, &errOut); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "compare:") {
		t.Errorf("it did not say why: %q", errOut.String())
	}
}

// TestTheRunDrawsItsCandidatesAgain covers the confirmation seam end to end. The
// stub answers slowly while the corpus is walked and quickly when a candidate is
// drawn again, which is exactly the shape of the thirteen false positives that put
// this check here.
func TestTheRunDrawsItsCandidatesAgain(t *testing.T) {
	dir := tinyCorpus(t)
	ref := filepath.Join(t.TempDir(), "ref.tsv")
	// A reference in which both pages were fast. 7 fields, and the header line
	// is skipped by readTimings just as it is in a file the tool wrote.
	rows := "population\tdocument\tpage\tours\ttheirs\tshare\thung\n"
	for _, pop := range []string{"alpha", "beta"} {
		rows += fmt.Sprintf("%s\tone.pdf\t0\t%d\t%d\t%.6f\t-\n",
			pop, 100*time.Millisecond, 100*time.Millisecond, 0.001)
	}
	if err := os.WriteFile(ref, []byte(rows), 0o644); err != nil {
		t.Fatal(err)
	}

	// draws counts every call so the walk and the confirmation can answer
	// differently. The two documents of tinyCorpus make the walk two calls.
	for _, tc := range []struct {
		name          string
		redraw        time.Duration
		redrawPage    int
		wantSubstring string
	}{
		// Both documents of tinyCorpus regress, so the counts are 2.
		{"still slower is confirmed", 400 * time.Millisecond, 0, "2 page(s) at least"},
		{"faster is dropped", 90 * time.Millisecond, 0, "2 dropped by re-measurement (~), 0 confirmed"},
		// A confirmation that cannot find the page it asked for leaves the
		// candidate standing: an unread answer is not a negative.
		{"a page it cannot find is kept", 90 * time.Millisecond, 99, "2 page(s) at least"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			was := compareOne
			t.Cleanup(func() { compareOne = was })
			// Counted PER PATH rather than in total: the first time a document is
			// asked for it is the corpus walk, and every time after that is
			// something redrawing it. A total would break the moment anything else
			// in the run redraws a page, which is exactly what happened.
			seen := map[string]int{}
			draws := 0
			compareOne = func(p string, _ compare.Options) []compare.Result {
				draws++
				seen[p]++
				ours, page := 400*time.Millisecond, 0
				if seen[p] > 1 { // past the walk: this is a redraw
					ours, page = tc.redraw, tc.redrawPage
				}
				return []compare.Result{{
					Path: p, Page: page, Ours: ours,
					Theirs: 100 * time.Millisecond, Share: 0.001,
				}}
			}
			var out, errOut bytes.Buffer
			if code := run([]string{"-dir", dir, "-against", ref, "-confirm", "2"}, &out, &errOut); code != 0 {
				t.Fatalf("exit %d: %s", code, errOut.String())
			}
			if draws <= 2 {
				t.Errorf("compareOne was called %d times: nothing was drawn again", draws)
			}
			if !strings.Contains(out.String(), tc.wantSubstring) {
				t.Errorf("want %q in:\n%s", tc.wantSubstring, out.String())
			}
			// The mark is the verdict, so it is asserted rather than the prose.
			switch tc.name {
			case "still slower is confirmed":
				if n := strings.Count(out.String(), "\n! "); n != 2 {
					t.Errorf("%d rows marked confirmed, want 2:\n%s", n, out.String())
				}
			case "faster is dropped":
				if n := strings.Count(out.String(), "\n~ "); n != 2 {
					t.Errorf("%d rows marked noise, want 2:\n%s", n, out.String())
				}
			case "a page it cannot find is kept":
				if strings.Contains(out.String(), "\n! ") || strings.Contains(out.String(), "\n~ ") {
					t.Errorf("a candidate nothing drew again must carry no verdict:\n%s", out.String())
				}
			}
		})
	}
}
