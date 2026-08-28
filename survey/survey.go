// Package survey counts what a population of PDFs is actually made of.
//
// It exists because a decision was nearly taken on a number that measured the
// corpus rather than the world: JBIG2 appears in six of sixteen hundred
// government forms, which says only that government forms are not scanned
// books. What a survey has to answer is per population, and it has to
// distinguish a filter that appears from a filter whose absence leaves a page
// blank — those are very different amounts of harm.
package survey

import (
	"os"
	"sort"

	"github.com/go-pdfkit/reader"
)

// Counts is what one population is made of.
type Counts struct {
	// Documents is how many opened. Files is how many were looked at, so the
	// two together say how many refused to open at all.
	Files, Documents int
	// Pages is the total across the documents that opened.
	Pages int
	// UsedBy is, per image filter, how many DOCUMENTS carry at least one image
	// in it. This is the number a prevalence is quoted from.
	UsedBy map[string]int
	// Images is, per image filter, how many images there are. A single
	// document can hold hundreds, so this is not a prevalence.
	Images map[string]int
	// BlankWithout is, per image filter, how many PAGES have nothing on them
	// but images in that filter — the pages that come out blank when it is not
	// decoded. This is the harm, as against the presence.
	BlankWithout map[string]int
	// Refused is, per reason, how many documents would not open. A population's
	// refusals are part of what it is, and their REASONS decide whether they
	// are ours to fix: eleven per cent of scanned books refuse because they
	// carry Adobe's EBX lending DRM, which is not a defect and not something to
	// implement.
	Refused map[string]int
}

// newCounts makes the maps, so a caller never has to.
func newCounts() Counts {
	return Counts{
		UsedBy:       map[string]int{},
		Images:       map[string]int{},
		BlankWithout: map[string]int{},
		Refused:      map[string]int{},
	}
}

// Filters returns the filter names seen, in a stable order.
func (c Counts) Filters() []string { return keys(c.UsedBy) }

// Reasons returns the refusals seen, in a stable order.
func (c Counts) Reasons() []string { return keys(c.Refused) }

// keys is the names of a count, sorted, so two runs print the same thing.
func keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// marking are the operators that put something on a page. A page whose only
// marks are images is a scan; a page with text and lines on it is not, and
// failing to decode one of its pictures does not leave it blank.
var marking = map[string]bool{
	"S": true, "s": true, "f": true, "F": true, "f*": true,
	"B": true, "B*": true, "b": true, "b*": true, "sh": true,
	"Tj": true, "TJ": true, "'": true, "\"": true,
}

// Survey reads every document it is given and adds up what they are made of.
// A document that will not open is counted in Files and not in Documents: a
// population's refusals are part of what it is.
func Survey(paths []string, pagesPerDoc int) Counts {
	c := newCounts()
	for _, path := range paths {
		c.Files++
		if surveyOne(path, pagesPerDoc, &c) {
			c.Documents++
		}
	}
	return c
}

// surveyOne adds one document, and says whether it opened.
func surveyOne(path string, pagesPerDoc int, c *Counts) (opened bool) {
	defer func() {
		// A corpus is somebody else's bytes. One document that panics must not
		// end a survey of ten thousand.
		_ = recover()
	}()
	b, err := os.ReadFile(path)
	if err != nil {
		c.Refused[err.Error()]++
		return false
	}
	d, err := reader.Open(b)
	if err != nil {
		c.Refused[err.Error()]++
		return false
	}
	inThisDocument := map[string]bool{}
	pages := d.PageCount()
	if pagesPerDoc > 0 && pages > pagesPerDoc {
		pages = pagesPerDoc
	}
	for p := 1; p <= pages; p++ {
		c.Pages++
		surveyPage(d, p, c, inThisDocument)
	}
	for name := range inThisDocument {
		c.UsedBy[name]++
	}
	return true
}

// surveyPage counts one page's images and decides whether the page is nothing
// but them.
// The reader answers a dangling reference with Null and no error, and Page
// refuses only a number outside the document — which the loop above never asks
// for. So the errors here cannot happen and are dropped rather than handled: a
// branch that cannot be reached cannot be tested, and an untested branch in the
// code that decides what a measurement says is worse than no branch.
func surveyPage(d *reader.Document, p int, c *Counts, inThisDocument map[string]bool) {
	page, _ := d.Page(p)
	res, _ := d.Resolve(page["Resources"])
	rd, ok := reader.ToDict(res)
	if !ok {
		return
	}
	onThisPage := map[string]bool{}
	images := 0
	xo, _ := d.Resolve(rd["XObject"])
	if xd, ok := reader.ToDict(xo); ok {
		for _, v := range xd {
			name, ok := imageFilter(d, v)
			if !ok {
				continue
			}
			images++
			c.Images[name]++
			onThisPage[name] = true
			inThisDocument[name] = true
		}
	}
	if images == 0 || marks(d, p) > 0 {
		return
	}
	// Nothing on this page but pictures: it is a scan, and whichever filter
	// they are in is the one whose absence leaves it blank.
	for name := range onThisPage {
		c.BlankWithout[name]++
	}
}

// imageFilter answers with the filter an image XObject is encoded in, and
// whether v was an image at all. A chain ending in no image filter — a plain
// Flate bitmap — is reported as "" and counted like any other, because a
// population's raw images are part of what it is.
func imageFilter(d *reader.Document, v reader.Object) (string, bool) {
	o, _ := d.Resolve(v)
	st, ok := reader.ToStream(o)
	if !ok {
		return "", false
	}
	if sub, _ := reader.ToName(st.Dict["Subtype"]); sub != "Image" {
		return "", false
	}
	f, _ := d.Resolve(st.Dict["Filter"])
	if n, ok := reader.ToName(f); ok {
		return string(n), true
	}
	if arr, ok := reader.ToArray(f); ok && len(arr) > 0 {
		if n, ok := reader.ToName(arr[len(arr)-1]); ok {
			return string(n), true
		}
	}
	return "none", true
}

// marks counts the operators on a page that put something on it.
func marks(d *reader.Document, p int) int {
	ops, _ := d.PageOperations(p)
	n := 0
	for _, op := range ops {
		if marking[op.Operator] {
			n++
		}
	}
	return n
}
