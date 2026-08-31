package main

import (
	"bytes"
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
			Path: filepath.Join("alpha", name), Origin: "alpha", Source: "u", SHA256: "x"})
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
