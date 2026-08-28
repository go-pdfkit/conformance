// Package corpus builds and records the populations of real PDFs that the
// conformance tools are run against.
//
// A corpus is a directory of files and a MANIFEST.tsv beside them. The manifest
// is what makes the corpus a measurement rather than a pile: it says where each
// file came from, when, how big it is and what its bytes hash to, so a figure
// quoted from it can be reproduced and a file that changed underneath can be
// noticed.
package corpus

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ManifestName is the file a corpus records itself in, beside its documents.
const ManifestName = "MANIFEST.tsv"

// An Entry is one document and where it came from.
type Entry struct {
	// Path is relative to the corpus directory, so a corpus can be moved.
	Path string
	// Origin is the population the document was drawn from — an issuing body,
	// an archive collection — and is what a prevalence is computed over. Two
	// documents from different origins are not interchangeable evidence.
	Origin string
	// Source is the URL it was fetched from.
	Source string
	Bytes  int64
	// SHA256 is of the bytes as fetched, so a figure can be tied to the exact
	// document that produced it.
	SHA256  string
	Fetched time.Time
}

// Read loads a corpus's manifest. A directory with no manifest is an empty
// corpus rather than an error: that is what a corpus about to be built looks
// like.
func Read(dir string) ([]Entry, error) {
	f, err := os.Open(filepath.Join(dir, ManifestName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for line := 0; sc.Scan(); line++ {
		text := sc.Text()
		if line == 0 && strings.HasPrefix(text, "path\t") {
			continue // the header
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		e, err := parse(text)
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", ManifestName, line+1, err)
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// parse reads one manifest row.
func parse(line string) (Entry, error) {
	f := strings.Split(line, "\t")
	if len(f) != 6 {
		return Entry{}, fmt.Errorf("%d fields, want 6", len(f))
	}
	n, err := strconv.ParseInt(f[3], 10, 64)
	if err != nil {
		return Entry{}, fmt.Errorf("size: %w", err)
	}
	when, err := time.Parse(time.RFC3339, f[5])
	if err != nil {
		return Entry{}, fmt.Errorf("time: %w", err)
	}
	return Entry{Path: f[0], Origin: f[1], Source: f[2], Bytes: n, SHA256: f[4], Fetched: when}, nil
}

// Write records a corpus, sorted by path so that two runs that fetched the same
// documents in a different order produce the same file.
func Write(dir string, entries []Entry) error {
	sorted := append([]Entry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var b strings.Builder
	b.WriteString("path\torigin\tsource\tbytes\tsha256\tfetched\n")
	for _, e := range sorted {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%d\t%s\t%s\n",
			e.Path, e.Origin, e.Source, e.Bytes, e.SHA256, e.Fetched.UTC().Format(time.RFC3339))
	}
	return os.WriteFile(filepath.Join(dir, ManifestName), []byte(b.String()), 0o644)
}

// Origins counts the documents in each population, which is the number any
// prevalence has to be divided by.
func Origins(entries []Entry) map[string]int {
	out := map[string]int{}
	for _, e := range entries {
		out[e.Origin]++
	}
	return out
}

// Digest is the SHA-256 of what r yields, and how many bytes that was.
func Digest(r io.Reader, w io.Writer) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(h, w), r)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
