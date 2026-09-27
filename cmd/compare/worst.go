package main

import (
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
