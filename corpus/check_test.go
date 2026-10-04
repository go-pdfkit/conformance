package corpus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// put writes a document into a corpus and returns the entry that describes it,
// with the digest cut to n characters so a manifest that records a PREFIX can
// be tested as such. The forms corpus keeps sixteen.
func put(t *testing.T, dir, path, body string, n int) Entry {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(full)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sum, size, err := Digest(f, nilWriter{})
	if err != nil {
		t.Fatal(err)
	}
	if n > 0 && n < len(sum) {
		sum = sum[:n]
	}
	return Entry{Path: path, Origin: strings.SplitN(path, "/", 2)[0],
		Source: "https://x/" + path, Bytes: size, SHA256: sum,
		Fetched: time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)}
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestACorpusThatAgreesWithItsManifest(t *testing.T) {
	dir := t.TempDir()
	// One row with the full digest and one with sixteen characters of it,
	// because the two corpora this repository is run against do exactly that
	// and a check that demanded all sixty-four would call 2 268 rows changed.
	es := []Entry{
		put(t, dir, "a/one.pdf", "first", 0),
		put(t, dir, "b/two.pdf", "second", 16),
	}
	if err := Write(dir, es); err != nil {
		t.Fatal(err)
	}
	got, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("a corpus that agrees reported %v", got)
	}
}

func TestEveryWayACorpusCanDisagree(t *testing.T) {
	dir := t.TempDir()
	es := []Entry{
		put(t, dir, "a/gone.pdf", "will be removed", 0),
		put(t, dir, "a/resized.pdf", "same name, other length", 0),
		put(t, dir, "a/rewritten.pdf", "same length, other bytes", 0),
		put(t, dir, "a/nodigest.pdf", "the manifest did not say", 0),
		put(t, dir, "a/kept.pdf", "untouched", 16),
	}
	es[3].SHA256 = "" // some documents predate the record that describes them
	if err := Write(dir, es); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "a/gone.pdf")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a/resized.pdf"), []byte("shorter"), 0o644); err != nil {
		t.Fatal(err)
	}
	// SAME LENGTH, other bytes: the size check cannot see this one and the
	// digest is the only thing that can.
	if err := os.WriteFile(filepath.Join(dir, "a/rewritten.pdf"),
		[]byte("same length, OTHER bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a/stranger.pdf"), []byte("nobody asked"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A file that is not a PDF is not a document of the corpus and must not
	// be reported: a corpus directory holds a manifest, and may hold notes.
	if err := os.WriteFile(filepath.Join(dir, "a/README.md"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"a/gone.pdf":      "missing",
		"a/resized.pdf":   "size",
		"a/rewritten.pdf": "digest",
		"a/stranger.pdf":  "unrecorded",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, p := range got {
		if want[p.Path] != p.Kind {
			t.Errorf("%s came back as %q, want %q", p.Path, p.Kind, want[p.Path])
		}
		if p.String() == "" {
			t.Errorf("%+v has no message", p)
		}
	}
	// Sorted by kind then path, so two runs over the same corpus print the
	// same thing and a diff of two reports is readable.
	for i := 1; i < len(got); i++ {
		if got[i-1].Kind > got[i].Kind {
			t.Errorf("not sorted by kind: %v", got)
		}
	}
}

func TestAFileThatWillNotOpenIsNotReportedAsMissing(t *testing.T) {
	// Reporting it as missing would send a reader to re-fetch a document that
	// is already on the disk.
	dir := t.TempDir()
	e := put(t, dir, "a/locked.pdf", "unreadable", 0)
	if err := Write(dir, []Entry{e}); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(dir, "a/locked.pdf")
	if err := os.Chmod(full, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(full, 0o644) })
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-0 file, so there is nothing to see")
	}
	got, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != "unreadable" {
		t.Fatalf("got %v", got)
	}
	if !strings.Contains(got[0].String(), "will not open") {
		t.Errorf("the message reads %q", got[0].String())
	}
}

func TestACheckOfSomethingThatIsNotACorpus(t *testing.T) {
	// A directory with no manifest is an empty corpus rather than an error,
	// which Read already decides -- so a check of one reports only what is on
	// disk and in no row.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lone.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != "unrecorded" {
		t.Fatalf("got %v", got)
	}
	// And a manifest that cannot be parsed is an error rather than an empty
	// corpus, or a corrupt record would read as a corpus that agrees.
	if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte("nonsense\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(dir); err == nil {
		t.Error("a manifest with no header came back as a corpus")
	}
}

func TestARowThatNamesADirectory(t *testing.T) {
	// The bytes of a directory cannot be read, and the walk never offers one,
	// so only a manifest row can bring it here. It must come back as an error
	// rather than as agreement.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a/folder.pdf"), 0o755); err != nil {
		t.Fatal(err)
	}
	e := Entry{Path: "a/folder.pdf", Origin: "a", Source: "https://x/1",
		Bytes: 0, SHA256: "deadbeef", Fetched: time.Now().UTC()}
	if err := Write(dir, []Entry{e}); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(dir); err == nil {
		t.Error("a row naming a directory came back without an error")
	}
}

func TestTwoProblemsOfOneKindReadInOrder(t *testing.T) {
	// Sorted by kind AND then by path, so two runs over the same corpus print
	// the same thing and a diff of two reports is about the corpus.
	dir := t.TempDir()
	e := put(t, dir, "a/kept.pdf", "untouched", 0)
	if err := Write(dir, []Entry{e}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a/zeta.pdf", "a/alpha.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "a/alpha.pdf" || got[1].Path != "a/zeta.pdf" {
		t.Fatalf("got %v", got)
	}
}

func TestADirectoryTheWalkCannotEnter(t *testing.T) {
	// A corpus half of which cannot be listed must be an error and not a
	// report of the half that could: "no problems" over an unreadable
	// directory is the worst answer available.
	if os.Geteuid() == 0 {
		t.Skip("root enters a mode-0 directory, so there is nothing to see")
	}
	dir := t.TempDir()
	e := put(t, dir, "a/kept.pdf", "untouched", 0)
	if err := Write(dir, []Entry{e}); err != nil {
		t.Fatal(err)
	}
	shut := filepath.Join(dir, "b")
	if err := os.MkdirAll(shut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shut, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(shut, 0o755) })
	if _, err := Check(dir); err == nil {
		t.Error("a corpus with an unreadable directory came back without an error")
	}
}
