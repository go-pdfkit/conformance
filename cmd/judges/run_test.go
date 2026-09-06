package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-pdfkit/conformance/internal/mdreport"
)

// A rig is a directory with a document, a pdfium_test that is a file, and a
// node directory with pdf.js installed — every check the harness makes for a
// judge's presence, satisfied without a single real one.
type rig struct{ dir, pdf, pdfium, node string }

func aRig(t *testing.T) rig {
	t.Helper()
	dir := t.TempDir()
	r := rig{dir: dir, pdf: filepath.Join(dir, "doc.pdf"),
		pdfium: filepath.Join(dir, "pdfium_test"), node: filepath.Join(dir, "judges")}
	os.WriteFile(r.pdf, []byte("%PDF-1.7\n"), 0o644)
	os.WriteFile(r.pdfium, []byte("#!/bin/sh\n"), 0o755)
	pkg := filepath.Join(r.node, "node_modules", "pdfjs-dist")
	os.MkdirAll(pkg, 0o755)
	os.WriteFile(filepath.Join(pkg, "package.json"), []byte(`{"version":"6.3.289"}`), 0o644)
	return r
}

func (r rig) args(more ...string) []string {
	return append([]string{"-pdfs", filepath.Join(r.dir, "*.pdf"), "-out", filepath.Join(r.dir, "out"),
		"-report", filepath.Join(r.dir, "JUDGES.md"), "-results", filepath.Join(r.dir, "judges.json"),
		"-pdfium", r.pdfium, "-nodedir", r.node}, more...)
}

func (r rig) report(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.dir, "JUDGES.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (r rig) results(t *testing.T) []fileResult {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.dir, "judges.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rs []fileResult
	if err := json.Unmarshal(b, &rs); err != nil {
		t.Fatal(err)
	}
	return rs
}

func TestRunAsksEveryReaderTheMachineHas(t *testing.T) {
	stubTools(t, everyReader, aMachine(t, 3, nil))
	r := aRig(t)
	var out, errOut bytes.Buffer
	if code := run(r.args(), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "1 PDFs judged") {
		t.Errorf("stdout %q", out.String())
	}
	got := r.report(t)
	for _, want := range []string{
		"Judges: qpdf, poppler, mupdf, gs, pdfium, pdfjs, quartz.",
		"pdf.js 6.3.289;",
		"| PDF | Bytes | qpdf | poppler | mupdf | gs | pdfium | pdfjs | quartz | consensus |",
		"| ✅ – · – · – | ✅ 3p · 1.000 · ref ⚠1 | ✅ 3p · 1.000 · Δ",
		"| ✅ – · 1.000 · Δ",  // gs: no page count, padding not counted
		"| ✅ 3p · 1.000 · Δ", // pdfium, in characters not bytes
		"(p1 only) |",        // quartz
		mdreport.Marker,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	rs := r.results(t)
	if len(rs) != 1 || len(rs[0].SampledPages) != 3 || rs[0].Bytes != 9 {
		t.Fatalf("%+v", rs)
	}
	// The sample was chosen once poppler said how many pages there are, and
	// every renderer after it drew all three.
	for _, v := range rs[0].Verdicts {
		switch v.Judge {
		case "poppler", "mupdf", "gs", "pdfium", "pdfjs":
			if len(v.Renders) != 3 || v.Ms < 0 {
				t.Errorf("%s drew %d pages", v.Judge, len(v.Renders))
			}
			if v.Judge != "poppler" && (v.WorstPct <= 0 || v.WorstPct >= 100) {
				t.Errorf("%s is Δ%v from poppler", v.Judge, v.WorstPct)
			}
		}
	}
	if rs[0].ConsensusMax <= 0 {
		t.Errorf("no consensus: %+v", rs[0])
	}
	// The progress went to stderr, judge by judge.
	if !strings.Contains(errOut.String(), "poppler  ok=true pages=3") || !strings.Contains(errOut.String(), "consensus max") {
		t.Errorf("stderr %q", errOut.String())
	}
}

func TestARerunKeepsTheAnalysis(t *testing.T) {
	stubTools(t, everyReader, aMachine(t, 1, nil))
	r := aRig(t)
	var out, errOut bytes.Buffer
	if code := run(r.args(), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	path := filepath.Join(r.dir, "JUDGES.md")
	old := r.report(t)
	os.WriteFile(path, []byte(strings.Replace(old, mdreport.Placeholder, mdreport.Marker+"\n\nquartz is judge noise here.\n", 1)), 0o644)
	if code := run(r.args(), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if got := r.report(t); !strings.Contains(got, "quartz is judge noise here.") || strings.Contains(got, "Analysis pending") {
		t.Errorf("the analysis did not survive:\n%s", got)
	}
}

func TestAJudgeThatIsNotThereIsSkippedNotFaked(t *testing.T) {
	stubTools(t, nil, func(string, string, []string) (string, string, error) {
		return "", "", os.ErrNotExist
	})
	r := aRig(t)
	var out, errOut bytes.Buffer
	if code := run(r.args("-pdfium", "", "-nodedir", filepath.Join(r.dir, "nowhere")), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := r.report(t)
	if !strings.Contains(got, "Judges: . Versions: poppler ?; mupdf ?; gs ?; qpdf ?; pdf.js ?; macOS ?.") {
		t.Errorf("the header does not say nobody judged:\n%s", got)
	}
	if !strings.Contains(got, "| PDF | Bytes | consensus |\n|---|---|---|\n") {
		t.Errorf("the table has columns for judges that are not there:\n%s", got)
	}
	for _, v := range r.results(t)[0].Verdicts {
		if !v.Skipped || v.OK {
			t.Errorf("%s was not skipped: %+v", v.Judge, v)
		}
	}
}

func TestWithoutPopplerThereIsNoReferenceButStillAConsensus(t *testing.T) {
	stubTools(t, []string{"mutool", "gs", "sips"}, aMachine(t, 3, nil))
	r := aRig(t)
	var out, errOut bytes.Buffer
	if code := run(r.args("-pdfium", ""), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	rs := r.results(t)
	// Nobody said how many pages there are, so only page 1 is sampled.
	if len(rs[0].SampledPages) != 1 {
		t.Errorf("sampled %v", rs[0].SampledPages)
	}
	for _, v := range rs[0].Verdicts {
		if len(v.Diffs) != 0 {
			t.Errorf("%s has a distance to a reference that is not there", v.Judge)
		}
	}
	if rs[0].ConsensusMax <= 0 {
		t.Errorf("mupdf, gs and quartz did not disagree: %+v", rs[0])
	}
	if !strings.Contains(r.report(t), "| ✅ 3p · 10 · – |") {
		t.Errorf("without a reference the text is a count:\n%s", r.report(t))
	}
}

func TestPdfiumIsFoundThroughTheEnvironment(t *testing.T) {
	stubTools(t, nil, aMachine(t, 1, nil))
	r := aRig(t)
	t.Setenv("PDFIUM_TEST", r.pdfium)
	var out, errOut bytes.Buffer
	args := []string{"-pdfs", r.pdf, "-out", filepath.Join(r.dir, "out"),
		"-report", filepath.Join(r.dir, "JUDGES.md"), "-results", filepath.Join(r.dir, "judges.json")}
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(r.report(t), "Judges: pdfium.") {
		t.Errorf("%s", r.report(t))
	}
}

func TestADocumentThatVanishedIsStillNamed(t *testing.T) {
	// A glob matches a dangling link; the size is nought and every judge
	// says what it found, rather than the run stopping on it.
	stubTools(t, []string{"qpdf"}, func(string, string, []string) (string, string, error) {
		return "", "", os.ErrNotExist
	})
	r := aRig(t)
	os.Remove(r.pdf)
	if err := os.Symlink(filepath.Join(r.dir, "gone", "doc.pdf"), r.pdf); err != nil {
		t.Skip(err)
	}
	var out, errOut bytes.Buffer
	if code := run(r.args("-pdfium", ""), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(r.report(t), "| 0 B | ❌ file does not exist |") {
		t.Errorf("%s", r.report(t))
	}
}

func TestRunSaysWhatIsWrong(t *testing.T) {
	stubTools(t, everyReader, aMachine(t, 1, nil))
	r := aRig(t)
	for _, tc := range []struct {
		name string
		args []string
		code int
		want string
	}{
		{"no documents named", nil, 2, "-pdfs is needed"},
		{"a flag it does not have", []string{"-nonsense"}, 2, "flag provided but not defined"},
		{"a glob nothing matches", []string{"-pdfs", filepath.Join(r.dir, "*.nothing")}, 1, "no PDFs matched"},
		{"results it cannot write", r.args("-results", filepath.Join(r.dir, "no", "judges.json")), 1, "judges: results:"},
		{"a report it cannot write", r.args("-report", filepath.Join(r.dir, "no", "JUDGES.md")), 1, "judges: report:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run(tc.args, &out, &errOut); code != tc.code {
				t.Errorf("exit %d, want %d: %s", code, tc.code, errOut.String())
			}
			if !strings.Contains(errOut.String(), tc.want) {
				t.Errorf("stderr %q, want %q", errOut.String(), tc.want)
			}
		})
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("JUDGES_TEST_X", "")
	if envOr("JUDGES_TEST_X", "d") != "d" {
		t.Error("unset")
	}
	t.Setenv("JUDGES_TEST_X", "v")
	if envOr("JUDGES_TEST_X", "d") != "v" {
		t.Error("set")
	}
}

func TestMainCallsRun(t *testing.T) {
	oldExit, oldArgs := osExit, os.Args
	defer func() { osExit, os.Args = oldExit, oldArgs }()
	got := -1
	osExit = func(code int) { got = code }
	os.Args = []string{"judges"}
	main()
	if got != 2 {
		t.Errorf("main exited %d, want 2", got)
	}
}
