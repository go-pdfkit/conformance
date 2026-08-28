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

func TestAManifestThatWillNotParseSaysWhy(t *testing.T) {
	head := "path\torigin\tsource\tbytes\tsha256\tfetched\n"
	for _, tc := range []struct {
		name, body, want string
	}{
		{"a header naming nothing we know", "nothing\tuseful\n", "names no path column"},
		{"a header missing one column", "path\torigin\tsource\tbytes\tsha256\n", "names no fetched column"},
		{"a row shorter than the header", head + "a\tb\n", "2 fields, and the header names"},
		{"a size that is not a number", head + "a\tb\tc\tnot-a-number\te\t2026-08-28T12:00:00Z\n", "size"},
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
		})
	}
}

func TestAnEarlierSchemaIsStillReadable(t *testing.T) {
	// The forms corpus was gathered before this repository existed and says
	// "issuer|file|url|bytes|sha256-8|fetched" — the same six facts under
	// other names, with the population in one column and the bare file name in
	// another. Making the format self-describing was cheaper than re-fetching
	// two thousand documents.
	dir := t.TempDir()
	body := "issuer\tfile\turl\tbytes\tsha256-8\tfetched\n" +
		"ca-cra\tgst190.pdf\thttps://x/gst190.pdf\t403556\t306ed166\t2026-08-26T14:06\n" +
		"us-irs\tf1040.pdf\thttps://x/f1040.pdf\t220237\t-\tfetched 2026-08-26 (pre-existing)\n"
	if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("read %d rows", len(got))
	}
	// The population and the bare name are rejoined, so a path is a path
	// whichever tool wrote it.
	if got[0].Path != filepath.Join("ca-cra", "gst190.pdf") {
		t.Errorf("path %q", got[0].Path)
	}
	if got[0].Bytes != 403556 || got[0].Fetched.Year() != 2026 {
		t.Errorf("got %+v", got[0])
	}
	// A fact the manifest does not give is recorded as absent rather than
	// refused: throwing two thousand documents away over a loose date would be
	// worse than a missing date.
	if got[1].SHA256 != "" {
		t.Errorf("a %q digest was kept as %q", "-", got[1].SHA256)
	}
	if !got[1].Fetched.IsZero() {
		t.Errorf("free text was read as a time: %v", got[1].Fetched)
	}
}

func TestAnEmptyFileIsAnEmptyCorpus(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(dir)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
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

func TestARowShortOfAnyColumnIsReported(t *testing.T) {
	// The header decides where each fact sits, so a row can run out before any
	// of them. One header, rows of growing length: each stops one column
	// further along.
	// The path column last as well, since a row can run out before the very
	// first fact a manifest is supposed to carry.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestName),
		[]byte("origin\tsource\tsha256\tbytes\tfetched\tpath\nonly-one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil || !strings.Contains(err.Error(), "the header names") {
		t.Errorf("a row short of its path gave %v", err)
	}

	head := "path\torigin\tsource\tsha256\tbytes\tfetched\n"
	for _, row := range []string{
		"",
		"a",
		"a\tb",
		"a\tb\tc",
		"a\tb\tc\td",
		"a\tb\tc\td\t5",
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(head+row+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Read(dir)
		if row == "" {
			// A blank line is not a short row; it is not a row at all.
			if err != nil {
				t.Errorf("a blank line was read as a row: %v", err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), "the header names") {
			t.Errorf("row %q gave %v", row, err)
		}
	}
}
