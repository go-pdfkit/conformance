package main

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCellReadsPagesTextAndDistance(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    verdict
		ref  int
		want string
	}{
		{"a refusal", verdict{Err: "boom"}, 10, "❌ boom"},
		{"the reference", verdict{Judge: "poppler", OK: true, Pages: 3, TextChar: 10}, 10, "✅ 3p · 1.000 · ref"},
		{"a judge with a distance", verdict{Judge: "mupdf", OK: true, Pages: 3, TextChar: 11, WorstPct: 4.25, WorstPage: 2,
			Diffs: map[int]pageDiff{2: {}}}, 10, "✅ 3p · 1.100 · Δ4.2% (p2)"},
		{"quartz, which only has page 1", verdict{Judge: "quartz", OK: true, TextChar: -1, WorstPct: 9, WorstPage: 1,
			Diffs: map[int]pageDiff{1: {}}}, 10, "✅ – · – · Δ9.0% (p1 only)"},
		{"text where the reference had none", verdict{Judge: "gs", OK: true, TextChar: 7}, 0, "✅ – · 7 · –"},
		{"warnings", verdict{Judge: "qpdf", OK: true, TextChar: -1, Warnings: 2}, 10, "✅ – · – · – ⚠2"},
	} {
		if got := cell(tc.v, tc.ref); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestBytesAreReadable(t *testing.T) {
	for n, want := range map[int64]string{7: "7 B", 6887: "7 KB", 1055265: "1.1 MB"} {
		if got := fmtBytes(n); got != want {
			t.Errorf("%d: %q, want %q", n, got, want)
		}
	}
}

func TestVersionIsTheFirstLineOnEitherStream(t *testing.T) {
	stubTools(t, nil, func(_, name string, _ []string) (string, string, error) {
		switch name {
		case "pdftoppm":
			return "", "pdftoppm version 26.04.0\nCopyright\n", nil
		case "gs":
			return strings.Repeat("9", 70) + "\n", "", nil
		}
		return "", "", errors.New("not found")
	})
	if got := version("pdftoppm", "-v"); got != "pdftoppm version 26.04.0" {
		t.Errorf("on stderr: %q", got)
	}
	if got := version("gs", "--version"); len(got) != 60 {
		t.Errorf("a long line: %q", got)
	}
	if got := version("absent"); got != "?" {
		t.Errorf("a tool that is not there: %q", got)
	}
}

func TestPdfjsVersionIsReadOffThePackage(t *testing.T) {
	was := nodeDir
	defer func() { nodeDir = was }()
	nodeDir = t.TempDir()
	if got := pdfjsVersion(); got != "?" {
		t.Errorf("not installed: %q", got)
	}
	pkg := filepath.Join(nodeDir, "node_modules", "pdfjs-dist", "package.json")
	os.MkdirAll(filepath.Dir(pkg), 0o755)
	os.WriteFile(pkg, []byte("{"), 0o644)
	if got := pdfjsVersion(); got != "?" {
		t.Errorf("a package that will not parse: %q", got)
	}
	os.WriteFile(pkg, []byte(`{"name":"pdfjs-dist","version":"6.3.289"}`), 0o644)
	if got := pdfjsVersion(); got != "6.3.289" {
		t.Errorf("%q", got)
	}
}

func TestWriteResultsSaysWhenItCannot(t *testing.T) {
	dir := t.TempDir()
	if err := writeResults(filepath.Join(dir, "no", "such", "judges.json"), nil); err == nil {
		t.Error("written into a directory that does not exist")
	}
	// A value JSON cannot carry — nothing in the harness produces one, but a
	// record that could not be encoded must not be reported as written.
	if err := writeResults(filepath.Join(dir, "judges.json"), []fileResult{{ConsensusMax: math.NaN()}}); err == nil {
		t.Error("NaN was encoded")
	}
	if err := writeResults(filepath.Join(dir, "judges.json"), []fileResult{{File: "a.pdf"}}); err != nil {
		t.Error(err)
	}
}

func TestTheReportNamesItsJudgesAndTheirVersions(t *testing.T) {
	stubTools(t, everyReader, aMachine(t, 1, nil))
	wasNow, wasNode := now, nodeDir
	defer func() { now, nodeDir = wasNow, wasNode }()
	now = func() time.Time { return time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC) }
	nodeDir = t.TempDir()
	judges := []judge{
		{name: "qpdf", avail: func() bool { return true }},
		{name: "poppler", avail: func() bool { return true }},
		{name: "pdfium", avail: func() bool { return false }},
	}
	results := []fileResult{{File: "a.pdf", Bytes: 6887, ConsensusMax: 1.25, ConsensusPg: 3, Verdicts: []verdict{
		{Judge: "qpdf", OK: true, TextChar: -1},
		{Judge: "poppler", OK: true, Pages: 3, TextChar: 10},
		{Judge: "pdfium", Skipped: true, TextChar: -1},
	}}}
	path := filepath.Join(t.TempDir(), "JUDGES.md")
	if err := writeReport(path, results, judges); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	got := string(b)
	for _, want := range []string{
		"# Judged by every reader on this machine — 2026-09-06\n",
		"Judges: qpdf, poppler. Versions: poppler pdftoppm version 26.04.0; mupdf mutool version 1.28.3; gs 10.07.1; qpdf qpdf version 12.4.1; pdf.js ?; macOS 26.6.2.\n",
		"| PDF | Bytes | qpdf | poppler | consensus |\n|---|---|---|---|---|\n",
		"| a.pdf | 7 KB | ✅ – · – · – | ✅ 3p · 1.000 · ref | 1.2% (p3) |\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if err := writeReport(filepath.Join(t.TempDir(), "no", "JUDGES.md"), results, judges); err == nil {
		t.Error("a report was written into a directory that does not exist")
	}
}
