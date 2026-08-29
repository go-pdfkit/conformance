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
package compare

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-gfx/gfx/raster"
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
}

// Options say how to draw.
type Options struct {
	// DPI both renderers are asked for.
	DPI float64
	// MaxDuration bounds our own renderer. Poppler is bounded by the caller's
	// patience.
	MaxDuration time.Duration
	// Pages is how many of each document to judge; 0 means the first only.
	Pages int
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
	ours, err := render.Page(d, p, render.Options{DPI: opt.DPI, MaxDuration: opt.MaxDuration})
	r.Ours = time.Since(start)
	if ours == nil {
		r.Note = "we drew nothing"
		if err != nil {
			r.Note += ": " + err.Error()
		}
		return r
	}
	theirs, took, err := poppler(path, p, opt.DPI)
	r.Theirs = took
	if err != nil {
		r.Note = "they drew nothing: " + err.Error()
		return r
	}
	if theirs.W != ours.W || theirs.H != ours.H {
		r.Note = fmt.Sprintf("different sizes: %dx%d against %dx%d",
			ours.W, ours.H, theirs.W, theirs.H)
		return r
	}
	r.Share = difference(ours, theirs)
	r.Note = ""
	return r
}

// popplerCommand is a variable so a test can stand in for the other renderer
// without one being installed.
var popplerCommand = func(args ...string) error { return exec.Command("pdftoppm", args...).Run() }

// poppler draws one page with pdftoppm and reads the result back.
func poppler(path string, page int, dpi float64) (*raster.Image, time.Duration, error) {
	dir, err := os.MkdirTemp("", "compare")
	if err != nil {
		return nil, 0, err
	}
	defer os.RemoveAll(dir)
	stem := filepath.Join(dir, "p")
	start := time.Now()
	err = popplerCommand("-cropbox", "-r", fmt.Sprint(int(dpi)),
		"-f", fmt.Sprint(page), "-l", fmt.Sprint(page), "-png", path, stem)
	took := time.Since(start)
	if err != nil {
		return nil, took, err
	}
	matches, _ := filepath.Glob(stem + "*.png")
	if len(matches) == 0 {
		return nil, took, fmt.Errorf("it wrote no picture")
	}
	f, err := os.Open(matches[0])
	if err != nil {
		return nil, took, err
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return nil, took, err
	}
	return raster.FromImage(im), took, nil
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
}

// slowKept is how many slow pages are named. Enough to see whether they are
// one document's doing or spread across a population, and few enough that a
// report of a bad run stays readable.
const slowKept = 5

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
		if r.Share < 0 {
			s.NotCompared++
			s.Notes[note(r.Note)]++
			continue
		}
		shares = append(shares, r.Share)
	}
	s.Compared = len(shares)
	if len(shares) == 0 {
		return s
	}
	// Worst first, and then by document, so a rerun names them in the same
	// order when two pages took the same time.
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
