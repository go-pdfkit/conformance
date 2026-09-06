package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// stubTools stands every binary in: lookPath answers for the names in
// present, and runCmd hands each invocation to h instead of a process, so the
// harness runs on a machine that has none of them — the CI runner is one.
func stubTools(t *testing.T, present []string, h func(dir, name string, args []string) (string, string, error)) {
	t.Helper()
	wasRun, wasLook := runCmd, lookPath
	t.Cleanup(func() { runCmd, lookPath = wasRun, wasLook })
	runCmd = func(_ context.Context, dir, name string, args ...string) (string, string, error) {
		return h(dir, name, args)
	}
	lookPath = func(bin string) (string, error) {
		for _, p := range present {
			if p == bin {
				return "/usr/bin/" + bin, nil
			}
		}
		return "", errors.New("not found")
	}
}

// every reader the harness knows.
var everyReader = []string{"qpdf", "pdfinfo", "pdftoppm", "pdftotext", "mutool", "gs", "node", "sips"}

// writePNG writes a w×h page, white with a black box, so two judges that
// draw the box in different places differ by a share that can be predicted.
func writePNG(t *testing.T, path string, w, h int, box image.Rectangle) {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := uint8(255)
			if image.Pt(x, y).In(box) {
				c = 0
			}
			img.SetGray(x, y, color.Gray{Y: c})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

// exitErr is a genuine *exec.ExitError carrying code, since one cannot be
// built by hand and the qpdf judge reads the code off it.
func exitErr(t *testing.T, code int) error {
	t.Helper()
	err := exec.Command("sh", "-c", "exit "+strconv.Itoa(code)).Run()
	if err == nil {
		t.Fatalf("sh -c 'exit %d' succeeded", code)
	}
	return err
}

// utf32le encodes s the way pdfium_test writes its --txt.
func utf32le(s string, bom bool) []byte {
	var b []byte
	if bom {
		b = append(b, 0xFF, 0xFE, 0, 0)
	}
	for _, r := range s {
		b = append(b, byte(r), byte(r>>8), byte(r>>16), byte(r>>24))
	}
	return b
}

func argAfter(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func argWithPrefix(args []string, prefix string) string {
	for _, a := range args {
		if strings.HasPrefix(a, prefix) {
			return strings.TrimPrefix(a, prefix)
		}
	}
	return ""
}

// key names an invocation the way the fakes are keyed: the tool, and the verb
// that tells its uses apart.
func key(name string, args []string) string {
	base := filepath.Base(name)
	switch {
	case base == "mutool" && args[0] == "draw":
		if argAfter(args, "-F") == "txt" {
			return "mutool draw txt"
		}
		return "mutool draw png"
	case base == "gs" && args[0] != "--version":
		if argWithPrefix(args, "-sDEVICE=") == "txtwrite" {
			return "gs txt"
		}
		return "gs png"
	case base == "node", base == "mutool", strings.HasPrefix(args[0], "-"):
		return base + " " + args[0]
	}
	return base
}

// aMachine emulates every reader over an n-page document whose text is
// "hello world" on each page. fail names the invocations that refuse, by
// key. Each judge puts its ink in its own place so their renders differ.
func aMachine(t *testing.T, n int, fail map[string]error) func(dir, name string, args []string) (string, string, error) {
	t.Helper()
	box := func(judge string) image.Rectangle {
		off := map[string]int{"poppler": 0, "mupdf": 2, "gs": 20, "pdfium": 4, "pdfjs": 1, "quartz": 40}[judge]
		return image.Rect(off, off, off+30, off+30)
	}
	return func(dir, name string, args []string) (string, string, error) {
		k := key(name, args)
		if err, ok := fail[k]; ok {
			return "", "refused\n", err
		}
		switch k {
		case "qpdf --check":
			return "", "", nil
		case "qpdf --version":
			return "qpdf version 12.4.1\n", "", nil
		case "pdfinfo":
			return fmt.Sprintf("Title:          x\nPages:          %d\n", n), "Syntax Warning: something\n", nil
		case "pdftotext":
			return "hello world\n", "", nil
		case "pdftoppm -png":
			writePNG(t, args[len(args)-1]+".png", 80, 100, box("poppler"))
			return "", "", nil
		case "pdftoppm -v":
			return "", "pdftoppm version 26.04.0\nCopyright 2005-2026 The Poppler Developers\n", nil
		case "mutool info":
			return fmt.Sprintf("Pages: %d\n", n), "", nil
		case "mutool draw txt":
			return "hello world\n", "", nil
		case "mutool draw png":
			writePNG(t, argAfter(args, "-o"), 80, 100, box("mupdf"))
			return "", "", nil
		case "mutool -v":
			return "", "mutool version 1.28.3\n", nil
		case "gs txt":
			// txtwrite pads its lines to the column layout; the count must not
			// see that.
			os.WriteFile(argWithPrefix(args, "-sOutputFile="), []byte("hello      world   \n"), 0o644)
			return "", "", nil
		case "gs png":
			writePNG(t, argWithPrefix(args, "-sOutputFile="), 80, 100, box("gs"))
			return "", "", nil
		case "gs --version":
			return "10.07.1\n", "", nil
		case "pdfium_test --txt":
			for i := 0; i < n; i++ {
				os.WriteFile(filepath.Join(dir, fmt.Sprintf("pdfium.pdf.%d.txt", i)), utf32le("hello world", true), 0o644)
			}
			return "", fmt.Sprintf("Processing PDF file pdfium.pdf.\nProcessed %d pages.\n", n), nil
		case "pdfium_test --png":
			writePNG(t, filepath.Join(dir, "pdfium.pdf."+argWithPrefix(args, "--pages=")+".png"), 80, 100, box("pdfium"))
			return "", "Processing PDF file pdfium.pdf.\nProcessed 1 pages.\n", nil
		case "node pdfjs-text.mjs":
			return fmt.Sprintf("pages %d\nhello world\n", n), "", nil
		case "node pdfjs-render.mjs":
			writePNG(t, args[2], 80, 100, box("pdfjs"))
			return fmt.Sprintf("pages %d\n", n), "", nil
		case "sips -s":
			writePNG(t, argAfter(args, "--out"), 80, 100, box("quartz"))
			return args[3] + "\n  " + argAfter(args, "--out") + "\n", "", nil
		case "sw_vers -productVersion":
			return "26.6.2\n", "", nil
		}
		t.Fatalf("an invocation nothing expected: %s %v", name, args)
		return "", "", nil
	}
}

// aVerdict runs one judge over a two-page document under the machine above.
func aVerdict(t *testing.T, j func(context.Context, string, string, []int) verdict, fail map[string]error) verdict {
	t.Helper()
	stubTools(t, everyReader, aMachine(t, 2, fail))
	dir := t.TempDir()
	pdf := filepath.Join(dir, "doc.pdf")
	os.WriteFile(pdf, []byte("%PDF-1.7\n"), 0o644)
	pdfiumBin = filepath.Join(dir, "pdfium_test")
	nodeDir = dir
	return j(context.Background(), pdf, dir, []int{1, 2})
}

func TestQpdfReadsItsExitCode(t *testing.T) {
	// 0 is clean, 3 is warnings only — which is a pass with a count beside
	// it — and anything else is a refusal.
	if v := aVerdict(t, judgeQpdf, nil); !v.OK || v.Err != "" || v.TextChar != -1 {
		t.Errorf("clean: %+v", v)
	}
	if v := aVerdict(t, judgeQpdf, map[string]error{"qpdf --check": exitErr(t, 3)}); !v.OK {
		t.Errorf("warnings only: %+v", v)
	}
	v := aVerdict(t, judgeQpdf, map[string]error{"qpdf --check": exitErr(t, 2)})
	if v.OK || v.Err != "refused" {
		t.Errorf("errors: %+v", v)
	}
}

func TestQpdfCountsItsWarnings(t *testing.T) {
	stubTools(t, everyReader, func(string, string, []string) (string, string, error) {
		return "WARNING: a\nWARNING: b\n", "", nil
	})
	if v := judgeQpdf(context.Background(), "x.pdf", "", nil); !v.OK || v.Warnings != 2 {
		t.Errorf("%+v", v)
	}
}

func TestPopplerIsTheReference(t *testing.T) {
	v := aVerdict(t, judgePoppler, nil)
	if !v.OK || v.Pages != 2 || v.TextChar != 10 || len(v.Renders) != 2 {
		t.Fatalf("%+v", v)
	}
	if !strings.HasSuffix(v.Renders[2], "poppler_p2.png") {
		t.Errorf("page 2 is at %q", v.Renders[2])
	}
	// pdfinfo complained on one line.
	if v.Warnings != 1 {
		t.Errorf("%d warnings", v.Warnings)
	}
	for _, k := range []string{"pdfinfo", "pdftotext", "pdftoppm -png"} {
		v := aVerdict(t, judgePoppler, map[string]error{k: errors.New("boom")})
		if v.OK || v.Err != "refused" {
			t.Errorf("%s failing: %+v", k, v)
		}
	}
}

func TestMupdfSurvivesAnInfoThatFails(t *testing.T) {
	v := aVerdict(t, judgeMupdf, nil)
	if !v.OK || v.Pages != 2 || v.TextChar != 10 || len(v.Renders) != 2 || v.Warnings != 0 {
		t.Fatalf("%+v", v)
	}
	// mutool info refusing costs the page count and a warning, not the
	// verdict: the text and the renders are what the comparison needs.
	v = aVerdict(t, judgeMupdf, map[string]error{"mutool info": errors.New("boom")})
	if !v.OK || v.Pages != 0 || v.Warnings != 1 {
		t.Errorf("info failing: %+v", v)
	}
	for _, k := range []string{"mutool draw txt", "mutool draw png"} {
		if v := aVerdict(t, judgeMupdf, map[string]error{k: errors.New("boom")}); v.OK || v.Err != "refused" {
			t.Errorf("%s failing: %+v", k, v)
		}
	}
}

func TestGhostscriptCountsGlyphsNotPadding(t *testing.T) {
	v := aVerdict(t, judgeGs, nil)
	if !v.OK || v.Pages != 0 || v.TextChar != 10 || len(v.Renders) != 2 {
		t.Fatalf("%+v", v)
	}
	for _, k := range []string{"gs txt", "gs png"} {
		if v := aVerdict(t, judgeGs, map[string]error{k: errors.New("boom")}); v.OK || v.Err != "refused" {
			t.Errorf("%s failing: %+v", k, v)
		}
	}
	// txtwrite that succeeds without writing anything is a text of nought,
	// not a failure.
	machine := aMachine(t, 2, nil)
	stubTools(t, everyReader, func(dir, name string, args []string) (string, string, error) {
		if key(name, args) == "gs txt" {
			return "", "", nil
		}
		return machine(dir, name, args)
	})
	if v := judgeGs(context.Background(), "x.pdf", t.TempDir(), []int{1}); !v.OK || v.TextChar != 0 {
		t.Errorf("%+v", v)
	}
}

func TestPdfiumWorksOnACopyAndReadsUTF32(t *testing.T) {
	v := aVerdict(t, judgePdfium, nil)
	if !v.OK || v.Pages != 2 || v.TextChar != 20 || len(v.Renders) != 2 {
		t.Fatalf("%+v", v)
	}
	// Its progress narration is not a warning.
	if v.Warnings != 0 {
		t.Errorf("%d warnings from progress lines", v.Warnings)
	}
	if !strings.HasSuffix(v.Renders[2], "pdfium_p2.png") {
		t.Errorf("page 2 is at %q", v.Renders[2])
	}
	for _, k := range []string{"pdfium_test --txt", "pdfium_test --png"} {
		if v := aVerdict(t, judgePdfium, map[string]error{k: errors.New("boom")}); v.OK || v.Err != "refused" {
			t.Errorf("%s failing: %+v", k, v)
		}
	}
}

func TestPdfiumAnythingElseOnStderrIsAWarning(t *testing.T) {
	machine := aMachine(t, 1, nil)
	stubTools(t, everyReader, func(dir, name string, args []string) (string, string, error) {
		o, e, err := machine(dir, name, args)
		if strings.HasPrefix(key(name, args), "pdfium_test") {
			e += "Warning: font not found\n"
		}
		return o, e, err
	})
	dir := t.TempDir()
	pdf := filepath.Join(dir, "doc.pdf")
	os.WriteFile(pdf, []byte("%PDF"), 0o644)
	pdfiumBin = filepath.Join(dir, "pdfium_test")
	v := judgePdfium(context.Background(), pdf, dir, []int{1})
	if !v.OK || v.Warnings != 2 {
		t.Errorf("%+v", v)
	}
}

func TestPdfiumSaysWhenItCannotWork(t *testing.T) {
	stubTools(t, everyReader, aMachine(t, 1, nil))
	dir := t.TempDir()
	pdfiumBin = filepath.Join(dir, "pdfium_test")
	if v := judgePdfium(context.Background(), filepath.Join(dir, "absent.pdf"), dir, []int{1}); v.OK || v.Err == "" {
		t.Errorf("an absent document: %+v", v)
	}
	pdf := filepath.Join(dir, "doc.pdf")
	os.WriteFile(pdf, []byte("%PDF"), 0o644)
	if v := judgePdfium(context.Background(), pdf, filepath.Join(pdf, "not-a-dir"), []int{1}); v.OK || v.Err == "" {
		t.Errorf("an output directory that is a file: %+v", v)
	}
	// A page it said it drew but did not is named as missing.
	machine := aMachine(t, 1, nil)
	stubTools(t, everyReader, func(dir, name string, args []string) (string, string, error) {
		if key(name, args) == "pdfium_test --png" {
			return "", "", nil
		}
		return machine(dir, name, args)
	})
	if v := judgePdfium(context.Background(), pdf, dir, []int{1}); v.OK || v.Err != "no render for page 1" {
		t.Errorf("a render that was not written: %+v", v)
	}
	// A .txt it cannot read is left out of the count rather than failing it.
	stubTools(t, everyReader, machine)
	os.MkdirAll(filepath.Join(dir, "pdfium.pdf.9.txt"), 0o755)
	if v := judgePdfium(context.Background(), pdf, dir, []int{1}); !v.OK || v.TextChar != 10 {
		t.Errorf("an unreadable text: %+v", v)
	}
}

func TestUTF32LEIsCountedInCharacters(t *testing.T) {
	// pdfium_test's --txt is UTF-32LE with a BOM: four bytes a character,
	// which counted as bytes would read as four times the text.
	if got := utf32leToString(utf32le("héllo", true)); got != "héllo" {
		t.Errorf("with a BOM: %q", got)
	}
	if got := utf32leToString(utf32le("hi", false)); got != "hi" {
		t.Errorf("without one: %q", got)
	}
	// A trailing partial character is not a character.
	if got := utf32leToString(append(utf32le("a", true), 0x62, 0)); got != "a" {
		t.Errorf("with a torn tail: %q", got)
	}
	if got := utf32leToString(nil); got != "" {
		t.Errorf("empty: %q", got)
	}
}

func TestPdfjsReadsItsPageLine(t *testing.T) {
	v := aVerdict(t, judgePdfjs, nil)
	if !v.OK || v.Pages != 2 || v.TextChar != 10 || len(v.Renders) != 2 {
		t.Fatalf("%+v", v)
	}
	for _, k := range []string{"node pdfjs-text.mjs", "node pdfjs-render.mjs"} {
		if v := aVerdict(t, judgePdfjs, map[string]error{k: errors.New("boom")}); v.OK || v.Err != "refused" {
			t.Errorf("%s failing: %+v", k, v)
		}
	}
	// Without the page line, the text is all there is.
	machine := aMachine(t, 2, nil)
	stubTools(t, everyReader, func(dir, name string, args []string) (string, string, error) {
		if key(name, args) == "node pdfjs-text.mjs" {
			return "hello world\n", "", nil
		}
		return machine(dir, name, args)
	})
	if v := judgePdfjs(context.Background(), "x.pdf", t.TempDir(), []int{1}); !v.OK || v.Pages != 0 || v.TextChar != 10 {
		t.Errorf("%+v", v)
	}
}

func TestQuartzRendersPageOneOnly(t *testing.T) {
	v := aVerdict(t, judgeQuartz, nil)
	if !v.OK || v.TextChar != -1 || len(v.Renders) != 1 || v.Renders[1] == "" {
		t.Fatalf("%+v", v)
	}
	if v := aVerdict(t, judgeQuartz, map[string]error{"sips -s": errors.New("boom")}); v.OK || v.Err != "refused" {
		t.Errorf("failing: %+v", v)
	}
	stubTools(t, everyReader, func(_, _ string, args []string) (string, string, error) {
		writePNG(t, argAfter(args, "--out"), 4, 4, image.Rect(0, 0, 1, 1))
		return "Warning: x\nError: y\n", "", nil
	})
	if v := judgeQuartz(context.Background(), "x.pdf", t.TempDir(), nil); !v.OK || v.Warnings != 2 {
		t.Errorf("%+v", v)
	}
}

func TestRunCmdKeepsTheStreamsApartAndHonoursTheDirectory(t *testing.T) {
	dir := t.TempDir()
	o, e, err := runCmd(context.Background(), dir, "sh", "-c", "pwd; echo err 1>&2; exit 3")
	if exitCode(err) != 3 {
		t.Fatalf("exit %d, %v", exitCode(err), err)
	}
	want, _ := filepath.EvalSymlinks(dir)
	if got, _ := filepath.EvalSymlinks(strings.TrimSpace(o)); got != want {
		t.Errorf("ran in %q, not %q", got, want)
	}
	if e != "err\n" {
		t.Errorf("stderr %q", e)
	}
	if exitCode(errors.New("not a process")) != -1 {
		t.Error("an error that is not an exit is given a code")
	}
}

func TestAHangIsSaidAsAHangWhateverTheToolPrinted(t *testing.T) {
	// The deadline is read off the context, not the error: a killed process
	// reports a signal, and "signal: killed" in a cell is a hang nobody can
	// tell from a crash. And the tool's stderr so far — its warnings — is not
	// why the cell is red.
	was := judgeTimeout
	defer func() { judgeTimeout = was }()
	judgeTimeout = 50 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), judgeTimeout)
	defer cancel()
	_, _, err := runCmd(ctx, "", "sleep", "30")
	var h *hang
	if !errors.As(err, &h) {
		t.Fatalf("a tool that does not return was not called a hang: %v", err)
	}
	got := firstLine("Syntax Warning: so far so good\n", err)
	if got != "hung: sleep did not finish within 50ms" {
		t.Errorf("a hang reads %q", got)
	}
}

func TestFirstLineIsTheFirstThingSaid(t *testing.T) {
	if got := firstLine("\n  \n  the reason \nmore\n", errors.New("x")); got != "the reason" {
		t.Errorf("%q", got)
	}
	long := strings.Repeat("y", 200)
	if got := firstLine(long, nil); got != long[:120]+"…" {
		t.Errorf("%q", got)
	}
	if got := firstLine("  \n", errors.New("exit status 1")); got != "exit status 1" {
		t.Errorf("nothing said: %q", got)
	}
}

func TestSamplePagesIsFirstMiddleLast(t *testing.T) {
	for n, want := range map[int][]int{0: {1}, 1: {1}, 2: {1, 2}, 3: {1, 2, 3}, 10: {1, 5, 10}, 11: {1, 6, 11}} {
		if got := samplePages(n); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%d pages: %v, want %v", n, got, want)
		}
	}
}

func TestSmallHelpers(t *testing.T) {
	if pagesFrom("Title: x\nPages:          12\n") != 12 || pagesFrom("pages: 3") != 3 || pagesFrom("Pages: many") != 0 {
		t.Error("pagesFrom")
	}
	if countLines("") != 0 || countLines("  \n") != 0 || countLines("a\nb\n") != 2 {
		t.Error("countLines")
	}
	if textLen("hello   world\n\tà") != 11 {
		t.Errorf("textLen %d", textLen("hello   world\n\tà"))
	}
	stubTools(t, []string{"qpdf"}, nil)
	if !have("qpdf") || have("mutool") {
		t.Error("have")
	}
}
