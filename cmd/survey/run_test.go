package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-pdfkit/conformance/corpus"
	"github.com/go-pdfkit/conformance/survey"
)

// corpusOf writes a tiny corpus: a manifest naming two populations, and a file
// for each. The files need not be readable PDFs — a refusal is a result.
func corpusOf(t *testing.T, withManifest bool) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "one.pdf"), []byte("not a pdf"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if withManifest {
		if err := corpus.Write(dir, []corpus.Entry{
			{Path: "alpha/one.pdf", Origin: "alpha", Source: "u", SHA256: "x"},
			{Path: "beta/one.pdf", Origin: "beta", Source: "u", SHA256: "x"},
		}); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunReportsEachPopulationSeparately(t *testing.T) {
	// A prevalence is per population or it is not a prevalence, so the report
	// is grouped and never totalled across them.
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", corpusOf(t, true)}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "alpha\t1 files") || !strings.Contains(got, "beta\t1 files") {
		t.Errorf("got %q", got)
	}
	// The refusal is reported with its reason, so nobody mistakes a file that
	// is not a PDF for a defect.
	if !strings.Contains(got, "refused:") {
		t.Errorf("the refusal is not reported: %q", got)
	}
}

func TestRunCanBeAskedForOnePopulation(t *testing.T) {
	var out, errOut bytes.Buffer
	run([]string{"-dir", corpusOf(t, true), "-only", "beta"}, &out, &errOut)
	if strings.Contains(out.String(), "alpha") || !strings.Contains(out.String(), "beta") {
		t.Errorf("got %q", out.String())
	}
}

func TestACorpusWithNoManifestIsGroupedByItsDirectories(t *testing.T) {
	// A corpus somebody assembled by hand is still surveyable.
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", corpusOf(t, false)}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "alpha") || !strings.Contains(out.String(), "beta") {
		t.Errorf("got %q", out.String())
	}
}

func TestASingleDirectoryOfFilesIsOnePopulation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", dir}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), filepath.Base(dir)) {
		t.Errorf("got %q", out.String())
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
		{"a directory that is not there", func(t *testing.T) []string {
			return []string{"-dir", filepath.Join(t.TempDir(), "absent")}
		}, 1},
		{"a population that is not in it", func(t *testing.T) []string {
			return []string{"-dir", corpusOf(t, true), "-only", "gamma"}
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
			var out, errOut bytes.Buffer
			if code := run(tc.args(t), &out, &errOut); code != tc.want {
				t.Errorf("exit %d, want %d: %s", code, tc.want, errOut.String())
			}
		})
	}
}

func TestOnlySoManyPagesCanBeAskedFor(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", corpusOf(t, true), "-pages", "1"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestMainCallsRun(t *testing.T) {
	oldExit, oldArgs := osExit, os.Args
	defer func() { osExit, os.Args = oldExit, oldArgs }()
	got := -1
	osExit = func(code int) { got = code }
	os.Args = []string{"survey"}
	main()
	if got != 2 {
		t.Errorf("main exited %d, want 2", got)
	}
}

func TestAPopulationWithNothingInItStillReportsItself(t *testing.T) {
	// Nothing opened, so the share cannot be a percentage of anything — and
	// dividing by it would be the kind of arithmetic that produces NaN in a
	// report somebody then quotes.
	var out bytes.Buffer
	report(&out, "empty", survey.Counts{
		Files: 3, UsedBy: map[string]int{"JPXDecode": 0},
		Images: map[string]int{}, BlankWithout: map[string]int{}, Refused: map[string]int{},
	})
	got := out.String()
	if !strings.Contains(got, "empty\t3 files") || !strings.Contains(got, "0.0%") {
		t.Errorf("got %q", got)
	}
}

func TestTheShareIsOutOfTheDocumentsThatOpened(t *testing.T) {
	var out bytes.Buffer
	report(&out, "p", survey.Counts{
		Files: 4, Documents: 4, Pages: 8,
		UsedBy: map[string]int{"JPXDecode": 3}, Images: map[string]int{"JPXDecode": 9},
		BlankWithout: map[string]int{"JPXDecode": 2}, Refused: map[string]int{},
	})
	// Three of four is seventy-five per cent, and the pages that would be
	// blank are reported beside it because they are the harm, not the presence.
	if !strings.Contains(out.String(), "75.0%") || !strings.Contains(out.String(), "2 pages blank") {
		t.Errorf("got %q", out.String())
	}
}

func TestACorpusItCannotWalkIsReported(t *testing.T) {
	dir := t.TempDir()
	shut := filepath.Join(dir, "shut")
	if err := os.Mkdir(shut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shut, "one.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shut, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(shut, 0o755) })

	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", dir}, &out, &errOut); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}

func TestTheMasksAreReportedBesideTheFilters(t *testing.T) {
	var out bytes.Buffer
	report(&out, "scans", survey.Counts{
		Files: 1, Documents: 1, Pages: 1,
		UsedBy: map[string]int{"JPXDecode": 1}, Images: map[string]int{"JPXDecode": 1},
		BlankWithout: map[string]int{}, Refused: map[string]int{},
		MaskedBy: map[string]int{"JBIG2Decode": 4089},
	})
	if !strings.Contains(out.String(), "4089 images are SHAPED BY") {
		t.Errorf("the mask count is not reported: %q", out.String())
	}
}
