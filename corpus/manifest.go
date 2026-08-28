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
	// document that produced it. Empty when the manifest did not say.
	SHA256 string
	// Fetched is when it was taken. The zero time means the manifest did not
	// say — some documents predate the record that describes them.
	Fetched time.Time
}

// Read loads a corpus's manifest.
//
// The header says what the columns are, so a manifest written by an earlier
// tool is still readable: the forms corpus gathered before this repository
// existed says "issuer|file|url|bytes|sha256-8|fetched", which is the same six
// facts under other names. Making the format self-describing was cheaper than
// re-fetching two thousand documents, and a corpus that has to be re-fetched to
// be read is not much of a record.
//
// A directory with no manifest is an empty corpus rather than an error: that is
// what a corpus about to be built looks like.
func Read(dir string) ([]Entry, error) {
	f, err := os.Open(filepath.Join(dir, ManifestName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	if !sc.Scan() {
		return nil, sc.Err()
	}
	cols, err := columns(sc.Text())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestName, err)
	}

	var out []Entry
	for line := 2; sc.Scan(); line++ {
		text := sc.Text()
		if strings.TrimSpace(text) == "" {
			continue
		}
		e, err := cols.parse(text)
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", ManifestName, line, err)
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// A layout is where each fact sits in a row, worked out from the header.
type layout struct{ path, origin, source, bytes, sha, fetched int }

// known maps every column name any manifest of ours has used onto the fact it
// carries.
var known = map[string]func(*layout, int){
	"path":     func(l *layout, i int) { l.path = i },
	"file":     func(l *layout, i int) { l.path = i },
	"origin":   func(l *layout, i int) { l.origin = i },
	"issuer":   func(l *layout, i int) { l.origin = i },
	"source":   func(l *layout, i int) { l.source = i },
	"url":      func(l *layout, i int) { l.source = i },
	"bytes":    func(l *layout, i int) { l.bytes = i },
	"sha256":   func(l *layout, i int) { l.sha = i },
	"sha256-8": func(l *layout, i int) { l.sha = i },
	"fetched":  func(l *layout, i int) { l.fetched = i },
}

// columns reads the header. Every fact must be named: a manifest that does not
// say where its documents came from is a list, and the point of a manifest is
// that it is not one.
func columns(header string) (layout, error) {
	l := layout{-1, -1, -1, -1, -1, -1}
	for i, name := range strings.Split(header, "\t") {
		if set, ok := known[strings.TrimSpace(name)]; ok {
			set(&l, i)
		}
	}
	for _, c := range []struct {
		at   int
		name string
	}{
		{l.path, "path"}, {l.origin, "origin"}, {l.source, "source"},
		{l.bytes, "bytes"}, {l.sha, "sha256"}, {l.fetched, "fetched"},
	} {
		if c.at < 0 {
			return l, fmt.Errorf("the header names no %s column: %q", c.name, header)
		}
	}
	return l, nil
}

// at reads one field, or reports that the row is short.
func at(f []string, i int) (string, error) {
	if i >= len(f) {
		return "", fmt.Errorf("%d fields, and the header names %d", len(f), i+1)
	}
	return f[i], nil
}

// parse reads one row by the layout the header declared.
func (l layout) parse(line string) (Entry, error) {
	f := strings.Split(line, "\t")
	var e Entry
	var err error
	if e.Path, err = at(f, l.path); err != nil {
		return e, err
	}
	if e.Origin, err = at(f, l.origin); err != nil {
		return e, err
	}
	if e.Source, err = at(f, l.source); err != nil {
		return e, err
	}
	if e.SHA256, err = at(f, l.sha); err != nil {
		return e, err
	}
	size, err := at(f, l.bytes)
	if err != nil {
		return e, err
	}
	if e.Bytes, err = strconv.ParseInt(size, 10, 64); err != nil {
		return e, fmt.Errorf("size: %w", err)
	}
	when, err := at(f, l.fetched)
	if err != nil {
		return e, err
	}
	// A fact the manifest does not give is recorded as absent rather than
	// refused. The forms corpus has rows whose digest is "-" and whose date is
	// free text, because those documents were already on disk when it was
	// assembled; throwing the whole record away over that would leave two
	// thousand documents with no provenance at all, which is worse than a
	// missing date. What must be there is where a document is, which
	// population it belongs to and where it came from.
	e.Fetched, _ = parseTime(when)
	if e.SHA256 == "-" {
		e.SHA256 = ""
	}
	// An older manifest keeps the population in one column and the bare file
	// name in another, the population being the directory. Rejoin them so a
	// path is a path whichever tool wrote it.
	if e.Origin != "" && !strings.ContainsRune(e.Path, filepath.Separator) {
		e.Path = filepath.Join(e.Origin, e.Path)
	}
	return e, nil
}

// parseTime accepts the shapes a manifest has been written in, and reports the
// zero time for anything else — which Entry.Fetched documents as "the manifest
// did not say".
func parseTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
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
