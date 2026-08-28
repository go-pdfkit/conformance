package corpus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// An Archive fetches documents from the Internet Archive, which is where mass
// digitisation lives — and therefore where the codecs a form corpus never
// exercises are found. A scanned page is a fax: JBIG2 and CCITT are what it is
// stored in, and almost nothing else uses them.
type Archive struct {
	// Base is the site, so a test can point this at its own server.
	Base string
	// Client is the one used for every request; nil means http.DefaultClient.
	Client *http.Client
}

// base answers with the configured site or the real one.
func (a *Archive) base() string {
	if a.Base != "" {
		return a.Base
	}
	return "https://archive.org"
}

func (a *Archive) client() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return http.DefaultClient
}

// get performs one request and hands back the body, which the caller closes.
func (a *Archive) get(ctx context.Context, u string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	// The archive asks that a client say who it is.
	req.Header.Set("User-Agent", "go-pdfkit-conformance/1 (+https://github.com/go-pdfkit/conformance)")
	resp, err := a.client().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s", u, resp.Status)
	}
	return resp.Body, nil
}

// Search returns the identifiers of items matching a query, one page at a time.
// rows above a few hundred are refused by the site, so a caller after more asks
// for more pages rather than a bigger page.
func (a *Archive) Search(ctx context.Context, query string, rows, page int) ([]string, error) {
	u := a.base() + "/advancedsearch.php?" + url.Values{
		"q":      {query},
		"fl[]":   {"identifier"},
		"rows":   {fmt.Sprint(rows)},
		"page":   {fmt.Sprint(page)},
		"output": {"json"},
	}.Encode()
	body, err := a.get(ctx, u)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var payload struct {
		Response struct {
			Docs []struct {
				Identifier string `json:"identifier"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("search %q: %w", query, err)
	}
	out := make([]string, 0, len(payload.Response.Docs))
	for _, d := range payload.Response.Docs {
		if d.Identifier != "" {
			out = append(out, d.Identifier)
		}
	}
	return out, nil
}

// A File is one of an item's files: its name and how big it is.
type File struct {
	Name  string
	Bytes int64
}

// PDF finds an item's PDF and how big it is. An item may hold several — a
// scanned book often has both a full-colour one and a bitonal one — and the
// smallest is taken, because the bitonal one is the scanned page as a fax,
// which is the thing worth having here.
func (a *Archive) PDF(ctx context.Context, identifier string) (File, error) {
	body, err := a.get(ctx, a.base()+"/metadata/"+url.PathEscape(identifier))
	if err != nil {
		return File{}, err
	}
	defer body.Close()

	var meta struct {
		Files []struct {
			Name   string `json:"name"`
			Format string `json:"format"`
			Size   string `json:"size"`
		} `json:"files"`
	}
	if err := json.NewDecoder(body).Decode(&meta); err != nil {
		return File{}, fmt.Errorf("metadata %s: %w", identifier, err)
	}
	best := File{}
	for _, f := range meta.Files {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".pdf") {
			continue
		}
		var n int64
		fmt.Sscan(f.Size, &n)
		if n <= 0 {
			continue
		}
		if best.Name == "" || n < best.Bytes {
			best = File{Name: f.Name, Bytes: n}
		}
	}
	if best.Name == "" {
		return File{}, fmt.Errorf("%s holds no PDF", identifier)
	}
	return best, nil
}

// Fetch writes one of an item's files to w and returns its digest and size.
func (a *Archive) Fetch(ctx context.Context, identifier, name string, w io.Writer) (string, int64, error) {
	body, err := a.get(ctx, a.URL(identifier, name))
	if err != nil {
		return "", 0, err
	}
	defer body.Close()
	return Digest(body, w)
}

// URL is where one of an item's files lives, which the manifest records so that
// a document can be fetched again.
func (a *Archive) URL(identifier, name string) string {
	return a.base() + "/download/" + url.PathEscape(identifier) + "/" + url.PathEscape(name)
}

// Now is the clock the harvester stamps entries with, so a test can hold it
// still.
var Now = func() time.Time { return time.Now() }
