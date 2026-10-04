package corpus

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// A Problem is one way a corpus and the manifest beside it disagree.
type Problem struct {
	// Path is relative to the corpus directory, as the manifest writes it.
	Path string
	// Kind is why: "missing", "unreadable", "size", "digest" or
	// "unrecorded".
	Kind string
	// Want and Got say what the manifest records and what is on disk. Both
	// are empty for a kind where there is nothing to compare.
	Want, Got string
}

func (p Problem) String() string {
	switch p.Kind {
	case "missing":
		return fmt.Sprintf("%s: in the manifest and not on disk", p.Path)
	case "unrecorded":
		return fmt.Sprintf("%s: on disk and in no manifest row", p.Path)
	case "unreadable":
		return fmt.Sprintf("%s: on disk and will not open: %s", p.Path, p.Got)
	default:
		return fmt.Sprintf("%s: the manifest says %s %s, the file is %s",
			p.Path, p.Kind, p.Want, p.Got)
	}
}

// Check reports every way a corpus directory and its manifest disagree.
//
// The package comment says the manifest is what makes a corpus a measurement
// rather than a pile, "so a figure quoted from it can be reproduced and a file
// that changed underneath can be noticed". Nothing noticed: the hash was
// recorded and never read back, and a file in no row was invisible to every
// tool here.
//
// It is not a hypothetical. Measured 2026-10-04 over the two corpora this
// repository is run against, FOUR files of `pdfscans` were on disk and in no
// manifest row -- and three of those four are documents our reader refuses, so
// `images` and `compare`, which walk the manifest, counted 63 refusals where a
// sweep that walked the directory counted 66. Two true numbers for one corpus,
// and no way to tell which a figure came from. See baseline/README.md §33.
//
// The digest is compared over AS MANY CHARACTERS AS THE MANIFEST RECORDS. The
// forms corpus, gathered before this repository existed, keeps sixteen of them
// under the header "sha256-8"; the scans corpus keeps all sixty-four. Sixteen
// hex characters is sixty-four bits, which answers "did this file change"
// perfectly well, and a check that demanded all sixty-four would call every one
// of 2 268 rows changed and be deleted by the first person to run it.
func Check(dir string) ([]Problem, error) {
	entries, err := Read(dir)
	if err != nil {
		return nil, err
	}
	var out []Problem
	recorded := make(map[string]bool, len(entries))
	for _, e := range entries {
		recorded[e.Path] = true
		full := filepath.Join(dir, e.Path)
		info, err := os.Stat(full)
		if err != nil {
			out = append(out, Problem{Path: e.Path, Kind: "missing"})
			continue
		}
		if e.Bytes > 0 && info.Size() != e.Bytes {
			out = append(out, Problem{Path: e.Path, Kind: "size",
				Want: fmt.Sprint(e.Bytes), Got: fmt.Sprint(info.Size())})
			// The digest is not read as well: a file of another size has
			// another digest by construction, and saying so twice about one
			// file makes a count of problems a count of nothing.
			continue
		}
		if e.SHA256 == "" {
			// Some documents predate the record that describes them, which
			// Entry.SHA256 already says. There is nothing to check.
			continue
		}
		f, err := os.Open(full)
		if err != nil {
			// NOT "missing": the file is there and its bytes cannot be read,
			// which is a different thing to go and look at. Reporting it as
			// missing would send a reader to re-fetch a document that is
			// already on the disk.
			out = append(out, Problem{Path: e.Path, Kind: "unreadable", Got: err.Error()})
			continue
		}
		sum, _, err := Digest(f, io.Discard)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Path, err)
		}
		if n := len(e.SHA256); !strings.EqualFold(sum[:min(n, len(sum))], e.SHA256) {
			out = append(out, Problem{Path: e.Path, Kind: "digest",
				Want: e.SHA256, Got: sum[:min(n, len(sum))]})
		}
	}
	// And the other direction, which is the one nothing here could see.
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".pdf") {
			return nil
		}
		// Trimmed rather than filepath.Rel'd: WalkDir hands back paths that
		// begin with the root it was given, so there is no case where Rel
		// fails and an error branch nothing can reach is a branch nobody can
		// test.
		rel := strings.TrimPrefix(p, dir+string(filepath.Separator))
		if !recorded[rel] {
			out = append(out, Problem{Path: rel, Kind: "unrecorded"})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}
