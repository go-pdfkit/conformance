// Copyright (c) the go-pdfkit authors.
// SPDX-License-Identifier: BSD-3-Clause

// Package parity measures how much of another toolkit's published tool list
// this fleet can be SHOWN to do.
//
// ⛔ It refuses to report a figure if any claim cannot be verified. A parity
// number assembled from a mapping nobody checks is a vibe with a decimal point
// in it. The first version of this measurement, kept by hand, was wrong by
// under-counting, and the only reason anybody found out is that somebody said
// "not sure that map is up to date".
//
// Two kinds of claim, and each is checked against something that was BUILT:
//
//   - "verb:NAME" — the verb has to appear in a command's own help output,
//     read off a binary compiled from the source being measured. Not from a
//     README, not from anybody's notes.
//   - "REPO#SYMBOL" — the symbol has to appear in that repository's Go source.
//
// A tool with no claim counts as NOT covered. That is the conservative
// direction, and it is deliberate: this must never make the fleet look better
// than it is.
package parity

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// A Claim is what a repository is said to do, and how to check it.
type Claim struct {
	Tool  string // the other toolkit's own name for it
	Claim string // "" means we do not do this
}

// Report is the outcome of a measurement.
type Report struct {
	Tools     []string // every tool the other toolkit documents
	Covered   []string // those a verified claim answers
	Uncovered []string // those nothing claims
}

// Percent is the share covered. It is only ever reached when every claim
// verified, because [Measure] returns an error otherwise.
func (r Report) Percent() float64 {
	if len(r.Tools) == 0 {
		return 0
	}
	return 100 * float64(len(r.Covered)) / float64(len(r.Tools))
}

// ReadTools lists the other toolkit's tools from its own documentation
// directory: one Markdown file per tool, which is how BentoPDF publishes them.
//
// ⛔ Reading its list rather than restating it is the point. A list typed out
// here would be a snapshot of what somebody believed on the day they typed it,
// and the denominator of a parity figure is exactly the thing that must not
// quietly drift.
func ReadTools(dir fs.FS, skip ...string) ([]string, error) {
	es, err := fs.ReadDir(dir, ".")
	if err != nil {
		return nil, fmt.Errorf("reading the tool list: %w", err)
	}
	ignore := map[string]bool{}
	for _, s := range skip {
		ignore[s] = true
	}
	var out []string
	for _, e := range es {
		if e.IsDir() || path.Ext(e.Name()) != ".md" {
			continue
		}
		n := strings.TrimSuffix(e.Name(), ".md")
		if ignore[n] {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		// ⛔ Said out loud. A directory that listed nothing is a failure of the
		// instrument, and reporting "0 tools, 100% parity" from it would be the
		// most flattering possible reading of a broken measurement.
		return nil, errors.New("the tool list is empty, which is a failure rather than a result")
	}
	sort.Strings(out)
	return out, nil
}

// Verbs reads a command's verbs off its own help output: the lines indented by
// exactly two spaces whose first word is lower-case.
//
// It takes a reader rather than running anything, so that the shape of the help
// text is testable without a binary and the library never execs.
func Verbs(help io.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(help)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			continue
		}
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == strings.ToLower(f[0]) {
			out = append(out, f[0])
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading the help output: %w", err)
	}
	if len(out) == 0 {
		return nil, errors.New("that help output names no verbs: the binary is wrong, or its help changed shape")
	}
	return out, nil
}

// Verify checks one claim. The error says what could not be shown.
func Verify(claim string, verbs []string, src fs.FS) error {
	if v, ok := strings.CutPrefix(claim, "verb:"); ok {
		for _, have := range verbs {
			if have == v {
				return nil
			}
		}
		return fmt.Errorf("the verb %q is not in the binary", v)
	}
	repo, sym, ok := strings.Cut(claim, "#")
	if !ok || repo == "" || sym == "" {
		return fmt.Errorf("a claim is either verb:NAME or REPO#SYMBOL, not %q", claim)
	}
	dir := path.Base(repo)
	if _, err := fs.Stat(src, dir); err != nil {
		return fmt.Errorf("%s is not checked out, so this claim cannot be shown either way", repo)
	}
	found := false
	// ⛔ A walk error is PROPAGATED, not swallowed. Returning nil for it would
	// turn "this repository could not be read" into "this repository does not
	// contain X" — a false negative that reads exactly like a real gap, which
	// is the direction a capability measurement must never fail in.
	err := fs.WalkDir(src, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), sym) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking %s: %w", repo, err)
	}
	if !found {
		return fmt.Errorf("%s does not contain %q", repo, sym)
	}
	return nil
}

// Measure checks every claim and counts what is covered.
//
// It refuses in three ways before it counts anything, and the three are
// different mistakes:
//
//   - a tool claimed twice. The map keeps the last one and the total comes out
//     short of what the table appears to say, SILENTLY — which is how this
//     check came to exist: ten entries were added and the figure rose by nine.
//   - a claim for a tool the other toolkit does not have. It inflates nothing
//     by itself, but it means the table is being written against something
//     other than the list, and that error has a direction nobody notices.
//   - a claim that cannot be shown. An unverified claim is not a capability.
func Measure(tools []string, table []Claim, verbs []string, src fs.FS) (Report, error) {
	have := map[string]bool{}
	for _, t := range tools {
		have[t] = true
	}

	byTool := map[string]string{}
	var dup, stray []string
	for _, c := range table {
		if _, seen := byTool[c.Tool]; seen {
			dup = append(dup, c.Tool)
		}
		if !have[c.Tool] {
			stray = append(stray, c.Tool)
		}
		byTool[c.Tool] = c.Claim
	}
	if len(dup) > 0 {
		sort.Strings(dup)
		return Report{}, fmt.Errorf("claimed twice: %s", strings.Join(dup, " "))
	}
	if len(stray) > 0 {
		sort.Strings(stray)
		return Report{}, fmt.Errorf("claimed but not in the tool list: %s", strings.Join(stray, " "))
	}

	r := Report{Tools: tools}
	var unproven []string
	for _, t := range tools {
		claim := byTool[t]
		if claim == "" {
			r.Uncovered = append(r.Uncovered, t)
			continue
		}
		if err := Verify(claim, verbs, src); err != nil {
			unproven = append(unproven, fmt.Sprintf("%s → %s: %v", t, claim, err))
			continue
		}
		r.Covered = append(r.Covered, t)
	}
	if len(unproven) > 0 {
		return Report{}, fmt.Errorf("%d claim(s) could not be shown, so no figure is reported:\n  %s",
			len(unproven), strings.Join(unproven, "\n  "))
	}
	return r, nil
}
