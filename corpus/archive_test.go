package corpus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeArchive stands in for the site: a search that answers with the
// identifiers it was given, metadata per item, and the file bytes.
type fakeArchive struct {
	pages   [][]string                     // one page of identifiers each
	files   map[string][]map[string]string // identifier -> file records
	bytes   map[string]string              // "identifier/name" -> body
	refuse  map[string]int                 // "identifier/name" -> status
	badJSON bool
}

func (f *fakeArchive) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/advancedsearch.php", func(w http.ResponseWriter, r *http.Request) {
		if f.badJSON {
			w.Write([]byte("not json"))
			return
		}
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			page = int(p[0] - '0')
		}
		ids := []string{}
		if page-1 < len(f.pages) {
			ids = f.pages[page-1]
		}
		docs := []map[string]string{}
		for _, id := range ids {
			docs = append(docs, map[string]string{"identifier": id})
		}
		json.NewEncoder(w).Encode(map[string]any{"response": map[string]any{"docs": docs}})
	})
	mux.HandleFunc("/metadata/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/metadata/")
		if f.badJSON {
			w.Write([]byte("not json"))
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"files": f.files[id]})
	})
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/download/")
		if code, ok := f.refuse[key]; ok {
			w.WriteHeader(code)
			return
		}
		body, ok := f.bytes[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(body))
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func TestSearchReadsTheIdentifiersBack(t *testing.T) {
	f := &fakeArchive{pages: [][]string{{"one", "two"}, {}}}
	a := &Archive{Base: f.server(t).URL}
	got, err := a.Search(context.Background(), "anything", 10, 1)
	if err != nil || len(got) != 2 || got[0] != "one" {
		t.Fatalf("got %v, %v", got, err)
	}
	if got, _ := a.Search(context.Background(), "anything", 10, 2); len(got) != 0 {
		t.Errorf("page 2 gave %v", got)
	}
}

func TestSearchReportsWhatWentWrong(t *testing.T) {
	t.Run("a site that is not there", func(t *testing.T) {
		a := &Archive{Base: "http://127.0.0.1:1"}
		if _, err := a.Search(context.Background(), "x", 1, 1); err == nil {
			t.Error("no error")
		}
	})
	t.Run("an answer that is not JSON", func(t *testing.T) {
		f := &fakeArchive{badJSON: true}
		a := &Archive{Base: f.server(t).URL}
		if _, err := a.Search(context.Background(), "x", 1, 1); err == nil {
			t.Error("no error")
		}
	})
	t.Run("a request that cannot be built", func(t *testing.T) {
		a := &Archive{Base: "://"}
		if _, err := a.Search(context.Background(), "x", 1, 1); err == nil {
			t.Error("no error")
		}
	})
}

func TestPDFTakesTheSmallestOne(t *testing.T) {
	// A scanned book often holds a full-colour PDF and a bitonal one. The
	// bitonal one is the scanned page as a fax, which is what is worth having.
	f := &fakeArchive{files: map[string][]map[string]string{
		"book": {
			{"name": "book.pdf", "format": "Text PDF", "size": "900"},
			{"name": "book_bw.pdf", "format": "Text PDF", "size": "100"},
			{"name": "book_meta.xml", "format": "Metadata", "size": "10"},
			{"name": "broken.pdf", "format": "Text PDF", "size": "nonsense"},
		},
	}}
	a := &Archive{Base: f.server(t).URL}
	got, err := a.PDF(context.Background(), "book")
	if err != nil || got.Name != "book_bw.pdf" || got.Bytes != 100 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestPDFSaysWhenThereIsNone(t *testing.T) {
	f := &fakeArchive{files: map[string][]map[string]string{
		"empty": {{"name": "scan.jp2", "format": "JP2", "size": "10"}},
	}}
	a := &Archive{Base: f.server(t).URL}
	if _, err := a.PDF(context.Background(), "empty"); err == nil {
		t.Error("no error for an item with no PDF")
	}
	if _, err := (&Archive{Base: "http://127.0.0.1:1"}).PDF(context.Background(), "x"); err == nil {
		t.Error("no error from a site that is not there")
	}
	bad := &fakeArchive{badJSON: true}
	if _, err := (&Archive{Base: bad.server(t).URL}).PDF(context.Background(), "x"); err == nil {
		t.Error("no error from metadata that is not JSON")
	}
}

func TestFetchWritesTheBytesAndSaysWhatTheyHashTo(t *testing.T) {
	f := &fakeArchive{bytes: map[string]string{"item/x.pdf": "%PDF-1.7 hello"}}
	a := &Archive{Base: f.server(t).URL}
	var out strings.Builder
	sum, n, err := a.Fetch(context.Background(), "item", "x.pdf", &out)
	if err != nil || n != 14 || out.String() != "%PDF-1.7 hello" || sum == "" {
		t.Fatalf("got %q, %d, %v", out.String(), n, err)
	}
	if _, _, err := a.Fetch(context.Background(), "item", "missing.pdf", &out); err == nil {
		t.Error("no error for a file that is not there")
	}
}

func TestTheDefaultsAreTheRealSiteAndTheRealClient(t *testing.T) {
	a := &Archive{}
	if !strings.HasPrefix(a.base(), "https://archive.org") {
		t.Errorf("base %q", a.base())
	}
	if a.client() != http.DefaultClient {
		t.Error("the default client is not the default client")
	}
	if u := a.URL("an item", "a file.pdf"); !strings.Contains(u, "an%20item") {
		t.Errorf("URL %q does not escape", u)
	}
}

func TestHarvestFillsAnOriginAndRecordsIt(t *testing.T) {
	f := &fakeArchive{
		pages: [][]string{{"a", "b", "c"}, {}},
		files: map[string][]map[string]string{
			"a": {{"name": "a.pdf", "format": "Text PDF", "size": "20"}},
			"b": {{"name": "b.pdf", "format": "Text PDF", "size": "20"}},
			"c": {{"name": "c.pdf", "format": "Text PDF", "size": "999999"}},
		},
		bytes: map[string]string{
			"a/a.pdf": "%PDF-a", "b/b.pdf": "%PDF-b", "c/c.pdf": "%PDF-c",
		},
	}
	realNow := Now
	Now = func() time.Time { return time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC) }
	defer func() { Now = realNow }()

	dir := t.TempDir()
	got, err := Harvest(context.Background(), &Archive{Base: f.server(t).URL}, Plan{
		Dir: dir, Origin: "scans", Query: "q", Want: 2, MaxBytes: 100, Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("harvested %d, want 2: %+v", len(got), got)
	}
	// c is over the size limit and must not be among them.
	for _, e := range got {
		if strings.Contains(e.Path, "c.pdf") {
			t.Errorf("a document over the limit was taken: %+v", e)
		}
	}
	// And it is on disk, and in the manifest, and the manifest reads back.
	back, err := Read(dir)
	if err != nil || len(back) != 2 {
		t.Fatalf("manifest has %d entries, %v", len(back), err)
	}
	if _, err := os.Stat(filepath.Join(dir, back[0].Path)); err != nil {
		t.Errorf("%s is not on disk: %v", back[0].Path, err)
	}
}

func TestHarvestIsResumable(t *testing.T) {
	// What is already in the manifest is skipped, so an interrupted run is
	// continued by running it again and a corpus is extended by asking for
	// more.
	f := &fakeArchive{
		pages: [][]string{{"a", "b"}, {}},
		files: map[string][]map[string]string{
			"a": {{"name": "a.pdf", "format": "Text PDF", "size": "20"}},
			"b": {{"name": "b.pdf", "format": "Text PDF", "size": "20"}},
		},
		bytes: map[string]string{"a/a.pdf": "%PDF-a", "b/b.pdf": "%PDF-b"},
	}
	a := &Archive{Base: f.server(t).URL}
	dir := t.TempDir()
	p := Plan{Dir: dir, Origin: "scans", Query: "q", Want: 1, Workers: 1}
	if _, err := Harvest(context.Background(), a, p); err != nil {
		t.Fatal(err)
	}
	p.Want = 2
	got, err := Harvest(context.Background(), a, p)
	if err != nil || len(got) != 2 {
		t.Fatalf("the second run gave %d, %v", len(got), err)
	}
}

func TestHarvestRefusesWhatIsNotAPDF(t *testing.T) {
	// A server that answers 200 with an error page would otherwise put an HTML
	// document in a corpus of PDFs, and every later measurement would count it.
	f := &fakeArchive{
		pages: [][]string{{"a"}, {}},
		files: map[string][]map[string]string{
			"a": {{"name": "a.pdf", "format": "Text PDF", "size": "20"}},
		},
		bytes: map[string]string{"a/a.pdf": "<html>not found</html>"},
	}
	dir := t.TempDir()
	got, err := Harvest(context.Background(), &Archive{Base: f.server(t).URL}, Plan{
		Dir: dir, Origin: "scans", Query: "q", Want: 1, Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("an HTML page was taken into a corpus of PDFs: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "scans", "a.pdf")); !os.IsNotExist(err) {
		t.Error("the refused file was left on disk")
	}
}

func TestTheConfiguredClientIsUsed(t *testing.T) {
	c := &http.Client{}
	if (&Archive{Client: c}).client() != c {
		t.Error("the configured client was ignored")
	}
}

func TestNowIsTheClock(t *testing.T) {
	// Now is a variable so a test can hold the clock still; by default it is
	// the real one, and an entry stamped with it is not from 1970.
	if Now().Year() < 2020 {
		t.Errorf("Now() is %v", Now())
	}
}

func TestHarvestRefusesACorpusItCannotWriteInto(t *testing.T) {
	// A file where the corpus directory should be.
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Harvest(context.Background(), &Archive{}, Plan{Dir: file, Origin: "o"}); err == nil {
		t.Error("no error harvesting into a file")
	}
}

func TestHarvestRefusesAManifestItCannotRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ManifestName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Harvest(context.Background(), &Archive{}, Plan{Dir: dir, Origin: "o"}); err == nil {
		t.Error("no error from a manifest that is a directory")
	}
}

func TestHarvestStopsWhenTheSearchDoes(t *testing.T) {
	// A search that fails ends the run rather than spinning through sixty
	// pages of the same error.
	f := &fakeArchive{badJSON: true}
	got, err := Harvest(context.Background(), &Archive{Base: f.server(t).URL}, Plan{
		Dir: t.TempDir(), Origin: "scans", Query: "q", Want: 1, Workers: 1,
	})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %d entries, %v", len(got), err)
	}
}

func TestHarvestStopsWhenItsContextDoes(t *testing.T) {
	// The search succeeds and hands back more identifiers than the single
	// worker can take, so the one feeding them is waiting on the channel when
	// the context is cancelled. That is the only place cancellation can be
	// noticed once a run is under way, and it has to be noticed there or the
	// run outlives its caller.
	blocked := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	mux := http.NewServeMux()
	mux.HandleFunc("/advancedsearch.php", func(w http.ResponseWriter, r *http.Request) {
		docs := []map[string]string{}
		if r.URL.Query().Get("page") == "1" {
			for _, id := range []string{"a", "b", "c", "d", "e"} {
				docs = append(docs, map[string]string{"identifier": id})
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"response": map[string]any{"docs": docs}})
	})
	mux.HandleFunc("/metadata/", func(w http.ResponseWriter, r *http.Request) {
		close(blocked) // the worker has taken one and is now busy
		<-ctx.Done()   // and stays busy until the harvest is cancelled
		w.WriteHeader(http.StatusInternalServerError)
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	go func() { <-blocked; cancel() }()
	got, err := Harvest(ctx, &Archive{Base: s.URL}, Plan{
		Dir: t.TempDir(), Origin: "scans", Query: "q", Want: 5, Workers: 1,
		Log: func(string, ...any) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("a cancelled harvest took %d documents", len(got))
	}
}

func TestHarvestReportsAManifestItCannotWrite(t *testing.T) {
	// The corpus directory is made read-only after the documents land, so the
	// manifest cannot be written and the run says so rather than reporting a
	// corpus it did not record.
	f := &fakeArchive{
		pages: [][]string{{"a"}, {}},
		files: map[string][]map[string]string{
			"a": {{"name": "a.pdf", "format": "Text PDF", "size": "20"}},
		},
		bytes: map[string]string{"a/a.pdf": "%PDF-a"},
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scans"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if _, err := Harvest(context.Background(), &Archive{Base: f.server(t).URL}, Plan{
		Dir: dir, Origin: "scans", Query: "q", Want: 1, Workers: 1,
	}); err == nil {
		t.Error("no error from a manifest that could not be written")
	}
}

func TestFetchingReportsWhatWentWrong(t *testing.T) {
	f := &fakeArchive{
		pages: [][]string{{"missing-metadata", "refused", "unwritable"}, {}},
		files: map[string][]map[string]string{
			"refused":    {{"name": "r.pdf", "format": "Text PDF", "size": "20"}},
			"unwritable": {{"name": "u.pdf", "format": "Text PDF", "size": "20"}},
		},
		refuse: map[string]int{"refused/r.pdf": http.StatusUnauthorized},
	}
	a := &Archive{Base: f.server(t).URL}
	dir := t.TempDir()
	into := filepath.Join(dir, "scans")
	if err := os.MkdirAll(into, 0o755); err != nil {
		t.Fatal(err)
	}
	p := Plan{Dir: dir, Origin: "scans", Log: func(string, ...any) {}}

	// An item whose metadata names no PDF.
	if _, err := fetchOne(context.Background(), a, p, into, "missing-metadata"); err == nil {
		t.Error("no error for an item with no PDF")
	}
	// A download the archive refuses — a great many of its texts are lending
	// restricted, so this is the common case rather than the odd one.
	if _, err := fetchOne(context.Background(), a, p, into, "refused"); err == nil {
		t.Error("no error for a refused download")
	}
	// A place the file cannot be created.
	if _, err := fetchOne(context.Background(), a, p, filepath.Join(into, "nope"), "unwritable"); err == nil {
		t.Error("no error for a file that could not be created")
	}
}

func TestReadingReportsAManifestItCannotOpen(t *testing.T) {
	// Not "there is none" — that is an empty corpus — but one that is there
	// and refuses to be read.
	dir := t.TempDir()
	path := filepath.Join(dir, ManifestName)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })
	if _, err := Read(dir); err == nil {
		t.Error("a manifest that cannot be opened was read")
	}
}

func TestWhatIsNotThereDoesNotLookLikeAPDF(t *testing.T) {
	if looksLikePDF(filepath.Join(t.TempDir(), "absent")) {
		t.Error("a file that is not there looked like a PDF")
	}
	short := filepath.Join(t.TempDir(), "short")
	if err := os.WriteFile(short, []byte("%PD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if looksLikePDF(short) {
		t.Error("three bytes looked like a PDF")
	}
}

func TestTheDefaultWorkerCountIsUsedWhenNoneIsGiven(t *testing.T) {
	f := &fakeArchive{pages: [][]string{{}}}
	if _, err := Harvest(context.Background(), &Archive{Base: f.server(t).URL}, Plan{
		Dir: t.TempDir(), Origin: "scans", Query: "q", Want: 1, // Workers unset
	}); err != nil {
		t.Fatal(err)
	}
}
