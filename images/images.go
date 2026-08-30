// Package images decodes each picture a page draws, with go-pdfkit and with an
// implementation that is not ours, and says how many came out the same.
//
// It exists because a page is a composition, and a composition hides its
// parts. Drawing a whole page and comparing it answers "does this page look
// right", which is the question that matters to a reader — but when the answer
// is "mostly", it cannot say WHICH picture was wrong, and a picture that is
// wholly wrong can move a page's number by a few percent and pass. That
// happened: go-pdfkit/render v0.12.0 shipped drawing scanned pages dark
// because what was measured was that ink appeared and not that the right ink
// appeared.
//
// So this compares picture against picture, and reports by the filter each was
// stored in, because "our CCITT is right and our JBIG2 is wrong" is the
// finding a per-page number cannot produce.
//
// Exact equality IS the question here, unlike the page comparison. There is no
// rasteriser between the codec and the pixels: two implementations of the same
// image format either agree bit for bit or one of them is wrong.
package images

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-gfx/gfx/raster"
	"github.com/go-pdfkit/reader"
	"github.com/go-pdfkit/render"
)

// A Result is one picture judged.
type Result struct {
	Path string
	Page int
	// Name is the resource name the page draws it by.
	Name string
	// Filter is the image format it was stored in, empty for plain samples.
	Filter string
	// Stencil says it is a one-bit mask rather than a picture.
	Stencil bool
	// Decoded says a /Decode array shaped our samples. pdfimages writes the
	// samples as stored, so this picture and theirs are not the same question
	// and are counted apart rather than as a disagreement.
	Decoded bool
	// Size is what the picture came out as.
	W, H int
	// Share is the fraction of pixels differing, or -1 when the picture could
	// not be judged, which Note explains. A Share of 1 with Inverted set means
	// the two are exact complements.
	Share float64
	// Inverted says the picture and theirs are the same everywhere, read the
	// other way round.
	//
	// pdfimages writes a JBIG2 mask's bitmap with the opposite polarity to the
	// one the samples have when that stream is used as a soft mask. That is a
	// convention, not a disagreement — and it is worth telling apart from a
	// real one, because "identical up to inversion" and "wrong" look the same
	// in a count of differing pixels and only one of them needs looking into.
	Inverted bool
	// Missing says which side had nothing, when nothing could be compared.
	// It is a field rather than a reading of Note because a baseline has to
	// count these apart: a run that agrees less because ours stopped opening
	// documents and a run that agrees less because it decoded them wrongly
	// are different events, and a total that lumps them says neither.
	Missing Missing
	Note    string
}

// A Missing says which side produced nothing.
type Missing string

const (
	// Judged means there was a picture on both sides and they were compared.
	Judged Missing = ""
	// Ours means ours would not open the document or draw the page.
	Ours Missing = "ours"
	// Theirs means the judge took no picture out of a page ours drew pictures
	// for. That is not a disagreement and not a fault of ours: it is a page
	// this cannot get an answer about, and one that must be counted so that a
	// corpus getting harder is not read as a decoder getting worse.
	Theirs Missing = "theirs"
)

// Options say how much to look at.
type Options struct {
	// Pages is how many pages of each document to take the pictures of; 0
	// means the first only.
	Pages int
}

// Judge takes the pictures out of one document twice and compares them.
func Judge(path string, opt Options) []Result {
	pages := opt.Pages
	if pages <= 0 {
		pages = 1
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return []Result{{Path: path, Share: -1, Missing: Ours, Note: "unreadable: " + err.Error()}}
	}
	d, err := reader.Open(b)
	if err != nil {
		return []Result{{Path: path, Share: -1, Missing: Ours, Note: "refused: " + err.Error()}}
	}
	if n := d.PageCount(); pages > n {
		pages = n
	}
	var out []Result
	for p := 1; p <= pages; p++ {
		out = append(out, judgePage(d, path, p)...)
	}
	return out
}

// judgePage judges the pictures of one page.
func judgePage(d *reader.Document, path string, p int) []Result {
	ours, err := render.Images(d, p)
	if err != nil {
		return []Result{{Path: path, Page: p, Share: -1, Missing: Ours, Note: "no page: " + err.Error()}}
	}
	if len(ours) == 0 {
		return nil
	}
	theirs, err := poppler(path, p)
	if err != nil {
		return []Result{{Path: path, Page: p, Share: -1, Missing: Theirs, Note: "they took nothing out: " + err.Error()}}
	}
	// The two do not agree on an order, and neither has a name the other
	// knows, so a picture is matched to a picture of the same size. Where
	// several share a size the first unclaimed one is taken, which is right
	// as often as it is wrong and is reported either way.
	claimed := make([]bool, len(theirs))
	out := make([]Result, 0, len(ours))
	for _, im := range ours {
		r := Result{Path: path, Page: p, Name: im.Name, Filter: im.Filter,
			Stencil: im.Stencil, Decoded: im.Decoded,
			W: im.Pic.W, H: im.Pic.H, Share: -1}
		j := match(theirs, claimed, im.Pic)
		if j < 0 {
			r.Note = "they took out nothing this size"
			out = append(out, r)
			continue
		}
		claimed[j] = true
		r.Share = difference(im.Pic, theirs[j])
		if r.Share == 1 {
			r.Inverted = true
		}
		out = append(out, r)
	}
	return out
}

// match finds an unclaimed picture of the same size.
func match(theirs []*raster.Image, claimed []bool, ours *raster.Image) int {
	for j, t := range theirs {
		if !claimed[j] && t.W == ours.W && t.H == ours.H {
			return j
		}
	}
	return -1
}

// difference is the fraction of pixels that are not the same ink.
//
// It reads only whether a pixel is dark, not how dark. The two sides disagree
// about depth and colour space in ways that are not errors: poppler writes a
// stencil out as one bit and this carries the shape in an alpha channel, and a
// CMYK picture comes back converted by two different sets of arithmetic. What
// both agree on, when they agree at all, is where the ink is.
func difference(a, b *raster.Image) float64 {
	if a.W != b.W || a.H != b.H || a.W*a.H == 0 {
		return -1
	}
	differ := 0
	for i := 0; i < a.W*a.H; i++ {
		if dark(a, i) != dark(b, i) {
			differ++
		}
	}
	return float64(differ) / float64(a.W*a.H)
}

// dark says whether one pixel is ink. A transparent pixel is paper whatever
// colour is under it, which is what makes a stencil comparable with the one
// bit poppler writes for the same mask.
func dark(im *raster.Image, i int) bool {
	r, g, b, a := im.Pix[i*4], im.Pix[i*4+1], im.Pix[i*4+2], im.Pix[i*4+3]
	if a < 128 {
		return false
	}
	return (uint32(r)*299+uint32(g)*587+uint32(b)*114)/1000 < 128
}

// popplerCommand is a variable so a test can stand in for the other
// implementation without one being installed.
var popplerCommand = func(args ...string) error { return exec.Command("pdfimages", args...).Run() }

// poppler takes the pictures out of one page with pdfimages.
//
// pdfimages EXTRACTS rather than renders, which is the whole point: asking
// pdftoppm would put poppler's rasteriser between the codec and the answer,
// and its resampling shows up as every decoder disagreeing with it slightly.
// Measured that way, four JBIG2 decoders all looked wrong; measured this way,
// one was exact on every stream and another was wrong on 91% of them.
func poppler(path string, page int) ([]*raster.Image, error) {
	dir, err := os.MkdirTemp("", "images")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	stem := filepath.Join(dir, "i")
	if err := popplerCommand("-png", "-f", fmt.Sprint(page), "-l", fmt.Sprint(page),
		path, stem); err != nil {
		return nil, err
	}
	// Glob answers with an error only for a pattern it cannot parse, and this
	// pattern is a temporary directory of our own making, so a failure here
	// leaves no names and is reported below as nothing having come out.
	names, _ := filepath.Glob(stem + "-*.png")
	// Glob's order is the filesystem's; the numbering is pdfimages's.
	sort.Strings(names)
	out := make([]*raster.Image, 0, len(names))
	for _, name := range names {
		im, err := readPNG(name)
		if err != nil {
			continue
		}
		out = append(out, im)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no pictures came out")
	}
	return out, nil
}

// readPNG reads one of the files pdfimages wrote.
func readPNG(name string) (*raster.Image, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return raster.FromImage(img), nil
}

// Counts is what a population came to, per filter.
type Counts struct {
	// Pictures is how many were judged.
	Pictures int
	// Exact is how many came out identical.
	Exact int
	// Unmatched is how many the other implementation had no picture for.
	Unmatched int
	// Inverted is how many came out as the other's exact complement.
	Inverted int
	// Remapped is how many carried a /Decode array, which we apply and
	// pdfimages does not, so the two sides were not asked the same question.
	// They are counted here rather than as disagreements: on a one-bit mask a
	// /Decode of [1 0] inverts every pixel, which reads as total disagreement
	// and is none.
	Remapped int
	// Shares holds the difference of each picture that differed.
	Shares []float64
}

// Tally groups results by the filter each picture was stored in.
func Tally(rs []Result) map[string]*Counts {
	by := map[string]*Counts{}
	for _, r := range rs {
		if r.Name == "" {
			continue // a whole document that could not be opened
		}
		key := r.Filter
		if key == "" {
			key = "(samples)"
		}
		if r.Stencil {
			key += " mask"
		}
		c := by[key]
		if c == nil {
			c = &Counts{}
			by[key] = c
		}
		c.Pictures++
		switch {
		case r.Decoded:
			c.Remapped++
		case r.Share < 0:
			c.Unmatched++
		case r.Share == 0:
			c.Exact++
		case r.Inverted:
			c.Inverted++
		default:
			c.Shares = append(c.Shares, r.Share)
		}
	}
	return by
}

// Report writes a tally in the order that reads best: worst first, because a
// filter that is right everywhere is not the one to read about.
func Report(by map[string]*Counts) string {
	var sb strings.Builder
	for _, k := range order(by) {
		c := by[k]
		fmt.Fprintf(&sb, "%-22s %5d pictures  %5d exact  %5d inverted  %5d differing  %5d unmatched  %5d remapped",
			k, c.Pictures, c.Exact, c.Inverted, len(c.Shares), c.Unmatched, c.Remapped)
		if len(c.Shares) > 0 {
			median, worst := spread(c.Shares)
			fmt.Fprintf(&sb, "  median %.4f  worst %.4f", median, worst)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// rightness is the share of a filter's comparable pictures that came out
// identical, and it decides what is read first.
//
// A picture that was never comparable is not evidence either way, so it is left
// out — the remapped ones and the ones that came back as an exact complement
// alike. A filter with NOTHING comparable is therefore not evidence at all, and
// counting it as nought right would put it at the top of the report, above
// every filter actually known to be wrong.
func rightness(c *Counts) float64 {
	comparable := c.Pictures - c.Remapped - c.Inverted
	if comparable <= 0 {
		return 1
	}
	return float64(c.Exact) / float64(comparable)
}

// A Summary is one population's tally in a shape that keeps: named fields, a
// settled order, and no map. A baseline is only worth writing down if a later
// run can be diffed against it, and a Go map ordered by chance cannot be.
type Summary struct {
	Population string `json:"population"`
	// Documents is how many were looked at, which is not how many were
	// judged: a document that draws no pictures contributes none.
	Documents int `json:"documents"`
	// Refused is how many documents ours would not open, or drew no page of.
	Refused int `json:"refused"`
	// Declined is how many pages ours drew pictures for and the judge took
	// none out of, so there was nothing to compare them with.
	Declined int            `json:"declined"`
	Filters  []FilterCounts `json:"filters"`
}

// FilterCounts is one filter's line of a report, as data.
type FilterCounts struct {
	Filter    string `json:"filter"`
	Pictures  int    `json:"pictures"`
	Exact     int    `json:"exact"`
	Inverted  int    `json:"inverted"`
	Differing int    `json:"differing"`
	Unmatched int    `json:"unmatched"`
	Remapped  int    `json:"remapped"`
	// Median and Worst are the differing pictures' shares, absent when none
	// differed. A pointer because 0.0 is a real answer here — a filter whose
	// worst disagreement is nought pixels is not the same as one with nothing
	// to disagree about — and omitempty cannot tell those apart.
	Median *float64 `json:"median,omitempty"`
	Worst  *float64 `json:"worst,omitempty"`
}

// Summarize turns a population's results into the record of it, worst filter
// first, which is the same order Report reads in.
func Summarize(population string, documents int, rs []Result) Summary {
	s := Summary{Population: population, Documents: documents}
	for _, r := range rs {
		switch r.Missing {
		case Ours:
			s.Refused++
		case Theirs:
			s.Declined++
		}
	}
	by := Tally(rs)
	for _, k := range order(by) {
		c := by[k]
		f := FilterCounts{Filter: k, Pictures: c.Pictures, Exact: c.Exact,
			Inverted: c.Inverted, Differing: len(c.Shares),
			Unmatched: c.Unmatched, Remapped: c.Remapped}
		if len(c.Shares) > 0 {
			median, worst := spread(c.Shares)
			f.Median, f.Worst = &median, &worst
		}
		s.Filters = append(s.Filters, f)
	}
	return s
}

// spread is the middle and the end of the differing shares.
func spread(shares []float64) (median, worst float64) {
	s := append([]float64(nil), shares...)
	sort.Float64s(s)
	return s[len(s)/2], s[len(s)-1]
}

// order puts the filters worst first, because a filter that is right
// everywhere is not the one to read about.
func order(by map[string]*Counts) []string {
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ai, bi := rightness(by[keys[i]]), rightness(by[keys[j]])
		if ai != bi {
			return ai < bi
		}
		return keys[i] < keys[j]
	})
	return keys
}
