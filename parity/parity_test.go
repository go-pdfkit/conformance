// Copyright (c) the go-pdfkit authors.
// SPDX-License-Identifier: BSD-3-Clause

package parity

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func tools(names ...string) fstest.MapFS {
	m := fstest.MapFS{}
	for _, n := range names {
		m[n+".md"] = &fstest.MapFile{Data: []byte("# " + n)}
	}
	return m
}

const help = `
pdfops — do something to a PDF you already have.

usage: pdfops [-password <password>] <command> [options]

  merge        join files, in the order given
                 pdfops merge <out.pdf> <in.pdf> [in.pdf …]
  split        cut into files of at most n pages
                 pdfops split -every <n> <in.pdf> <out-directory>
`

func TestTheToolListIsReadFromTheOtherProjectsOwnDocumentation(t *testing.T) {
	// ⛔ The denominator of a parity figure is exactly the thing that must not
	// quietly drift. A list typed out here would be a snapshot of what
	// somebody believed on the day they typed it.
	got, err := ReadTools(tools("merge-pdf", "split-pdf", "index"), "index")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "merge-pdf,split-pdf" {
		t.Errorf("the tool list came back as %v", got)
	}
}

func TestAnEmptyToolListIsAFailureAndNotAHundredPercent(t *testing.T) {
	// ⛔ Reporting "0 tools, 100% parity" from a directory that listed nothing
	// is the most flattering possible reading of a broken instrument.
	if _, err := ReadTools(fstest.MapFS{}); err == nil {
		t.Error("an empty directory was accepted as a tool list")
	}
	if _, err := ReadTools(fstest.MapFS{"notes.txt": &fstest.MapFile{}}); err == nil {
		t.Error("a directory of no Markdown was accepted as a tool list")
	}
	if _, err := ReadTools(unreadableFS{}); err == nil {
		t.Error("a directory that cannot be read was accepted")
	}
}

func TestTheVerbsAreReadOffABinarysOwnHelp(t *testing.T) {
	got, err := Verbs(strings.NewReader(help))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "merge,split" {
		t.Errorf("the verbs came back as %v — the continuation lines are indented further "+
			"and must not be read as verbs", got)
	}
}

func TestHelpThatNamesNoVerbsIsAFailure(t *testing.T) {
	// ⛔ A binary whose help changed shape returns nothing, and nothing then
	// verifies, which would report every verb claim as a missing capability —
	// a fleet-wide regression invented by a formatting change.
	if _, err := Verbs(strings.NewReader("usage: pdfops <command>\n")); err == nil {
		t.Error("help with no verbs in it was accepted")
	}
	if _, err := Verbs(errReader{}); err == nil {
		t.Error("a reader that fails was accepted")
	}
}

// unreadableFS is a filesystem that refuses to be listed, which is what a
// wrong -tools path looks like from in here.
type unreadableFS struct{}

func (unreadableFS) Open(string) (fs.File, error) { return nil, fs.ErrPermission }

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("no") }

func TestAVerbClaimIsCheckedAgainstTheBinary(t *testing.T) {
	verbs := []string{"merge", "split"}
	if err := Verify("verb:merge", verbs, fstest.MapFS{}); err != nil {
		t.Errorf("a verb that is there was refused: %v", err)
	}
	if err := Verify("verb:teleport", verbs, fstest.MapFS{}); err == nil {
		t.Error("a verb that is not in the binary was accepted")
	}
}

func TestASymbolClaimIsCheckedAgainstTheSource(t *testing.T) {
	src := fstest.MapFS{
		"odf/parse.go":      &fstest.MapFile{Data: []byte("func Parse(b []byte) {}")},
		"odf/README.md":     &fstest.MapFile{Data: []byte("func Parse is documented here")},
		"empty/notes.txt":   &fstest.MapFile{Data: []byte("func Parse")},
		"empty/placeholder": &fstest.MapFile{Data: []byte("x")},
	}
	if err := Verify("go-odf/odf#func Parse", nil, src); err != nil {
		t.Errorf("a symbol that is there was refused: %v", err)
	}
	// ⛔ Go source only. A README saying a function exists is a claim, not the
	// function — and a README is exactly where a capability map goes stale.
	if err := Verify("go-x/empty#func Parse", nil, src); err == nil {
		t.Error("a symbol found only outside Go source was accepted")
	}
	if err := Verify("go-odf/odf#func Missing", nil, src); err == nil {
		t.Error("a symbol that is not there was accepted")
	}
	if err := Verify("go-nowhere/nope#func Parse", nil, src); err == nil {
		t.Error("a repository that is not checked out was accepted")
	}
	for _, bad := range []string{"nonsense", "#sym", "repo#"} {
		if err := Verify(bad, nil, src); err == nil {
			t.Errorf("the malformed claim %q was accepted", bad)
		}
	}
}

func TestASourceTreeThatCannotBeWalkedIsNotAMissingCapability(t *testing.T) {
	// ⛔ The false negative this guards. A repository that is checked out but
	// unreadable must not come back as "does not contain X": that reads exactly
	// like a real gap, and a capability measurement must never fail in the
	// direction that invents one.
	err := Verify("go-odf/odf#func Parse", nil, halfReadableFS{})
	if err == nil {
		t.Fatal("an unreadable repository was accepted")
	}
	if strings.Contains(err.Error(), "does not contain") {
		t.Errorf("an unreadable repository was reported as %q, which is the message for a "+
			"repository that was read and came up empty", err)
	}
}

// TestAFileThatCannotBeReadIsNotAMissingCapability is the same false negative
// one level down: the directory lists, and one of its Go files will not open.
func TestAFileThatCannotBeReadIsNotAMissingCapability(t *testing.T) {
	src := unopenableFile{fstest.MapFS{
		"odf/parse.go": &fstest.MapFile{Data: []byte("func Parse() {}")},
	}}
	err := Verify("go-odf/odf#func Parse", nil, src)
	if err == nil {
		t.Fatal("a file that cannot be opened was accepted")
	}
	if strings.Contains(err.Error(), "does not contain") {
		t.Errorf("an unreadable file was reported as %q, which is the message for a "+
			"repository that was read and came up empty", err)
	}
}

// unopenableFile lists like its wrapped filesystem and refuses to open any
// Go file in it.
type unopenableFile struct{ fs.FS }

func (u unopenableFile) Open(name string) (fs.File, error) {
	if strings.HasSuffix(name, ".go") {
		return nil, fs.ErrPermission
	}
	return u.FS.Open(name)
}

// halfReadableFS lets the repository be found and refuses to be read: Stat
// succeeds on the directory, and listing it fails.
type halfReadableFS struct{}

func (halfReadableFS) Open(name string) (fs.File, error) {
	if name == "odf" {
		return nil, fs.ErrPermission
	}
	return nil, fs.ErrNotExist
}

func (halfReadableFS) Stat(name string) (fs.FileInfo, error) {
	if name == "odf" {
		return dirInfo{}, nil
	}
	return nil, fs.ErrNotExist
}

type dirInfo struct{}

func (dirInfo) Name() string       { return "odf" }
func (dirInfo) Size() int64        { return 0 }
func (dirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (dirInfo) ModTime() time.Time { return time.Time{} }
func (dirInfo) IsDir() bool        { return true }
func (dirInfo) Sys() any           { return nil }

func TestATableThatClaimsSomethingTwiceIsRefused(t *testing.T) {
	// ⛔ SILENT otherwise: the map keeps the last entry and the total comes out
	// short of what the table appears to say. This check exists because ten
	// entries were added and the figure rose by nine.
	list := []string{"a", "b"}
	_, err := Measure(list, []Claim{
		{Tool: "a", Claim: "verb:x"},
		{Tool: "a", Claim: "verb:y"},
	}, []string{"x", "y"}, fstest.MapFS{})
	if err == nil {
		t.Fatal("a tool claimed twice was accepted")
	}
	if !strings.Contains(err.Error(), "twice") {
		t.Errorf("the refusal is %q", err)
	}
}

func TestATableThatNamesAToolTheOtherProjectDoesNotHaveIsRefused(t *testing.T) {
	// It inflates nothing by itself, but it means the table is being written
	// against something other than the list — and that error has a direction
	// nobody notices.
	_, err := Measure([]string{"a"}, []Claim{{Tool: "invented", Claim: "verb:x"}},
		[]string{"x"}, fstest.MapFS{})
	if err == nil {
		t.Fatal("a claim for a tool that does not exist was accepted")
	}
	if !strings.Contains(err.Error(), "not in the tool list") {
		t.Errorf("the refusal is %q", err)
	}
}

func TestNoFigureIsReportedWhenAClaimCannotBeShown(t *testing.T) {
	// ⛔ The whole point. A parity number assembled from a mapping nobody
	// checks is a vibe with a decimal point in it.
	_, err := Measure([]string{"a", "b"}, []Claim{{Tool: "a", Claim: "verb:nowhere"}},
		[]string{"x"}, fstest.MapFS{})
	if err == nil {
		t.Fatal("a figure was reported over an unverifiable claim")
	}
	if !strings.Contains(err.Error(), "no figure") && !strings.Contains(err.Error(), "could not be shown") {
		t.Errorf("the refusal is %q", err)
	}
}

func TestAToolWithNoClaimCountsAsNotCovered(t *testing.T) {
	// Absence is the default, and the conservative direction: this must never
	// make the fleet look better than it is.
	rep, err := Measure([]string{"a", "b", "c"},
		[]Claim{{Tool: "a", Claim: "verb:x"}, {Tool: "b", Claim: ""}},
		[]string{"x"}, fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Covered) != 1 || len(rep.Uncovered) != 2 {
		t.Errorf("%d covered, %d not, of %d", len(rep.Covered), len(rep.Uncovered), len(rep.Tools))
	}
	if got := rep.Percent(); got < 33.2 || got > 33.4 {
		t.Errorf("one of three is %.1f%%", got)
	}
	if (Report{}).Percent() != 0 {
		t.Error("an empty report is not zero per cent")
	}
}

func TestTheShippedTableIsWellFormed(t *testing.T) {
	// ⛔ The table in this repository is checked HERE rather than only when
	// somebody runs the command: a duplicate or a malformed claim would
	// otherwise sit in the source until the next measurement.
	seen := map[string]bool{}
	for _, c := range BentoPDF {
		if seen[c.Tool] {
			t.Errorf("%s is claimed twice", c.Tool)
		}
		seen[c.Tool] = true
		if c.Claim == "" {
			t.Errorf("%s carries an empty claim; leave it out instead", c.Tool)
			continue
		}
		if strings.HasPrefix(c.Claim, "verb:") {
			continue
		}
		repo, sym, ok := strings.Cut(c.Claim, "#")
		if !ok || repo == "" || sym == "" {
			t.Errorf("%s claims %q, which is neither verb:NAME nor REPO#SYMBOL", c.Tool, c.Claim)
		}
	}
	if len(BentoPDF) < 50 {
		t.Errorf("the table holds %d entries, which is fewer than this fleet was known to do", len(BentoPDF))
	}
}

var _ fs.FS = fstest.MapFS{}
