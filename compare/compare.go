// Package compare draws a page twice — once with go-pdfkit and once with an
// implementation that is not ours — and says how far apart the two pictures
// are.
//
// It exists because we are not a fit judge of our own output. A file our own
// reader reads back perfectly can draw nothing anywhere else; the way that was
// found was to ask poppler, and the way it stays found is to keep asking.
//
// Exact equality is not the question — two rasterisers disagree on every edge
// pixel — so what is measured is the share of pixels differing by more than a
// quarter of the range once both are reduced to grey and blurred slightly,
// which is what "the same page" means to somebody looking at it.
//
// # The judge is bounded, and a hang is named
//
// A poppler tool can hang on a document this corpus holds; the reason and the
// bound are in internal/poppler. So pdftoppm runs under it, and a page it does
// not finish is named with the tool that hung — Result.Tool and Summary.Hung —
// rather than dropped or retried. A named timeout is data about the corpus; a
// silent one is a gap a reader cannot tell from a page that scored badly.
package compare

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-gfx/gfx/raster"
	"github.com/go-pdfkit/conformance/internal/poppler"
	"github.com/go-pdfkit/reader"
	"github.com/go-pdfkit/render"
)

// A Result is one page judged.
type Result struct {
	Path string
	Page int
	// Share is the fraction of pixels that differ materially, or -1 when the
	// two could not be compared at all.
	Share float64
	// Note says why, when they could not be.
	Note string
	// Ours and Theirs are how long each renderer took, which is the other
	// thing worth knowing: a page drawn in no time may be a page drawn blank.
	Ours, Theirs time.Duration
	// Identical is the fraction of pixels that are the same BYTE FOR BYTE,
	// which is the strictest reading there is, and Mean is the mean absolute
	// per-channel difference in levels of 255. Both are -1 when the pages
	// could not be compared.
	//
	// At [Options.Super] of 1 these say little: two rasterisers disagree
	// about the sampling of every edge, so a seventh of an ordinary page is
	// not identical and none of that seventh is a defect. Drawn oversize and
	// reduced they become sharp -- 97.1% identical at 8x on the page those
	// figures came from -- and then what is NOT identical is worth opening.
	Identical, Mean float64
	// Colour is how far apart the two pages are in COLOUR, which Share
	// cannot say. Share reduces both to grey and asks how many pixels are
	// more than a quarter of the range apart, so a page tinted by nineteen
	// levels reads as nought -- and so does a scanned page that lost its
	// whole ink layer, which moves the page average by twelve levels and
	// almost no single pixel by sixty-four. Measured: such a page scored
	// 0.14% on Share.
	//
	// It is the mean per-channel difference inside the WORST square of
	// [tile] pixels, in levels of 255, and -1 when the pages could not be
	// compared.
	//
	// It is a MAGNITUDE and not a criterion. Share stays the pass criterion,
	// because "is this the same page to somebody looking at it" is the
	// question the corpus is judged on; this is the term that moves when a
	// colour space is read differently or a layer goes missing, so that such
	// a change is visible at all rather than invisible by construction.
	Colour float64
	// ColourWorst is the worst single blurred per-channel difference on the
	// page, which says whether Colour is a tint over a whole square or one
	// hard edge inside it.
	ColourWorst float64
	// Tool is the poppler tool that did not finish within poppler.Timeout,
	// empty when none did. A page the judge would not answer about is not a
	// page that disagreed, and only the named ones can be gone and looked at.
	Tool string
}

// Options say how to draw.
type Options struct {
	// DPI both renderers are asked for.
	DPI float64
	// MaxDuration bounds our own renderer. The judge is bounded by
	// poppler.Timeout, which is a repository-wide setting rather than an
	// option because it is not a property of the comparison: it is the point
	// at which the other implementation is declared to have hung.
	MaxDuration time.Duration
	// Pages is how many of each document to judge; 0 means the first only.
	Pages int
	// Super is the factor both pages are drawn OVERSIZE by and then reduced
	// by before anything is compared. 0 and 1 both mean "draw at DPI and
	// compare what comes out".
	//
	// It exists because most of what two rasterisers disagree about is not
	// the page, it is the SAMPLING of the page: every glyph edge and every
	// curve falls between pixels, and each side rounds it its own way.
	// Drawing bigger and averaging back down makes both converge on the same
	// coverage, and what survives is content.
	//
	// Measured on one text-heavy form, pixels identical BIT FOR BIT between
	// the two renderers:
	//
	//	1x   85.8%   mean |diff| 9.37
	//	2x   90.4%   mean |diff| 5.07
	//	4x   94.0%   mean |diff| 2.69
	//	8x   97.1%   mean |diff| 1.90
	//
	// The cost is the square of it: 4x draws sixteen times the pixels. What
	// it buys is a comparison sharp enough to be read strictly, which at 1x
	// it is not -- a seventh of every page differs there, and none of that
	// seventh is a defect.
	Super int
}

// Compare judges one document, page by page.
//
// Both renderers are given the SAME box. pdftoppm shows the media box by
// default and this shows the crop box, which is what the specification
// prescribes — so -cropbox is passed, and without it a tenth of the pages
// cannot be compared at all because they come out different sizes.
func Compare(path string, opt Options) []Result {
	if opt.DPI == 0 {
		opt.DPI = 72
	}
	pages := opt.Pages
	if pages <= 0 {
		pages = 1
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return []Result{{Path: path, Share: -1, Note: "unreadable: " + err.Error()}}
	}
	d, err := reader.Open(b)
	if err != nil {
		return []Result{{Path: path, Share: -1, Note: "refused: " + err.Error()}}
	}
	if n := d.PageCount(); pages > n {
		pages = n
	}
	out := make([]Result, 0, pages)
	for p := 1; p <= pages; p++ {
		out = append(out, comparePage(d, path, p, opt))
	}
	return out
}

// comparePage judges one page.
func comparePage(d *reader.Document, path string, p int, opt Options) Result {
	r := Result{Path: path, Page: p, Share: -1}
	start := time.Now()
	super := opt.Super
	if super < 1 {
		super = 1
	}
	ours, err := render.Page(d, p, render.Options{DPI: opt.DPI * float64(super), MaxDuration: opt.MaxDuration})
	r.Ours = time.Since(start)
	if ours == nil {
		r.Note = "we drew nothing"
		if err != nil {
			r.Note += ": " + err.Error()
		}
		return r
	}
	theirs, took, hung, err := draw(path, p, opt.DPI*float64(super))
	r.Theirs = took
	if hung {
		r.Tool = "pdftoppm"
		r.Note = "hung: " + poppler.DidNotFinish("pdftoppm").Error()
		return r
	}
	if err != nil {
		r.Note = "they drew nothing: " + err.Error()
		return r
	}
	if theirs.W != ours.W || theirs.H != ours.H {
		r.Note = fmt.Sprintf("different sizes: %dx%d against %dx%d",
			ours.W, ours.H, theirs.W, theirs.H)
		return r
	}
	if super > 1 {
		ours, theirs = reduce(ours, super), reduce(theirs, super)
	}
	r.Share = difference(ours, theirs)
	r.Colour, r.ColourWorst = colourDifference(ours, theirs)
	r.Identical, r.Mean = exactness(ours, theirs)
	r.Note = ""
	return r
}

// popplerCommand is a variable so a test can stand in for the other renderer
// without one being installed. It answers whether the tool hung, and then
// whether it failed.
var popplerCommand = func(args ...string) (bool, error) {
	_, hung, err := poppler.Run("pdftoppm", args...)
	return hung, err
}

// draw draws one page with pdftoppm and reads the result back.
func draw(path string, page int, dpi float64) (*raster.Image, time.Duration, bool, error) {
	dir, err := os.MkdirTemp("", "compare")
	if err != nil {
		return nil, 0, false, err
	}
	defer os.RemoveAll(dir)
	stem := filepath.Join(dir, "p")
	start := time.Now()
	hung, err := popplerCommand("-cropbox", "-r", fmt.Sprint(int(dpi)),
		"-f", fmt.Sprint(page), "-l", fmt.Sprint(page), "-png", path, stem)
	took := time.Since(start)
	if hung {
		return nil, took, true, err
	}
	if err != nil {
		return nil, took, false, err
	}
	matches, _ := filepath.Glob(stem + "*.png")
	if len(matches) == 0 {
		return nil, took, false, fmt.Errorf("it wrote no picture")
	}
	f, err := os.Open(matches[0])
	if err != nil {
		return nil, took, false, err
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return nil, took, false, err
	}
	return raster.FromImage(im), took, false, nil
}

// difference is the share of pixels that differ materially once both pictures
// are reduced to grey and blurred a little.
func difference(a, b *raster.Image) float64 {
	ga := blur(grey(a), a.W, a.H)
	gb := blur(grey(b), b.W, b.H)
	bad := 0
	for i := range ga {
		d := int(ga[i]) - int(gb[i])
		if d < 0 {
			d = -d
		}
		if d > 64 {
			bad++
		}
	}
	if len(ga) == 0 {
		return 0
	}
	return float64(bad) / float64(len(ga))
}

// exactness is how much of the page is the same byte for byte, and how far
// apart the rest is. It is the reading a binary diff would give, which is
// worth having only once the sampling disagreement is out of the way: see
// [Options.Super].
func exactness(a, b *raster.Image) (identical, mean float64) {
	if a.W != b.W || a.H != b.H || a.W*a.H == 0 {
		return -1, -1
	}
	same, sum, n := 0, 0.0, a.W*a.H*3
	for i := range a.W * a.H {
		p := i * 4
		eq := true
		for c := range 3 {
			d := int(a.Pix[p+c]) - int(b.Pix[p+c])
			if d != 0 {
				eq = false
			}
			if d < 0 {
				d = -d
			}
			sum += float64(d)
		}
		if eq {
			same++
		}
	}
	return float64(same) / float64(a.W*a.H), sum / float64(n)
}

// reduce averages f by f blocks, which is how two rasterisers are made to
// agree about an EDGE: each converges on the same coverage once the pixel is
// bigger than the disagreement.
func reduce(src *raster.Image, f int) *raster.Image {
	w, h := src.W/f, src.H/f
	out := &raster.Image{W: w, H: h, Pix: make([]uint8, w*h*4)}
	for y := range h {
		for x := range w {
			var sum [4]int
			for dy := range f {
				for dx := range f {
					p := ((y*f+dy)*src.W + x*f + dx) * 4
					for c := range 4 {
						sum[c] += int(src.Pix[p+c])
					}
				}
			}
			o := (y*w + x) * 4
			for c := range 4 {
				out.Pix[o+c] = uint8(sum[c] / (f * f))
			}
		}
	}
	return out
}

// tile is the side of the square a page is divided into before its colour is
// judged.
//
// A page-level average cannot see a picture: one 320x290 picture on a 595x842
// page is 18% of it, so twenty levels inside that picture arrive as three
// levels of page average, and the edges of every glyph contribute more than
// that. Dividing the page and reporting the WORST square is what stops a local
// difference being averaged away -- the same reason this instrument reports a
// worst peak per picture and not only a median.
const tile = 32

// colourDifference is the colour disagreement between two pages: the mean
// per-channel difference inside the worst square of [tile] pixels, and the
// worst single blurred difference anywhere.
//
// Both pages are blurred first, because two rasterisers disagree completely on
// the edge pixels of every glyph and that disagreement is about SHAPE. The blur
// spreads an edge over its neighbours and leaves standing a difference that
// covers an area, which is what reading a colour space differently produces --
// and what a missing layer produces too.
//
// What this does NOT do is tell those two apart. A square of dense text drawn
// slightly differently and a square of tinted picture can reach the same number
// by different routes; the figure says "these squares are not the same colour",
// and which squares they are is a question for somebody looking.
func colourDifference(a, b *raster.Image) (worstTile, worst float64) {
	if a.W != b.W || a.H != b.H || a.W*a.H == 0 {
		return -1, -1
	}
	var ch [2][3][]uint8
	for c := range 3 {
		ch[0][c] = blur(channel(a, c), a.W, a.H)
		ch[1][c] = blur(channel(b, c), b.W, b.H)
	}
	for ty := 0; ty < a.H; ty += tile {
		for tx := 0; tx < a.W; tx += tile {
			sum, n := 0.0, 0
			for y := ty; y < ty+tile && y < a.H; y++ {
				for x := tx; x < tx+tile && x < a.W; x++ {
					i := y*a.W + x
					n++
					for c := range 3 {
						d := int(ch[0][c][i]) - int(ch[1][c][i])
						if d < 0 {
							d = -d
						}
						sum += float64(d)
						if float64(d) > worst {
							worst = float64(d)
						}
					}
				}
			}
			if m := sum / float64(n*3); m > worstTile {
				worstTile = m
			}
		}
	}
	return worstTile, worst
}

// channel lifts one of red, green and blue out of a picture.
func channel(img *raster.Image, c int) []uint8 {
	out := make([]uint8, img.W*img.H)
	for i := range out {
		out[i] = img.Pix[i*4+c]
	}
	return out
}

// grey reduces an image to one byte a pixel.
func grey(img *raster.Image) []uint8 {
	out := make([]uint8, img.W*img.H)
	for i := range out {
		p := i * 4
		out[i] = uint8((299*int(img.Pix[p]) + 587*int(img.Pix[p+1]) + 114*int(img.Pix[p+2])) / 1000)
	}
	return out
}

// blur is a three-by-three box: enough to forgive a rasteriser's edge pixels
// without forgiving a missing letter.
func blur(src []uint8, w, h int) []uint8 {
	out := make([]uint8, len(src))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sum, n := 0, 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					px, py := x+dx, y+dy
					if px < 0 || py < 0 || px >= w || py >= h {
						continue
					}
					sum += int(src[py*w+px])
					n++
				}
			}
			out[y*w+x] = uint8(sum / n)
		}
	}
	return out
}

// Summary is what a run of results says as a whole.
type Summary struct {
	Compared, NotCompared int
	Median, P90, P99, Max float64
	Under                 map[float64]int
	Notes                 map[string]int
	// IdenticalMean is the mean of [Result.Identical] over the pages
	// compared, and MeanDiff the mean of [Result.Mean].
	IdenticalMean, MeanDiff float64
	// ColourMean is the MEAN over the pages compared of each page's worst
	// square, and ColourMax the largest of them. Both are needed and the
	// difference between them matters: one page of 250 at fifty levels moves
	// the mean by a fifth of a level and is invisible in it, which is how a
	// scanned page drawn without its ink layer hid inside a population that
	// read 14.98 before the fix and 14.83 after.
	//
	// ColourWorst is the worst single blurred pixel any page reached.
	//
	// They are magnitudes beside the criterion, never a criterion.
	ColourMean, ColourMax, ColourWorst float64
	// Slowest is the longest our own renderer took on any one page.
	Slowest time.Duration
	// Over is how many pages took longer than the threshold given.
	Over int
	// Slow names the pages that took longest, worst first.
	//
	// A count of pages over a threshold says a corpus has a problem and gives
	// nobody a way to go and look at it. What is wanted is the document and
	// the page, so this keeps them.
	Slow []Result
	// Worst names the pages that disagree most, worst first.
	//
	// The same argument Slow makes: a count of pages over a threshold says a
	// corpus has a problem and gives nobody a way to go and look at it. This
	// file has reported "9 pages at or over 10%" without saying WHICH nine
	// since it was written, and a disagreement nobody can open is a number
	// rather than a finding.
	Worst []Result
	// Hung names every page the judge would not finish drawing. Unlike Slow
	// it is not capped: a cap is a way of dropping names, and a hang that is
	// not named is indistinguishable from a page that scored badly, which is
	// the whole reason the bound exists.
	Hung []Result
}

// slowKept is how many slow pages are named. Enough to see whether they are
// one document's doing or spread across a population, and few enough that a
// report of a bad run stays readable.
const slowKept = 5

// worstKept is how many disagreeing pages are named. More than slowKept
// because they are what gets looked at: a slow page is a symptom of one thing
// and a wrong page can be wrong in as many ways as there are features.
const worstKept = 12

// Summarise turns results into the distribution worth quoting.
func Summarise(rs []Result, slow time.Duration) Summary {
	s := Summary{Under: map[float64]int{}, Notes: map[string]int{}}
	var shares []float64
	for _, r := range rs {
		if r.Ours > s.Slowest {
			s.Slowest = r.Ours
		}
		if slow > 0 && r.Ours > slow {
			s.Over++
			s.Slow = append(s.Slow, r)
		}
		if r.Tool != "" {
			s.Hung = append(s.Hung, r)
		}
		if r.Share > 0 {
			s.Worst = append(s.Worst, r)
		}
		if r.Share < 0 {
			s.NotCompared++
			s.Notes[note(r.Note)]++
			continue
		}
		shares = append(shares, r.Share)
		if r.Identical >= 0 {
			s.IdenticalMean += r.Identical
			s.MeanDiff += r.Mean
		}
		if r.Colour >= 0 {
			s.ColourMean += r.Colour
			if r.Colour > s.ColourMax {
				s.ColourMax = r.Colour
			}
			if r.ColourWorst > s.ColourWorst {
				s.ColourWorst = r.ColourWorst
			}
		}
	}
	s.Compared = len(shares)
	if s.Compared > 0 {
		s.ColourMean /= float64(s.Compared)
		s.IdenticalMean /= float64(s.Compared)
		s.MeanDiff /= float64(s.Compared)
	}
	if len(shares) == 0 {
		return s
	}
	// Worst first, and then by document, so a rerun names them in the same
	// order when two pages took the same time.
	sort.Slice(s.Worst, func(i, j int) bool {
		if s.Worst[i].Share != s.Worst[j].Share {
			return s.Worst[i].Share > s.Worst[j].Share
		}
		if s.Worst[i].Path != s.Worst[j].Path {
			return s.Worst[i].Path < s.Worst[j].Path
		}
		return s.Worst[i].Page < s.Worst[j].Page
	})
	if len(s.Worst) > worstKept {
		s.Worst = s.Worst[:worstKept]
	}
	sort.Slice(s.Slow, func(i, j int) bool {
		if s.Slow[i].Ours != s.Slow[j].Ours {
			return s.Slow[i].Ours > s.Slow[j].Ours
		}
		if s.Slow[i].Path != s.Slow[j].Path {
			return s.Slow[i].Path < s.Slow[j].Path
		}
		return s.Slow[i].Page < s.Slow[j].Page
	})
	if len(s.Slow) > slowKept {
		s.Slow = s.Slow[:slowKept]
	}
	sort.Float64s(shares)
	at := func(p float64) float64 { return shares[int(p*float64(len(shares)-1))] }
	s.Median, s.P90, s.P99, s.Max = at(0.5), at(0.9), at(0.99), shares[len(shares)-1]
	for _, t := range []float64{0.01, 0.02, 0.05, 0.10} {
		for _, v := range shares {
			if v < t {
				s.Under[t]++
			}
		}
	}
	return s
}

// note shortens a reason to its kind, so a hundred variations of one failure
// are counted as one.
func note(s string) string {
	for _, sep := range []string{":", " against "} {
		if i := indexOf(s, sep); i > 0 {
			s = s[:i]
		}
	}
	return s
}

// indexOf is strings.Index without the import, kept local so this file's
// dependencies are the two renderers and nothing else.
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
