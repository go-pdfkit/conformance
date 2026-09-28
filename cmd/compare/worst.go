package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-pdfkit/conformance/compare"
)

// confirmWorst draws each page of the worst-by-ratio pool again and keeps the
// minimum of the tries, then re-sorts and returns the first keep of them.
//
// It exists because a list selected on the MAXIMUM of one sample per page is
// biased, and the bias was measured rather than feared. Of the eight pages this
// project published as its worst, three were inflated when drawn again --
// 1 831 ms that is really 688, 288 ms that is really 144, 253 ms that is really
// 145 -- while the other five were right to within 2%. That is not bad luck. A
// top-N by ratio built from single samples preferentially picks the pages whose
// sample came out unluckily high, and leaves out the ones whose sample came out
// luckily low, so the published table overstates its own worst rows AND omits
// pages that belong in them. Redrawing the pool fixes both halves; that is why
// Summarise hands back a pool rather than the five a report prints.
//
// The minimum of each side is taken independently. A minimum is the estimator an
// accident cannot inflate, and the question each column answers is what that
// renderer costs -- not what the two cost in the same unlucky instant. Both are
// drawn back to back in every try, so neither gets a quieter machine than the
// other.
//
// draw is a seam so a test need not have poppler, and a page it cannot draw again
// keeps the reading it came in with: an answer nobody obtained must not read as a
// negative one.
func confirmWorst(pool []compare.Result, tries, keep int, draw func(path string, page int) (ours, theirs time.Duration, ok bool)) []compare.Result {
	out := make([]compare.Result, 0, len(pool))
	for _, r := range pool {
		var bo, bt time.Duration
		any := false
		for i := 0; i < tries; i++ {
			ours, theirs, ok := draw(r.Path, r.Page)
			if !ok || ours <= 0 || theirs <= 0 {
				continue
			}
			if !any || ours < bo {
				bo = ours
			}
			if !any || theirs < bt {
				bt = theirs
			}
			any = true
		}
		if any {
			r.Ours, r.Theirs = bo, bt
		}
		out = append(out, r)
	}
	// Re-sorted because confirming changes the order, which is the whole point:
	// a page can leave the printed list and another can enter it.
	sort.Slice(out, func(i, j int) bool {
		a := float64(out[i].Ours) / float64(out[i].Theirs)
		b := float64(out[j].Ours) / float64(out[j].Theirs)
		if a != b {
			return a > b
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Page < out[j].Page
	})
	if len(out) > keep {
		out = out[:keep]
	}
	return out
}

// writeTimings writes one population's rows, with the CONFIRMED duration for the
// rows that were drawn again.
//
// It exists because the file and the report disagreed, and the file was wrong. The
// walk records one timing per page; confirmWorst then draws the named rows again and
// the report prints those. Nothing carried the second reading back, so anything built
// from the file kept the selection bias the confirmation removes -- and that is not
// hypothetical. At render v0.60.0 the walk read `bulletindedepart23dutc.pdf` at
// 1 219 ms, which is 7.7 times what the page costs: re-measured it is 158 ms, and it
// was 174 ms one release earlier. The report dropped it. The file kept it, and §24's
// "Where we lose" table, which is built from the file, would have published it as the
// corpus's worst page at 3.05x.
//
// So the rows are held until the confirmation has run. MOST ROWS ARE STILL ONE
// TIMING: only the ones a report names are drawn again, because drawing 3 208 pages
// three times more would cost an hour to improve numbers that a sum over 3 208 pages
// barely moves. The file's README says which is which.
func writeTimingRows(w io.Writer, name string, rs []compare.Result, confirmed []compare.Result) {
	better := make(map[string]compare.Result, len(confirmed))
	for _, c := range confirmed {
		better[key(name, filepath.Base(c.Path), fmt.Sprint(c.Page))] = c
	}
	for _, r := range rs {
		ours, theirs := r.Ours, r.Theirs
		if c, ok := better[key(name, filepath.Base(r.Path), fmt.Sprint(r.Page))]; ok {
			ours, theirs = c.Ours, c.Theirs
		}
		// "-" rather than an empty field: a row that ends in a tab is a row whose
		// last column a reader cannot tell from a missing one.
		hung := r.Tool
		if hung == "" {
			hung = "-"
		}
		// SHARE, and it is not decoration. A page one side did not draw has Share
		// -1, and its duration is the time taken to decline rather than to render:
		// reading a timing without it counts a refusal as a very fast page. Three
		// 12 MB scans in this corpus come back in 4 to 15 ms against poppler's 1.7
		// to 2.8 SECONDS, and a table that cannot see Share calls each of them a
		// 0.004x win.
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%.6f\t%s\n",
			name, filepath.Base(r.Path), r.Page, ours, theirs, r.Share, hung)
	}
}
