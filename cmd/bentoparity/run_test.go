package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const help = `
pdfops — do something to a PDF you already have.

  merge        join files, in the order given
                 pdfops merge <out.pdf> <in.pdf>
`

// fleet writes the smallest world the shipped table can be measured in: a
// tools directory, and a source tree holding the repositories it names.
func fleet(t *testing.T) (toolsDir, srcDir string) {
	t.Helper()
	root := t.TempDir()
	toolsDir = filepath.Join(root, "tools")
	srcDir = filepath.Join(root, "src")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"merge-pdf", "index", "ocr-pdf"} {
		if err := os.WriteFile(filepath.Join(toolsDir, n+".md"), []byte("# "+n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return toolsDir, srcDir
}

func withHelp(t *testing.T, text string, err error) {
	t.Helper()
	was := helpOf
	t.Cleanup(func() { helpOf = was })
	helpOf = func(string) (io.Reader, error) {
		if err != nil {
			return nil, err
		}
		return strings.NewReader(text), nil
	}
}

func TestEveryMissingArgumentIsNamed(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"-tools", "x"},
		{"-tools", "x", "-pdfops", "y"},
	} {
		var out, errw bytes.Buffer
		if code := run(args, &out, &errw); code != 2 {
			t.Errorf("%v exited %d", args, code)
		}
		if !strings.Contains(errw.String(), "is required") {
			t.Errorf("%v said %q", args, errw.String())
		}
	}
	var out, errw bytes.Buffer
	if code := run([]string{"-nonesuch"}, &out, &errw); code != 2 {
		t.Errorf("an unknown flag exited %d", code)
	}
}

func TestAToolDirectoryThatIsNotThereIsReported(t *testing.T) {
	withHelp(t, help, nil)
	var out, errw bytes.Buffer
	code := run([]string{"-tools", filepath.Join(t.TempDir(), "nope"), "-pdfops", "x", "-src", "y"}, &out, &errw)
	if code != 1 {
		t.Errorf("exited %d", code)
	}
	if out.Len() != 0 {
		t.Errorf("a figure was printed over a tool list that could not be read: %q", out.String())
	}
}

func TestABinaryThatSaysNothingIsReported(t *testing.T) {
	tools, src := fleet(t)
	withHelp(t, "", os.ErrNotExist)
	var out, errw bytes.Buffer
	if code := run([]string{"-tools", tools, "-pdfops", "x", "-src", src}, &out, &errw); code != 1 {
		t.Errorf("exited %d", code)
	}

	// And a binary that answers with no verbs in it, which is the shape a
	// changed help text has.
	withHelp(t, "usage: pdfops <command>\n", nil)
	out.Reset()
	errw.Reset()
	if code := run([]string{"-tools", tools, "-pdfops", "x", "-src", src}, &out, &errw); code != 1 {
		t.Errorf("help with no verbs exited %d", code)
	}
	if !strings.Contains(errw.String(), "no verbs") {
		t.Errorf("it said %q", errw.String())
	}
}

func TestNoFigureIsPrintedWhenAClaimCannotBeShown(t *testing.T) {
	// ⛔ The reason the tool exists. Here the source tree is empty, so every
	// symbol claim is unverifiable — and the number must not appear anyway.
	tools, src := fleet(t)
	withHelp(t, help, nil)
	var out, errw bytes.Buffer
	code := run([]string{"-tools", tools, "-pdfops", "x", "-src", src}, &out, &errw)
	if code != 1 {
		t.Errorf("exited %d", code)
	}
	if strings.Contains(out.String(), "parity") {
		t.Errorf("a parity figure was printed: %q", out.String())
	}
	if !strings.Contains(errw.String(), "not a capability") {
		t.Errorf("it said %q", errw.String())
	}
}

func TestAMeasurementThatVerifiesPrintsTheFigureAndTheGaps(t *testing.T) {
	// A world small enough that every claim in it can be shown: one tool, one
	// verb, and one tool nothing claims.
	root := t.TempDir()
	tools := filepath.Join(root, "tools")
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	// Four tools, three of them uncovered, so the three-per-line wrap runs.
	for _, n := range []string{"merge-pdf", "ocr-pdf", "deskew-pdf", "repair-pdf", "index"} {
		if err := os.WriteFile(filepath.Join(tools, n+".md"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	was := table
	t.Cleanup(func() { table = was })
	table = smallTable
	withHelp(t, help, nil)

	var out, errw bytes.Buffer
	if code := run([]string{"-tools", tools, "-pdfops", "x", "-src", src}, &out, &errw); code != 0 {
		t.Fatalf("exited %d: %s", code, errw.String())
	}
	s := out.String()
	for _, want := range []string{"25.0%", "(1/4)", "ocr-pdf", "repair-pdf", "every claim verified"} {
		if !strings.Contains(s, want) {
			t.Errorf("the report does not carry %q:\n%s", want, s)
		}
	}
	// ⛔ "index" is BentoPDF's contents page, not a tool. Counting it would put
	// a permanent uncoverable entry in the denominator.
	if strings.Contains(s, "index") {
		t.Errorf("the contents page was counted as a tool:\n%s", s)
	}
}

func TestMainHandsBackTheExitCode(t *testing.T) {
	old, oldArgs := osExit, os.Args
	t.Cleanup(func() { osExit, os.Args = old, oldArgs })
	got := -1
	osExit = func(code int) { got = code }
	os.Args = []string{"bentoparity"}
	main()
	if got != 2 {
		t.Errorf("main exited %d with no arguments", got)
	}
}

func TestHelpOfRunsSomethingReal(t *testing.T) {
	// ⛔ helpOf is the one thing every other test here replaces, so nothing
	// else would ever run it. A command that prints usage and exits non-zero
	// is the normal shape of --help, and this asserts that is not treated as a
	// failure.
	r, err := helpOf("go")
	if err != nil {
		t.Fatalf("go --help: %v", err)
	}
	b, _ := io.ReadAll(r)
	if len(b) == 0 {
		t.Error("go --help printed nothing")
	}
	if _, err := helpOf(filepath.Join(t.TempDir(), "nothing-here")); err == nil {
		t.Error("a binary that is not there was accepted")
	}
}
