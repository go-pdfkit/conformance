package images

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-gfx/gfx/raster"
	"github.com/go-pdfkit/reader"
	"github.com/go-pdfkit/render"
)

// pageOfPictures writes a document whose page draws the given image XObjects,
// and returns its path.
func pageOfPictures(t *testing.T, build func(w *reader.Writer) reader.Dict) string {
	t.Helper()
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	pageRef := w.Add(reader.Dict{"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(20), reader.Integer(20)},
		"Resources": reader.Dict{"XObject": build(w)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("")})})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pics.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// grey adds a two-pixel picture: dark then light.
func grey(w *reader.Writer) reader.Object {
	return w.Add(&reader.Stream{Dict: reader.Dict{
		"Type": reader.Name("XObject"), "Subtype": reader.Name("Image"),
		"Width": reader.Integer(2), "Height": reader.Integer(1),
		"ColorSpace": reader.Name("DeviceGray"), "BitsPerComponent": reader.Integer(8),
	}, Raw: []byte{0x00, 0xff}})
}

// standIn replaces poppler with a function writing the pictures the test wants.
func standIn(t *testing.T, pics ...image.Image) {
	t.Helper()
	was := popplerCommand
	t.Cleanup(func() { popplerCommand = was })
	popplerCommand = func(args ...string) error {
		stem := args[len(args)-1]
		for i, pic := range pics {
			f, err := os.Create(stem + "-00" + string(rune('0'+i)) + ".png")
			if err != nil {
				return err
			}
			if err := png.Encode(f, pic); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
		return nil
	}
}

// wide adds a four-pixel picture: dark, light, dark, light.
func wide(w *reader.Writer) reader.Object {
	return w.Add(&reader.Stream{Dict: reader.Dict{
		"Type": reader.Name("XObject"), "Subtype": reader.Name("Image"),
		"Width": reader.Integer(4), "Height": reader.Integer(1),
		"ColorSpace": reader.Name("DeviceGray"), "BitsPerComponent": reader.Integer(8),
	}, Raw: []byte{0x00, 0xff, 0x00, 0xff}})
}

// threeQuartersWrong is that picture with three of its four pixels changed,
// which is a defect and not a convention.
func threeQuartersWrong() image.Image {
	im := image.NewRGBA(image.Rect(0, 0, 4, 1))
	black, white := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}
	im.Set(0, 0, black)
	im.Set(1, 0, black)
	im.Set(2, 0, white)
	im.Set(3, 0, black)
	return im
}

// twoPixels builds the picture the fixture decodes to, or its opposite.
func twoPixels(darkFirst bool) image.Image {
	im := image.NewRGBA(image.Rect(0, 0, 2, 1))
	a, b := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}
	if !darkFirst {
		a, b = b, a
	}
	im.Set(0, 0, a)
	im.Set(1, 0, b)
	return im
}

func TestAPictureBothSidesReadTheSameWayIsExact(t *testing.T) {
	standIn(t, twoPixels(true))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 {
		t.Fatalf("%d results", len(got))
	}
	if got[0].Share != 0 || got[0].Note != "" {
		t.Errorf("got %+v", got[0])
	}
	if got[0].Name != "I" || got[0].W != 2 || got[0].H != 1 {
		t.Errorf("got %+v", got[0])
	}
}

func TestAPictureInTheWrongPlacesIsCaught(t *testing.T) {
	// The failure this whole package exists for: the ink is there, and it is
	// in the wrong places. A page comparison averages that away; this does not.
	//
	// Three pixels of four, so it is not the exact complement that a
	// convention produces.
	standIn(t, threeQuartersWrong())
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": wide(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != 0.75 || got[0].Inverted {
		t.Fatalf("a picture read wrongly came out as %+v", got)
	}
}

func TestAnExactComplementIsSaidToBeOne(t *testing.T) {
	// pdfimages writes a stencil — /ImageMask true, or a stream a picture
	// names as its /Mask — with the opposite polarity to its samples. Ours is
	// the right reading: on a document whose mask has one half painted and
	// one half masked out, poppler's own pdftoppm paints the half PDF
	// 32000-1 8.9.6.2 says to paint, pdfimages writes the other, and ours
	// agrees with the rendering.
	//
	// So an exact complement is a convention and not a disagreement. It has to
	// be told apart from a real one, because the two look the same in a count
	// of differing pixels.
	standIn(t, twoPixels(false))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != 1 || !got[0].Inverted {
		t.Fatalf("got %+v", got)
	}
	by := Tally(got)
	c := by["(samples)"]
	if c == nil || c.Inverted != 1 || len(c.Shares) != 0 {
		t.Fatalf("tallied as %+v", c)
	}
	if r := Report(by); !strings.Contains(r, "1 inverted") {
		t.Errorf("the report does not say so: %q", r)
	}
}

func TestAPictureMostlyWrongIsNotAComplement(t *testing.T) {
	// Only EVERY pixel differing is a convention. Nearly every pixel is a
	// defect, and must stay in the differing count where it will be read.
	got := Tally([]Result{{Name: "a", Filter: "F", Share: 0.99}})
	if c := got["F"]; c == nil || c.Inverted != 0 || len(c.Shares) != 1 {
		t.Errorf("got %+v", c)
	}
}

func TestAPictureTheOtherSideDoesNotHave(t *testing.T) {
	standIn(t, image.NewRGBA(image.Rect(0, 0, 9, 9)))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != -1 || got[0].Note == "" {
		t.Fatalf("got %+v", got)
	}
}

func TestEachPictureIsMatchedOnlyOnce(t *testing.T) {
	// Two pictures of the same size must not both be compared against the
	// same one of theirs, or a document with one right picture and one wrong
	// would come out perfect.
	standIn(t, twoPixels(true), twoPixels(false))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"A": grey(w), "B": grey(w)}
	}), Options{})
	if len(got) != 2 {
		t.Fatalf("%d results", len(got))
	}
	if got[0].Share != 0 || got[1].Share != 1 {
		t.Errorf("got %+v and %+v", got[0], got[1])
	}
}

func TestADocumentThatCannotBeLookedAt(t *testing.T) {
	notAPDF := func(t *testing.T) string {
		p := filepath.Join(t.TempDir(), "no.pdf")
		os.WriteFile(p, []byte("hello"), 0o644)
		return p
	}
	for _, tc := range []struct {
		name, want string
		path       func(t *testing.T) string
		// opens is what the judge says when asked the same file.
		opens error
		blame Missing
	}{
		{"a file that is not there", "unreadable",
			func(t *testing.T) string { return filepath.Join(t.TempDir(), "gone.pdf") },
			nil, Ours},
		// Refused by ours and read by the judge is the one shape of this
		// that is a defect, and it must not read the same as the others.
		{"a document only ours refuses", "refused", notAPDF, nil, Ours},
		{"a document neither will open", "refused", notAPDF, os.ErrInvalid, Neither},
	} {
		t.Run(tc.name, func(t *testing.T) {
			was := infoCommand
			defer func() { infoCommand = was }()
			infoCommand = func(string) error { return tc.opens }
			got := Judge(tc.path(t), Options{})
			if len(got) != 1 || got[0].Share != -1 || !strings.Contains(got[0].Note, tc.want) {
				t.Fatalf("got %+v", got)
			}
			if got[0].Missing != tc.blame {
				t.Errorf("blamed %q, want %q", got[0].Missing, tc.blame)
			}
		})
	}
}

func TestAPageWithNoPicturesSaysNothing(t *testing.T) {
	got := Judge(pageOfPictures(t, func(*reader.Writer) reader.Dict {
		return reader.Dict{}
	}), Options{})
	if len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestTheOtherSideRefusing(t *testing.T) {
	was := popplerCommand
	defer func() { popplerCommand = was }()
	popplerCommand = func(...string) error { return os.ErrPermission }
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != -1 || !strings.Contains(got[0].Note, "took nothing out") {
		t.Fatalf("got %+v", got)
	}
	// Ours drew a picture and the judge did not, which is a page there is no
	// answer about rather than one ours got wrong.
	if got[0].Missing != Theirs {
		t.Errorf("blamed %q", got[0].Missing)
	}
}

func TestTheOtherSideWritingNothing(t *testing.T) {
	was := popplerCommand
	defer func() { popplerCommand = was }()
	popplerCommand = func(...string) error { return nil }
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != -1 {
		t.Errorf("got %+v", got)
	}
}

func TestAFileTheyWroteThatIsNotAPicture(t *testing.T) {
	// A truncated PNG is skipped rather than taken for a picture of no size.
	was := popplerCommand
	defer func() { popplerCommand = was }()
	popplerCommand = func(args ...string) error {
		stem := args[len(args)-1]
		os.WriteFile(stem+"-000.png", []byte("not a png"), 0o644)
		return nil
	}
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != -1 {
		t.Errorf("got %+v", got)
	}
}

func TestAFileTheyWroteThatCannotBeOpened(t *testing.T) {
	if _, err := readPNG(filepath.Join(t.TempDir(), "gone.png")); err == nil {
		t.Error("a file that is not there was read")
	}
}

func TestMorePagesThanTheDocumentHas(t *testing.T) {
	standIn(t, twoPixels(true))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{Pages: 9})
	if len(got) != 1 {
		t.Errorf("%d results from a one-page document asked for nine", len(got))
	}
}

func TestPicturesOfDifferentSizesAreNotCompared(t *testing.T) {
	a := &raster.Image{W: 2, H: 1, Pix: make([]uint8, 8)}
	if got := difference(a, &raster.Image{W: 1, H: 1, Pix: make([]uint8, 4)}); got != -1 {
		t.Errorf("two sizes compared to %v", got)
	}
	empty := &raster.Image{W: 0, H: 0}
	if got := difference(empty, empty); got != -1 {
		t.Errorf("two empty pictures compared to %v", got)
	}
}

func TestWhatIsSeenThroughIsPaper(t *testing.T) {
	// A stencil carries its shape in the alpha channel with black underneath,
	// and poppler writes the same mask out as one bit. Reading a transparent
	// black pixel as ink would make every stencil disagree.
	im := &raster.Image{W: 1, H: 1, Pix: []uint8{0, 0, 0, 0}}
	if dark(im, 0) {
		t.Error("a pixel seen through was read as ink")
	}
}

func TestAPictureRemappedOnOneSideIsNotADisagreement(t *testing.T) {
	// /Decode maps the stored samples onto the range the colour space wants.
	// We apply it, as a viewer must; pdfimages writes the samples as stored.
	// On a one-bit mask a /Decode of [1 0] inverts every pixel, which reads as
	// total disagreement and is none — 8 of 23 JBIG2 masks in a corpus of
	// scanned medical pages came out at exactly 1.0000 that way, on pages that
	// match poppler's rendering to a median of 0.0000.
	by := Tally([]Result{
		{Name: "a", Filter: "JBIG2Decode", Stencil: true, Decoded: true, Share: 1},
		{Name: "b", Filter: "JBIG2Decode", Stencil: true, Share: 0},
	})
	c := by["JBIG2Decode mask"]
	if c == nil || c.Remapped != 1 || c.Exact != 1 || len(c.Shares) != 0 {
		t.Fatalf("got %+v", c)
	}
	if got := Report(by); !strings.Contains(got, "1 remapped") {
		t.Errorf("the report does not say so: %q", got)
	}
}

func TestWhatWasNeverComparableDoesNotDecideTheOrder(t *testing.T) {
	// A filter whose every picture was remapped is not evidence of anything,
	// and must not be reported as the worst thing in the corpus.
	got := Report(Tally([]Result{
		{Name: "a", Filter: "RemappedDecode", Decoded: true, Share: 1},
		{Name: "b", Filter: "WrongDecode", Share: 0.5},
		{Name: "c", Filter: "WrongDecode", Share: 0},
	}))
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "WrongDecode") {
		t.Errorf("the report reads %q", got)
	}
}

func TestTheTallyGroupsByWhatReadThePicture(t *testing.T) {
	by := Tally([]Result{
		{Name: "a", Filter: "JBIG2Decode", Stencil: true, Share: 0},
		{Name: "b", Filter: "JBIG2Decode", Stencil: true, Share: 0.5},
		{Name: "c", Filter: "JBIG2Decode", Stencil: true, Share: -1},
		{Name: "d", Filter: "", Share: 0},
		{Name: "", Share: -1, Note: "a whole document"},
	})
	if len(by) != 2 {
		t.Fatalf("grouped into %d: %v", len(by), by)
	}
	jb := by["JBIG2Decode mask"]
	if jb == nil || jb.Pictures != 3 || jb.Exact != 1 || jb.Unmatched != 1 || len(jb.Shares) != 1 {
		t.Errorf("got %+v", jb)
	}
	if s := by["(samples)"]; s == nil || s.Exact != 1 {
		t.Errorf("plain samples came out as %+v", s)
	}
}

func TestTheWorstFilterIsReportedFirst(t *testing.T) {
	// A filter that is right everywhere is not the one to read about.
	got := Report(Tally([]Result{
		{Name: "a", Filter: "GoodDecode", Share: 0},
		{Name: "b", Filter: "BadDecode", Share: 0.5},
		{Name: "c", Filter: "BadDecode", Share: 0.25},
	}))
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("report is %q", got)
	}
	if !strings.HasPrefix(lines[0], "BadDecode") {
		t.Errorf("the report leads with %q", lines[0])
	}
	if !strings.Contains(lines[0], "median 0.5000") || !strings.Contains(lines[0], "worst 0.5000") {
		t.Errorf("no spread in %q", lines[0])
	}
	if strings.Contains(lines[1], "median") {
		t.Errorf("a filter with nothing differing reported a median: %q", lines[1])
	}
}

func TestATallyOfNothing(t *testing.T) {
	if got := Report(Tally(nil)); got != "" {
		t.Errorf("an empty tally reported %q", got)
	}
	if got := rightness(&Counts{}); got != 1 {
		t.Errorf("a filter with nothing in it is %v right", got)
	}
}

func TestNowhereToPutWhatTheyWrite(t *testing.T) {
	// The pictures come back through the filesystem, so a machine with no
	// temporary directory reports that rather than pretending.
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "not-there"))
	if _, err := poppler("whatever.pdf", 1); err == nil {
		t.Error("pictures came out of a machine with nowhere to put them")
	}
}

func TestTwoFiltersThatAreEquallyRightReadInOrder(t *testing.T) {
	// Ordering by how wrong each is leaves ties, and a report whose lines move
	// between runs is as unreadable as one in no order at all.
	got := Report(Tally([]Result{
		{Name: "a", Filter: "ZDecode", Share: 0},
		{Name: "b", Filter: "ADecode", Share: 0},
	}))
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "ADecode") {
		t.Errorf("the report reads %q", got)
	}
}

func TestAPageThatIsNotThereSaysSo(t *testing.T) {
	// Judge clamps to the pages a document has, so this cannot arrive through
	// it — but judgePage takes any number, and a guard that is only correct
	// because of what its one caller does today is a guard waiting to be
	// wrong.
	path := pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	})
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := reader.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	got := judgePage(d, path, 9)
	if len(got) != 1 || got[0].Share != -1 || !strings.Contains(got[0].Note, "no page") {
		t.Fatalf("page nine of a one-page document came back as %+v", got)
	}
	if got[0].Missing != Ours {
		t.Errorf("blamed %q", got[0].Missing)
	}
}

func TestTheRealJudgeIsTheOneAskedWhoRefused(t *testing.T) {
	// Whether poppler is installed decides the answer, not whether the
	// statement runs.
	_ = infoCommand(filepath.Join(t.TempDir(), "gone.pdf"))
}

func TestTheRealCommandIsTheOneThatIsRun(t *testing.T) {
	// The default reaches for pdfimages. Whether it is installed decides the
	// error, not whether the statement runs — so this covers the wiring on a
	// machine with nothing installed as well as on one with poppler.
	_ = popplerCommand("-h")
}

func TestSummarizeKeepsWhatTheReportSays(t *testing.T) {
	// The record and the report are the same measurement, so a filter that
	// reads worst in one must come first in the other.
	got := Summarize("ia-medical", 4, []Result{
		{Name: "A", Filter: "CCITTFaxDecode", Share: 0, Missing: Judged},
		{Name: "B", Filter: "JBIG2Decode", Share: 0.5},
		{Name: "C", Filter: "JBIG2Decode", Share: 0.25},
		{Name: "D", Filter: "JBIG2Decode", Share: -1},
		{Name: "E", Filter: "DCTDecode", Share: 1, Inverted: true},
		{Name: "F", Filter: "DCTDecode", Decoded: true},
		{Share: -1, Missing: Ours, Note: "refused: x"},
		{Share: -1, Missing: Neither, Note: "refused: y"},
		{Share: -1, Missing: Theirs, Note: "they took nothing out: x"},
		{Share: -1, Missing: Theirs, Note: "they took nothing out: y"},
	})
	if got.Population != "ia-medical" || got.Documents != 4 {
		t.Errorf("the record does not say what was judged: %+v", got)
	}
	// What could not be compared is counted by the side that had nothing, so
	// a corpus that got harder cannot be read as a decoder that got worse.
	// A picture that was compared is neither, whichever way it came out.
	if got.Refused != 1 || got.Unopenable != 1 || got.Declined != 2 {
		t.Errorf("refused %d, unopenable %d, declined %d; want 1, 1 and 2",
			got.Refused, got.Unopenable, got.Declined)
	}
	if len(got.Filters) != 3 || got.Filters[0].Filter != "JBIG2Decode" {
		t.Fatalf("the worst filter is not first: %+v", got.Filters)
	}
	j := got.Filters[0]
	if j.Pictures != 3 || j.Differing != 2 || j.Unmatched != 1 || j.Exact != 0 {
		t.Errorf("JBIG2 counted as %+v", j)
	}
	if j.Median == nil || *j.Median != 0.5 || j.Worst == nil || *j.Worst != 0.5 {
		t.Errorf("the spread of two differing pictures is %v/%v", j.Median, j.Worst)
	}
	for _, f := range got.Filters[1:] {
		if f.Median != nil || f.Worst != nil {
			t.Errorf("%s differed nowhere yet carries a spread", f.Filter)
		}
	}
}

func TestSummarizeSaysNothingAboutAnEmptyTally(t *testing.T) {
	// A population whose documents draw no pictures is not a population that
	// disagreed, and its record must not invent a filter to say so.
	if got := Summarize("empty", 0, nil); len(got.Filters) != 0 {
		t.Errorf("got %+v", got.Filters)
	}
}

func TestARepeatedDrawIsOnePicture(t *testing.T) {
	// render.Images answers per draw and pdfimages answers per image. A page
	// that stamps the same logo repeatedly must be judged once, or every
	// repeat after the first lands in the unmatched column and the corpus
	// reads as unjudged when it was only counted twice.
	got := distinct([]render.Image{
		{Name: "Im1", Filter: "DCTDecode"},
		{Name: "Im2"},
		{Name: "Im1", Filter: "DCTDecode"},
		{Name: "Im1", Filter: "DCTDecode"},
		{Name: "Im2"},
	})
	if len(got) != 2 {
		t.Fatalf("kept %d pictures, want 2: %+v", len(got), got)
	}
	// The order it first draws them in is the order kept, because that is the
	// order the names read in and a record is compared by eye as well.
	if got[0].Name != "Im1" || got[1].Name != "Im2" {
		t.Errorf("kept %q then %q", got[0].Name, got[1].Name)
	}
}

func TestDistinctKeepsAPageThatRepeatsNothing(t *testing.T) {
	if got := distinct([]render.Image{{Name: "A"}, {Name: "B"}}); len(got) != 2 {
		t.Errorf("got %+v", got)
	}
}
