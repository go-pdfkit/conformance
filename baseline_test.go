// Copyright (c) 2026, the go-pdfkit/conformance authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A record is the part of a baseline JSON file this checks.
type record struct {
	Taken   string `json:"taken"`
	Modules []struct {
		Path    string `json:"path"`
		Version string `json:"version"`
	} `json:"modules"`
	Populations []struct {
		Population string `json:"population"`
	} `json:"populations"`
}

// TestTheBaselineReadmeDescribesTheRecordsBesideIt binds the prose to the data.
//
// baseline/README.md is written by hand from baseline/*.json, and the two have
// drifted before: a run left the module versions in the Conditions table naming
// the version the PREVIOUS run used, which is the same class of defect as a
// comment stating a rule the code ignores -- a reader has no way to tell.
//
// This does not check the tables' arithmetic, which is what the records are
// for. It checks the two things that go stale silently: that every population
// measured is named in the file, and that the module versions the file claims
// are the ones the records were taken with.
func TestTheBaselineReadmeDescribesTheRecordsBesideIt(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("baseline", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no baseline records: %v", err)
	}
	readme, err := os.ReadFile(filepath.Join("baseline", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)

	versions := map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var r record
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, p := range r.Populations {
			// The rows are written as `population`, in backticks.
			if !strings.Contains(text, "`"+p.Population+"`") {
				t.Errorf("%s measured population %q and the README never names it",
					filepath.Base(f), p.Population)
			}
		}
		for _, m := range r.Modules {
			if was, seen := versions[m.Path]; seen && was != m.Version {
				t.Errorf("the records disagree about %s: %s and %s -- they are not one run",
					m.Path, was, m.Version)
			}
			versions[m.Path] = m.Version
		}
	}

	// Only the modules the README names are checked: it lists the ones whose
	// version can change what is measured, and adding one to the records
	// should not fail this.
	//
	// It names them by a SUFFIX of the path -- `go-pdfkit/render`, not the
	// whole thing and not just `render`. Looking only for the full path and
	// the last segment found neither, so this loop skipped every module and
	// could not fail: it passed with the README naming v0.21.0 while every
	// record said v0.22.0, which is the exact drift it was written for.
	checked := 0
	for path, want := range versions {
		line, ok := row(text, path)
		if !ok {
			continue
		}
		checked++
		// On ITS OWN LINE, not anywhere in the file. The README quotes other
		// versions in its prose -- the run it replaces is named beside this
		// one in the summary table -- so a search over the whole text is
		// satisfied by a sentence about a different run and cannot fail.
		if !strings.Contains(line, want) {
			t.Errorf("the records were taken with %s %s, and the README's row for it reads %q",
				path, want, strings.TrimSpace(line))
		}
	}
	// A pass that checked nothing is not a pass.
	if checked == 0 {
		t.Errorf("the README names none of the %d modules the records list", len(versions))
	}
}

// row returns the README line that names a module, matching any suffix of its
// path in backticks -- `github.com/go-pdfkit/render`, `go-pdfkit/render` or
// `render`, whichever the file happens to use.
func row(text, path string) (string, bool) {
	for {
		want := "`" + path + "`"
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, want) {
				return line, true
			}
		}
		i := strings.IndexByte(path, '/')
		if i < 0 {
			return "", false
		}
		path = path[i+1:]
	}
}
