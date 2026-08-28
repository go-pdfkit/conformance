package corpus

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAManifestSurvivesBeingWrittenAndReadBack(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	want := []Entry{
		{Path: "b/second.pdf", Origin: "b", Source: "https://x/2", Bytes: 2, SHA256: "bb", Fetched: when},
		{Path: "a/first.pdf", Origin: "a", Source: "https://x/1", Bytes: 1, SHA256: "aa", Fetched: when},
	}
	if err := Write(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by path, so two runs that fetched the same documents in a
	// different order write the same file.
	if len(got) != 2 || got[0].Path != "a/first.pdf" || got[1].Path != "b/second.pdf" {
		t.Fatalf("got %+v", got)
	}
	if got[0].Source != "https://x/1" || got[1].Bytes != 2 || !got[0].Fetched.Equal(when) {
		t.Errorf("a field did not survive: %+v", got)
	}
	if n := Origins(got); n["a"] != 1 || n["b"] != 1 {
		t.Errorf("origins %v", n)
	}
}

func TestADirectoryWithNoManifestIsAnEmptyCorpus(t *testing.T) {
	// Which is what a corpus about to be built looks like.
	got, err := Read(t.TempDir())
	if err != nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestAManifestThatWillNotParseSaysWhichLine(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"too few fields", "a\tb\n", "2 fields"},
		{"a size that is not a number", "a\tb\tc\tnot-a-number\te\t2026-08-28T12:00:00Z\n", "size"},
		{"a time that is not a time", "a\tb\tc\t1\te\tyesterday\n", "time"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Read(dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want it to mention %q", err, tc.want)
			}
			if !strings.Contains(err.Error(), "line 1") {
				t.Errorf("the error does not say which line: %v", err)
			}
		})
	}
}

func TestAHeaderAndBlankLinesAreNotDocuments(t *testing.T) {
	dir := t.TempDir()
	body := "path\torigin\tsource\tbytes\tsha256\tfetched\n" +
		"\n" +
		"a/x.pdf\ta\thttps://x\t3\tdd\t2026-08-28T12:00:00Z\n"
	if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d entries, want 1: %+v", len(got), got)
	}
}

func TestReadingRefusesADirectoryItCannotOpen(t *testing.T) {
	dir := t.TempDir()
	// A directory where the manifest should be: opening it succeeds on some
	// systems and reading it fails, which is still not a manifest.
	if err := os.Mkdir(filepath.Join(dir, ManifestName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil {
		t.Error("a directory was read as a manifest")
	}
}

func TestWritingRefusesADirectoryThatIsNotThere(t *testing.T) {
	if err := Write(filepath.Join(t.TempDir(), "nope"), nil); err == nil {
		t.Error("no error writing into a directory that does not exist")
	}
}

func TestDigestHashesWhatItCopies(t *testing.T) {
	var out bytes.Buffer
	sum, n, err := Digest(strings.NewReader("abc"), &out)
	if err != nil || n != 3 || out.String() != "abc" {
		t.Fatalf("got %q, %d, %v", out.String(), n, err)
	}
	// The SHA-256 of "abc", which is the one everybody knows.
	if sum != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("digest %s", sum)
	}
}

// failingWriter is a sink that refuses, so the copy inside Digest can fail.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }

func TestDigestReportsAWriteThatFailed(t *testing.T) {
	if _, _, err := Digest(strings.NewReader("abc"), failingWriter{}); err == nil {
		t.Error("no error from a sink that refused")
	}
}
