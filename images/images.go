// Package images decodes each picture a page draws, with go-pdfkit and with an
// implementation that is not ours, and says how far apart the two came out.
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
// # The comparison is per channel, with a magnitude gate
//
// difference reads every pixel on both sides, subtracts them CHANNEL BY
// CHANNEL, and counts a pixel as differing only when some channel differs by
// more than Gate levels of 255. Share is that count over the pixels; Peak is
// the largest single-channel difference anywhere in the picture; and MSE and
// Mean are the aggregate error, in levels squared and in levels. A picture
// AGREES when Peak is at most Gate, which — with the count budget at zero —
// is the same statement as Share being nought, so the share is a report and
// never a budget. That is pdfium's shape (testing/image_diff.cpp:287-288,
// where a percentage is computed and then required to be zero) and the
// field's; the survey is in the repository README.
//
// # What was here before, and why it went
//
// Until conformance#16 the per-pixel predicate was a BISECTION: a pixel was
// ink when its alpha was at least 128 and its luminance below 128, and Share
// was the fraction of pixels whose ink classification differed. That is exact
// for a bilevel picture and blind everywhere else. A decoder rendering every
// pixel of a scan at luminance 120 where poppler renders 20 scored 0.000 —
// perfect agreement — on an error of 100 levels at every pixel. A systematic
// level or chroma shift is the characteristic failure of a lossy decoder, so
// the instrument was blind in exactly the direction the codecs fail.
//
// Every figure this package produced before that change was that narrower
// thing, so a record taken with it CANNOT be subtracted from a record taken
// with this one. baseline/README.md says which populations carry which.
//
// # Gate is 2, and the reason is ours rather than borrowed
//
// pdfium uses 3 (testing/utils/pixel_diff_util.h:11-12), but pdfium compares
// RENDERED PAGES, so its 3 buys slack for rasteriser and anti-aliasing
// differences. There is none to buy here: pdfimages EXTRACTS rather than
// renders, which is why this package asks it, so nothing sits between the
// codec and the pixels. What is left is codec rounding, and the standards
// bound that. ISO/IEC 10918-2 requires a conformant JPEG IDCT to be within one
// level per sample of the reference — read out of FFmpeg's implementation of
// that test, libavcodec/tests/dct.c:259:
//
//	spec_err = is_idct && (err_inf > 1 || omse > 0.02 || fabs(ome) > 0.0015);
//
// Two conformant decoders sit either side of that reference, so 2 is what they
// may legitimately differ by. Cairo's 25 (test/buffer-diff.c:43-44) is the
// ceiling no per-case exception may ever cross, for the reason Cairo states:
// "otherwise some problems could be masked".
//
// There is ONE gate and no per-case table, because there is no case yet.
// conformance#16 provides for raising Gate per population and per filter with
// a written reason; nothing has been measured that asks for it, and an
// exception mechanism with no exceptions in it is a promise rather than a
// measurement.
//
// # MSE and Mean are carried, and bounded by nothing
//
// pdfium bounds a mean squared error beside its per-channel delta
// (testing/image_diff.cpp:171-183, limit 0.05), and FFmpeg carries a SIGNED
// mean beside its MSE at dct.c:259. Both are recorded here, in the same units
// those projects use — levels squared and levels — so the published limits can
// be read against them directly.
//
// Neither is a pass criterion here, and that is deliberate. MSE catches
// accumulated noise and Mean catches BIAS, which is the failure the bisection
// could not see at all; but no bound on either has been MEASURED for pictures
// that were extracted rather than rendered, and adopting pdfium's limit for a
// different operation would repeat exactly the mistake the withdrawn 1%
// tolerance was: a number carried onto an instrument that did not produce it.
// They are reported so that a bound can be chosen from evidence later.
//
// # Colour conversion is not codec error, so it is counted apart
//
// Our side is RGBA from render; theirs is a PNG that poppler wrote. A CMYK, an
// ICC or a Lab picture reaches those two forms through two different sets of
// colour arithmetic, and per channel that difference is LARGE and is not a
// decoder disagreeing. Widening Gate to absorb it would destroy the gate.
//
// So each picture carries the colour space pdfimages reports for it
// (ImageOutputDev.cc:152-190 lists gray, rgb, cmyk, lab, icc, index, sep,
// devn, and "-" for a mask), and the pictures poppler had to CONVERT to reach
// RGB are tallied in their own bucket, with their own agreement figure and
// their own magnitudes — the way Remapped and Inverted are already counted
// apart. Only gray, rgb and "-" are treated as direct.
//
// Two honesties about that. index is counted as converted although its base
// space is often DeviceRGB, because pdfimages does not report the base and a
// picture that cannot be classified must not be counted as agreement. And
// poppler folds CalGray onto "gray" and CalRGB onto "rgb", so a few pictures
// counted direct did pass through a CIE conversion; that is poppler's
// resolution, not a claim of ours.
//
// A picture whose row could not be read at all also lands in the converted
// bucket, which makes a failure of pdfimages -list LOUD — every filter would
// read as wholly converted — rather than silently generous.
//
// # Alpha, and what a mask is compared in
//
// For an ordinary picture the compared channels are R, G and B. Alpha is not
// compared: pdfimages writes an opaque picture for anything that is not a
// mask, and writes a soft mask out as its own file, so a difference in alpha
// would be a difference in what the two tools chose to emit rather than in
// what the codecs produced.
//
// A MASK is not comparable channel for channel, because the two sides do not
// put it in the same channels, and render does not put it in ONE place
// either. Both were read out of the buffers rather than assumed:
//
//   - a /ImageMask true STENCIL carries no colour of its own — it paints
//     whatever colour is in force through its own shape — so render returns it
//     black with the shape in the ALPHA channel. us-opm/SF2801PR.pdf's Im0 is
//     one: 325x240, every RGB nought, exactly two alpha values.
//   - a /SMask is eight-bit greyscale, so render returns it OPAQUE with its
//     levels in RGB. us-opm/sf2822.pdf's Im0/SMask is one: 116x73, alpha 255
//     at every one of its 8468 pixels.
//
// poppler writes both as grey, with black where the mask paints. So a mask is
// compared in ONE derived channel, INK COVERAGE, by the same formula on both
// sides: alpha times 255 minus the luminance, over 255. On a stencil the
// luminance is nought so it reduces to the alpha; on a soft mask the alpha is
// 255 so it reduces to the inverted luminance; on poppler's side the alpha is
// always 255 and it is the inverted luminance. One formula, correct for both
// layouts, and it is what the old bisection was doing per bit.
//
// # Polarity stays its own signal
//
// pdfimages writes a stencil with the opposite polarity to the samples it
// holds. Under a magnitude measure that reads as maximal error at every pixel,
// which would lose a distinction worth keeping, so the complement is tested
// separately and in the same pass: Inverted says the direct comparison failed
// the gate and the complemented one passed it. A picture that passes directly
// is never called inverted, so a uniform mid-grey — whose complement is within
// the gate of itself — is reported as agreeing and not as a complement.
package images

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-gfx/gfx/raster"
	"github.com/go-pdfkit/reader"
	"github.com/go-pdfkit/render"
)

// Gate is D from conformance#16: a pixel counts as differing only when some
// compared channel differs by more than this many levels of 255.
//
// It is 2 because ISO/IEC 10918-2 allows a conformant JPEG IDCT one level per
// sample either side of the reference, so two conformant decoders may differ
// by two. See the package comment for the derivation, and for why it is not
// pdfium's 3.
const Gate = 2

// A Difference is one picture compared with the judge's, per channel.
//
// Every term is over the CHANNELS the package comment names: three for an
// ordinary picture, one derived coverage channel for a stencil.
type Difference struct {
	// Share is the fraction of pixels where some compared channel differs by
	// more than Gate, or -1 when the picture could not be judged. It is a
	// REPORT and not a budget: the count budget is zero, so Share being nought
	// and Peak being within Gate are the same statement.
	Share float64
	// Peak is the largest absolute channel difference anywhere in the
	// picture, in levels of 255. This is the criterion.
	Peak int
	// MSE is the mean squared error over every compared channel, in levels
	// squared — pdfium's units at testing/image_diff.cpp:181 and FFmpeg's
	// omse at dct.c:256.
	MSE float64
	// Mean is the signed mean error over every compared channel, ours minus
	// theirs, in levels — FFmpeg's ome. It is the term that catches BIAS,
	// which is what the bisection this replaced could not see.
	Mean float64
	// Inverted says the direct comparison failed the gate and the picture
	// against the judge's COMPLEMENT passes it.
	//
	// pdfimages writes a STENCIL with the opposite polarity to the samples it
	// holds: an image with /ImageMask true, or a stream a picture names as its
	// /Mask. That is a convention, not a disagreement — and it is worth
	// telling apart from a real one, because "identical up to inversion" and
	// "wrong" look the same in a magnitude and only one of them needs looking
	// into.
	//
	// Which side is right was measured rather than assumed. On a 644-byte
	// document whose 8x8 mask has its left half masked out and its right half
	// painted, poppler's own pdftoppm paints the right half, which is what
	// PDF 32000-1 8.9.6.2 says, pdfimages writes the left one, and ours
	// agrees with the rendering.
	//
	// It is about being a stencil and nothing else. Not about JBIG2: 27 of
	// the 28 in us-opm are CCITT, one is Flate, and the minimal document has
	// no filter. Not about soft masks, which are the one kind pdfimages
	// leaves alone.
	//
	// Not every complement this reports is that convention. When a page draws
	// many uniform pictures of one size, match pairs them by size and order
	// and pairs a black one with a white one; see
	// https://github.com/go-pdfkit/conformance/issues/13.
	Inverted bool
}

// A Result is one picture judged.
type Result struct {
	Path string
	Page int
	// Name is the resource name the page draws it by.
	Name string
	// Filter is the image format it was stored in, empty for plain samples.
	Filter string
	// Stencil says render returned this as a MASK rather than as a picture —
	// a /ImageMask true stencil, or a stream some picture named as its /SMask
	// or /Mask. It decides what channel the two sides are compared in.
	Stencil bool
	// Decoded says a /Decode array shaped our samples. pdfimages writes the
	// samples as stored, so this picture and theirs are not the same question
	// and are counted apart rather than as a disagreement.
	Decoded bool
	// Space is the colour space pdfimages reports for the judge's picture:
	// gray, rgb, cmyk, lab, icc, index, sep, devn, or "-" for a mask. It is
	// empty when no row could be read for it.
	Space string
	// Converted says poppler had to convert that space to reach RGB, so a
	// per-channel difference on this picture is colour arithmetic and not
	// codec error. Those are tallied apart; see the package comment.
	Converted bool
	// Size is what the picture came out as.
	W, H int
	// Difference is how far apart the two came out, or Share -1 when there
	// was no comparison, which Note explains.
	Difference
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
	// Ours means ours would not open the document or draw the page and the
	// judge would. That is a defect and the only one of these that is: a
	// document the field can read and we cannot.
	Ours Missing = "ours"
	// Neither means no implementation would open it — ours refused, and so
	// did the judge, asked separately about the same file.
	//
	// This has to be told apart from Ours or the count misleads in the
	// direction of comfort in one direction and panic in the other. Seven of
	// the twelve documents of the ia-texts population are refused by ours;
	// all seven carry Adobe's proprietary EBX_HANDLER and poppler refuses
	// every one of them too. Folded together with a real refusal, that reads
	// as a decoder failing on 58% of a population. Counted apart, it says
	// what is true: that population is five documents, not twelve.
	Neither Missing = "neither"
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

// unjudged is the Difference of a picture nothing could be said about.
func unjudged() Difference { return Difference{Share: -1} }

// Judge takes the pictures out of one document twice and compares them.
func Judge(path string, opt Options) []Result {
	pages := opt.Pages
	if pages <= 0 {
		pages = 1
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return []Result{{Path: path, Difference: unjudged(),
			Missing: Ours, Note: "unreadable: " + err.Error()}}
	}
	d, err := reader.Open(b)
	if err != nil {
		return []Result{{Path: path, Difference: unjudged(),
			Missing: blame(path), Note: "refused: " + err.Error()}}
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
//
// Nothing deduplicates the pictures here, and that is a decision about which
// version of render this is built against. Under v0.19.0 render.Images
// answered per DRAW — a page that stamped the same logo five hundred times
// yielded five hundred entries decoded from the same bytes, against
// pdfimages's one, and every repeat after the first landed in the unmatched
// column. This package compensated by keeping one entry per resource name.
//
// v0.20.0 removed the cause: a page's resources are a graph and were being
// walked as a tree, so a picture reached through two forms was decoded twice.
// Each picture now comes back once however many forms reach it, and the
// compensation is not merely redundant but WRONG — a name is unique within one
// resource dictionary and not across them. Measured under v0.20.0 over the
// whole forms corpus, 40 of its 2268 documents draw two different pictures
// that share a name on their first page, and collapsing on the name would drop
// 64 of them: 28 in qpdf's form-XObject fixtures, where two forms each name
// their own Im1, and 36 in fr-impots, where one issuer's real forms do the
// same with Im0.
func judgePage(d *reader.Document, path string, p int) []Result {
	ours, err := render.Images(d, p)
	if err != nil {
		return []Result{{Path: path, Page: p, Difference: unjudged(),
			Missing: Ours, Note: "no page: " + err.Error()}}
	}
	if len(ours) == 0 {
		return nil
	}
	theirs, err := poppler(path, p)
	if err != nil {
		return []Result{{Path: path, Page: p, Difference: unjudged(),
			Missing: Theirs, Note: "they took nothing out: " + err.Error()}}
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
			W: im.Pic.W, H: im.Pic.H, Difference: unjudged()}
		j := match(theirs, claimed, im.Pic)
		if j < 0 {
			r.Note = "they took out nothing this size"
			out = append(out, r)
			continue
		}
		claimed[j] = true
		r.Space = theirs[j].space
		r.Converted = converted(theirs[j].space)
		r.Difference = difference(im.Pic, theirs[j].pic, im.Stencil)
		out = append(out, r)
	}
	return out
}

// match finds an unclaimed picture of the same size.
func match(theirs []shot, claimed []bool, ours *raster.Image) int {
	for j, t := range theirs {
		if !claimed[j] && t.pic.W == ours.W && t.pic.H == ours.H {
			return j
		}
	}
	return -1
}

// difference compares two pictures channel by channel.
//
// It walks the pixels once and carries five things out of that walk: the
// largest single-channel difference, the same against the judge's complement,
// the count of pixels where some channel differs by more than Gate, and the
// squared and signed sums the two aggregate terms are means of.
//
// The complement is measured in the same pass rather than in a second one
// because polarity has to stay its own signal — see Difference.Inverted — and
// a stencil corpus would otherwise walk every differing picture twice.
func difference(ours, theirs *raster.Image, mask bool) Difference {
	n := ours.W * ours.H
	if ours.W != theirs.W || ours.H != theirs.H || n == 0 {
		return unjudged()
	}
	m := 3
	if mask {
		m = 1
	}
	var count, peak, flipped int
	var sum, squares float64
	for i := 0; i < n; i++ {
		a := channels(ours, i, mask)
		b := channels(theirs, i, mask)
		worst, worstFlipped := 0, 0
		for c := 0; c < m; c++ {
			e := a[c] - b[c]
			sum += float64(e)
			squares += float64(e) * float64(e)
			if e < 0 {
				e = -e
			}
			if e > worst {
				worst = e
			}
			f := a[c] - (255 - b[c])
			if f < 0 {
				f = -f
			}
			if f > worstFlipped {
				worstFlipped = f
			}
		}
		if worst > peak {
			peak = worst
		}
		if worstFlipped > flipped {
			flipped = worstFlipped
		}
		if worst > Gate {
			count++
		}
	}
	samples := float64(n * m)
	return Difference{
		Share:    float64(count) / float64(n),
		Peak:     peak,
		MSE:      squares / samples,
		Mean:     sum / samples,
		Inverted: peak > Gate && flipped <= Gate,
	}
}

// channels reads the channels of one pixel that are compared.
//
// A mask is compared in one derived channel, INK COVERAGE, and the rest of the
// array is left at zero. The formula is the same on both sides and is correct
// for each of the three layouts a mask arrives in — shape in alpha, levels in
// RGB, and poppler's opaque grey; see the package comment. Every other picture
// is compared in R, G and B, and alpha is left out of those for the reason
// given there.
func channels(im *raster.Image, i int, mask bool) [3]int {
	p := im.Pix[i*4 : i*4+4]
	if !mask {
		return [3]int{int(p[0]), int(p[1]), int(p[2])}
	}
	luma := int((uint32(p[0])*299 + uint32(p[1])*587 + uint32(p[2])*114) / 1000)
	return [3]int{int(p[3]) * (255 - luma) / 255}
}

// direct is the set of colour spaces pdfimages reports that reach RGB without
// a conversion: greyscale, RGB itself, and the "-" it prints for a mask, which
// carries no colour at all.
var direct = map[string]bool{"gray": true, "rgb": true, "-": true}

// converted says poppler had to convert this colour space to write its PNG, so
// a per-channel difference on that picture is colour arithmetic rather than a
// decoder disagreeing. An unread space counts as converted; see the package
// comment.
func converted(space string) bool { return !direct[space] }

// infoCommand is a variable so a test can ask without poppler installed.
//
// pdfinfo rather than pdfimages, because the question is only whether the
// document opens: pdfimages answers "no pictures came out" for a document it
// read perfectly well and one it could not read at all, and those are the two
// things that must not be confused here.
var infoCommand = func(path string) error { return exec.Command("pdfinfo", path).Run() }

// blame says whose refusal it was, by asking the judge the same question.
//
// A document ours refuses is not evidence of a defect until it is known that
// something else could read it. Encrypted-with-DRM, truncated and malformed
// documents are ordinary in a mass-digitisation corpus, and a refusal every
// implementation makes is a fact about the corpus.
func blame(path string) Missing {
	if infoCommand(path) != nil {
		return Neither
	}
	return Ours
}

// popplerCommand is a variable so a test can stand in for the other
// implementation without one being installed.
var popplerCommand = func(args ...string) error { return exec.Command("pdfimages", args...).Run() }

// listCommand is a variable for the same reason, and is separate because this
// one is asked for its OUTPUT rather than for whether it worked.
var listCommand = func(args ...string) ([]byte, error) {
	return exec.Command("pdfimages", args...).Output()
}

// A shot is one picture the judge took out, with what it says about it.
type shot struct {
	pic *raster.Image
	// num is the index pdfimages wrote in the file's name, which is also the
	// row it listed the picture on.
	num int
	// space is the colour space of that row, empty when none was read.
	space string
}

// poppler takes the pictures out of one page with pdfimages, and asks it what
// colour space each was in.
//
// pdfimages EXTRACTS rather than renders, which is the whole point: asking
// pdftoppm would put poppler's rasteriser between the codec and the answer,
// and its resampling shows up as every decoder disagreeing with it slightly.
// Measured that way, four JBIG2 decoders all looked wrong; measured this way,
// one was exact on every stream and another was wrong on 91% of them.
func poppler(path string, page int) ([]shot, error) {
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
	spaces := listing(path, page)
	out := make([]shot, 0, len(names))
	for _, name := range names {
		im, err := readPNG(name)
		if err != nil {
			continue
		}
		n := number(name)
		out = append(out, shot{pic: im, num: n, space: spaces[n]})
	}
	// Glob's order is the filesystem's, and a lexical sort is not pdfimages's
	// either: a page with more than a thousand pictures numbers one of them
	// 1000, which sorts before 999. The number is what orders them.
	sort.Slice(out, func(i, j int) bool { return out[i].num < out[j].num })
	if len(out) == 0 {
		return nil, fmt.Errorf("no pictures came out")
	}
	return out, nil
}

// number is the index pdfimages wrote into a file's name.
//
// It is -1 for a name that carries none, which leaves the picture with no
// colour space, so it is counted among the converted rather than credited as
// agreeing.
func number(name string) int {
	base := strings.TrimSuffix(filepath.Base(name), ".png")
	_, digits, _ := strings.Cut(base, "-")
	n, err := strconv.Atoi(digits)
	if err != nil {
		return -1
	}
	return n
}

// listing asks pdfimages what it is about to write, and reads the colour space
// off each row.
//
// The rows are "page num type width height color comp bpc enc ...", the num
// column is the index of the file pdfimages writes for that row, and the
// header and rule lines fail to parse as a number and are skipped. A listing
// that could not be taken at all leaves every picture unclassified, which the
// package comment explains is deliberately loud.
func listing(path string, page int) map[int]string {
	out, err := listCommand("-list", "-f", fmt.Sprint(page), "-l", fmt.Sprint(page), path)
	if err != nil {
		return nil
	}
	spaces := map[int]string{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 6 {
			continue
		}
		num, err := strconv.Atoi(f[1])
		if err != nil {
			continue
		}
		spaces[num] = f[5]
	}
	return spaces
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

// A Bucket is what one group of a filter's comparable pictures came to.
//
// There are two, because a picture poppler had to convert to reach RGB is not
// evidence about a codec and must not be averaged with one that needed no
// conversion; see the package comment.
type Bucket struct {
	// Pictures is how many were judged in this bucket.
	Pictures int
	// Exact is how many had no channel differ by more than Gate. It is the
	// criterion.
	Exact int
	// Identical is how many of those had no channel differ AT ALL, so a
	// reader can see how much of the agreement is bit equality and how much
	// the gate bought. One gate applies to every filter, which is a loosening
	// for the lossless ones — they could defensibly be held to nought — and
	// this is the count that says whether the loosening bought anything.
	Identical int
	// Inverted is how many agreed with the judge's complement instead.
	Inverted int
	// Diffs holds the magnitude of each picture that differed.
	Diffs []Difference
}

// Counts is what a population came to, per filter.
type Counts struct {
	// Pictures is how many were judged.
	Pictures int
	// Unmatched is how many the other implementation had no picture for.
	Unmatched int
	// Remapped is how many carried a /Decode array, which we apply and
	// pdfimages does not, so the two sides were not asked the same question.
	// They are counted here rather than as disagreements: on a one-bit mask a
	// /Decode of [1 0] inverts every pixel, which reads as total disagreement
	// and is none.
	Remapped int
	// Direct is the pictures whose colour space needed no conversion, which
	// are the ones the agreement figure is computed over.
	Direct Bucket
	// Converted is the rest, tallied with their own magnitudes so that the
	// claim "this is colour arithmetic and not a decoder" can be checked
	// rather than assumed.
	Converted Bucket
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
		default:
			bucket(c, r.Converted).add(r.Difference)
		}
	}
	return by
}

// bucket picks which of a filter's two buckets a picture belongs in.
func bucket(c *Counts, isConverted bool) *Bucket {
	if isConverted {
		return &c.Converted
	}
	return &c.Direct
}

// add files one judged picture under how it came out.
func (b *Bucket) add(d Difference) {
	b.Pictures++
	switch {
	case d.Share == 0:
		b.Exact++
		if d.Peak == 0 {
			b.Identical++
		}
	case d.Inverted:
		b.Inverted++
	default:
		b.Diffs = append(b.Diffs, d)
	}
}

// Report writes a tally in the order that reads best: worst first, because a
// filter that is right everywhere is not the one to read about.
//
// Each filter takes a line for what could not be compared and a line for each
// bucket that holds anything, because the two buckets are two different
// questions and a reader who adds them together has exactly the number the
// split exists to prevent.
func Report(by map[string]*Counts) string {
	var sb strings.Builder
	for _, key := range order(by) {
		c := by[key]
		fmt.Fprintf(&sb, "%-22s %5d pictures  %5d unmatched  %5d remapped\n",
			key, c.Pictures, c.Unmatched, c.Remapped)
		reportBucket(&sb, "direct", &c.Direct)
		reportBucket(&sb, "converted", &c.Converted)
	}
	return sb.String()
}

// reportBucket writes one bucket's line, and nothing at all for a bucket with
// no picture in it.
func reportBucket(sb *strings.Builder, name string, b *Bucket) {
	if b.Pictures == 0 {
		return
	}
	fmt.Fprintf(sb, "  %-9s %5d pictures  %5d exact (%d identical)  %5d inverted  %5d differing",
		name, b.Pictures, b.Exact, b.Identical, b.Inverted, len(b.Diffs))
	if len(b.Diffs) > 0 {
		t := terms(b.Diffs)
		fmt.Fprintf(sb, "  share %.4f/%.4f  peak %.0f/%.0f  mse %.4f/%.4f  mean %+.4f/%+.4f",
			t.Share.Median, t.Share.Worst, t.Peak.Median, t.Peak.Worst,
			t.MSE.Median, t.MSE.Worst, t.Mean.Median, t.Mean.Worst)
	}
	sb.WriteByte('\n')
}

// rightness is the share of a filter's DIRECT comparable pictures that agreed,
// and it decides what is read first.
//
// A picture that was never comparable is not evidence either way, so it is left
// out — the remapped ones, the colour-converted ones and the ones that came
// back as an exact complement alike. A filter with NOTHING comparable is
// therefore not evidence at all, and counting it as nought right would put it
// at the top of the report, above every filter actually known to be wrong.
func rightness(c *Counts) float64 {
	comparable := c.Direct.Pictures - c.Direct.Inverted
	if comparable <= 0 {
		return 1
	}
	return float64(c.Direct.Exact) / float64(comparable)
}

// A Summary is one population's tally in a shape that keeps: named fields, a
// settled order, and no map. A baseline is only worth writing down if a later
// run can be diffed against it, and a Go map ordered by chance cannot be.
type Summary struct {
	Population string `json:"population"`
	// Documents is how many were looked at, which is not how many were
	// judged: a document that draws no pictures contributes none.
	Documents int `json:"documents"`
	// Refused is how many documents ours would not open or draw, and the
	// judge would. This is the count that is a defect.
	Refused int `json:"refused"`
	// Unopenable is how many neither would open. A corpus is full of these
	// and they say nothing about a decoder, but a population's real size is
	// its documents minus these and a rate quoted without them is quoted over
	// the wrong denominator.
	Unopenable int `json:"unopenable"`
	// Declined is how many pages ours drew pictures for and the judge took
	// none out of, so there was nothing to compare them with.
	Declined int            `json:"declined"`
	Filters  []FilterCounts `json:"filters"`
}

// FilterCounts is one filter's line of a report, as data.
type FilterCounts struct {
	Filter   string `json:"filter"`
	Pictures int    `json:"pictures"`
	// Unmatched and Remapped are the pictures no comparison was made of.
	Unmatched int `json:"unmatched"`
	Remapped  int `json:"remapped"`
	// Direct and Converted are the two buckets, absent when empty. The
	// agreement figure is Direct's and never the two added together.
	Direct    *BucketCounts `json:"direct,omitempty"`
	Converted *BucketCounts `json:"converted,omitempty"`
}

// BucketCounts is one bucket of one filter, as data.
type BucketCounts struct {
	Pictures int `json:"pictures"`
	Exact    int `json:"exact"`
	// Identical is how many of the exact ones differed by nothing at all, so
	// bit equality can be told from agreement the gate bought.
	Identical int `json:"identical"`
	Inverted  int `json:"inverted"`
	Differing int `json:"differing"`
	// Terms is the spread of each magnitude over the differing pictures,
	// absent when none differed. A pointer because 0 is a real answer here —
	// a bucket whose worst peak is nought is not the same as one with nothing
	// to disagree about — and omitempty cannot tell those apart.
	Terms *Terms `json:"terms,omitempty"`
}

// Terms is the spread of each magnitude over a bucket's differing pictures.
type Terms struct {
	Share Spread `json:"share"`
	Peak  Spread `json:"peak"`
	MSE   Spread `json:"mse"`
	Mean  Spread `json:"mean"`
}

// A Spread is the middle and the far end of one term.
type Spread struct {
	Median float64 `json:"median"`
	// Worst is the value furthest from zero, WITH ITS SIGN, so that a signed
	// mean says which way the bias ran and not only how big it was.
	Worst float64 `json:"worst"`
}

// Summarize turns a population's results into the record of it, worst filter
// first, which is the same order Report reads in.
func Summarize(population string, documents int, rs []Result) Summary {
	s := Summary{Population: population, Documents: documents}
	for _, r := range rs {
		switch r.Missing {
		case Ours:
			s.Refused++
		case Neither:
			s.Unopenable++
		case Theirs:
			s.Declined++
		}
	}
	by := Tally(rs)
	for _, key := range order(by) {
		c := by[key]
		s.Filters = append(s.Filters, FilterCounts{Filter: key,
			Pictures: c.Pictures, Unmatched: c.Unmatched, Remapped: c.Remapped,
			Direct: bucketCounts(&c.Direct), Converted: bucketCounts(&c.Converted)})
	}
	return s
}

// bucketCounts is one bucket as data, and nothing at all for an empty one.
func bucketCounts(b *Bucket) *BucketCounts {
	if b.Pictures == 0 {
		return nil
	}
	out := &BucketCounts{Pictures: b.Pictures, Exact: b.Exact,
		Identical: b.Identical, Inverted: b.Inverted, Differing: len(b.Diffs)}
	if len(b.Diffs) > 0 {
		t := terms(b.Diffs)
		out.Terms = &t
	}
	return out
}

// terms is the spread of every magnitude over the pictures that differed.
func terms(ds []Difference) Terms {
	return Terms{
		Share: spread(ds, func(d Difference) float64 { return d.Share }),
		Peak:  spread(ds, func(d Difference) float64 { return float64(d.Peak) }),
		MSE:   spread(ds, func(d Difference) float64 { return d.MSE }),
		Mean:  spread(ds, func(d Difference) float64 { return d.Mean }),
	}
}

// spread is the middle and the far end of one term.
//
// The far end is the value furthest from zero rather than the largest, which
// matters for the signed mean and for nothing else: a bucket whose every
// picture is a level darker than the judge's has a worst mean of -1, and
// reporting its largest instead would say the bias was the smallest one seen.
func spread(ds []Difference, term func(Difference) float64) Spread {
	v := make([]float64, len(ds))
	for i, d := range ds {
		v[i] = term(d)
	}
	sort.Float64s(v)
	worst := v[len(v)-1]
	if -v[0] > worst {
		worst = v[0]
	}
	return Spread{Median: v[len(v)/2], Worst: worst}
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
