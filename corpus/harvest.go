package corpus

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// A Plan says what to add to a corpus and from where.
type Plan struct {
	// Dir is the corpus. It is created if it is not there.
	Dir string
	// Origin names the population these documents belong to. It is also the
	// subdirectory they land in, so a corpus of several populations stays
	// countable — a prevalence is per origin or it is not a prevalence.
	Origin string
	// Query is handed to the archive's search.
	Query string
	// Want is how many documents to end up with in this origin.
	Want int
	// MaxBytes refuses a document larger than this. A scanned book can be
	// hundreds of megabytes and says no more about which codec it uses than a
	// small one does.
	MaxBytes int64
	// Workers is how many are fetched at once. Be modest: this is somebody
	// else's server.
	Workers int
	// Log, when set, is told what happened to each document.
	Log func(string, ...any)
}

// Harvest adds documents to a corpus until the plan's origin holds Want of
// them, and records every one in the manifest.
//
// It is resumable by construction: what is already in the manifest is skipped,
// so an interrupted run is continued by running it again, and a corpus is
// extended by asking for a larger Want.
func Harvest(ctx context.Context, a *Archive, p Plan) ([]Entry, error) {
	if p.Workers <= 0 {
		p.Workers = 4
	}
	if p.Log == nil {
		p.Log = func(string, ...any) {}
	}
	into := filepath.Join(p.Dir, p.Origin)
	if err := os.MkdirAll(into, 0o755); err != nil {
		return nil, err
	}
	existing, err := Read(p.Dir)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	inOrigin := 0
	for _, e := range existing {
		have[e.Path] = true
		if e.Origin == p.Origin {
			inOrigin++
		}
	}

	added := make(chan Entry)
	var wg sync.WaitGroup
	work := make(chan string)
	for i := 0; i < p.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range work {
				e, err := fetchOne(ctx, a, p, into, id)
				if err != nil {
					p.Log("%s: %v", id, err)
					continue
				}
				added <- e
			}
		}()
	}
	go func() { wg.Wait(); close(added) }()

	// Feed identifiers a page at a time until enough have landed.
	go func() {
		defer close(work)
		for page := 1; page <= maxPages; page++ {
			ids, err := a.Search(ctx, p.Query, searchRows, page)
			if err != nil {
				p.Log("search page %d: %v", page, err)
				return
			}
			if len(ids) == 0 {
				return
			}
			for _, id := range ids {
				if have[filepath.Join(p.Origin, id+".pdf")] {
					continue
				}
				select {
				case work <- id:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	out := existing
	for e := range added {
		out = append(out, e)
		inOrigin++
		p.Log("%s (%d/%d)", e.Path, inOrigin, p.Want)
		if err := Write(p.Dir, out); err != nil {
			return out, err
		}
		if inOrigin >= p.Want {
			break
		}
	}
	return out, Write(p.Dir, out)
}

// searchRows and maxPages bound one harvest: enough identifiers to fill a
// reasonable Want without asking the archive for its whole index.
const (
	searchRows = 100
	maxPages   = 60
)

// fetchOne downloads a single item's PDF into the corpus.
func fetchOne(ctx context.Context, a *Archive, p Plan, into, id string) (Entry, error) {
	f, err := a.PDF(ctx, id)
	if err != nil {
		return Entry{}, err
	}
	if p.MaxBytes > 0 && f.Bytes > p.MaxBytes {
		return Entry{}, fmt.Errorf("%d bytes, over the %d limit", f.Bytes, p.MaxBytes)
	}
	name := strings.ReplaceAll(id, "/", "_") + ".pdf"
	path := filepath.Join(into, name)
	w, err := os.Create(path)
	if err != nil {
		return Entry{}, err
	}
	sum, n, err := a.Fetch(ctx, id, f.Name, w)
	cerr := w.Close()
	if err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
		return Entry{}, err
	}
	if !looksLikePDF(path) {
		os.Remove(path)
		return Entry{}, fmt.Errorf("what came back is not a PDF")
	}
	return Entry{
		Path:    filepath.Join(p.Origin, name),
		Origin:  p.Origin,
		Source:  a.URL(id, f.Name),
		Bytes:   n,
		SHA256:  sum,
		Fetched: Now(),
	}, nil
}

// looksLikePDF checks the header, because a server that answers 200 with an
// error page would otherwise put an HTML document in a corpus of PDFs and
// every later measurement would quietly count it.
func looksLikePDF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var head [5]byte
	if _, err := io.ReadFull(f, head[:]); err != nil {
		return false
	}
	return string(head[:]) == "%PDF-"
}
