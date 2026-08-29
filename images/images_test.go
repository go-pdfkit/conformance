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

func TestAPictureReadTheOtherWayRoundIsCaught(t *testing.T) {
	// The failure this whole package exists for: the ink is there, and it is
	// in the wrong places. A page comparison averages that away; this does not.
	standIn(t, twoPixels(false))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != 1 {
		t.Fatalf("a picture read backwards came out as %+v", got)
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
	for _, tc := range []struct {
		name, want string
		path       func(t *testing.T) string
	}{
		{"a file that is not there", "unreadable",
			func(t *testing.T) string { return filepath.Join(t.TempDir(), "gone.pdf") }},
		{"a file that is not a PDF", "refused", func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "no.pdf")
			os.WriteFile(p, []byte("hello"), 0o644)
			return p
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Judge(tc.path(t), Options{})
			if len(got) != 1 || got[0].Share != -1 || !strings.Contains(got[0].Note, tc.want) {
				t.Errorf("got %+v", got)
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
		t.Errorf("got %+v", got)
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
	if got := max(0, 0); got != 0 {
		t.Errorf("max(0,0) = %d", got)
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
		t.Errorf("page nine of a one-page document came back as %+v", got)
	}
}

func TestTheRealCommandIsTheOneThatIsRun(t *testing.T) {
	// The default reaches for pdfimages. Whether it is installed decides the
	// error, not whether the statement runs — so this covers the wiring on a
	// machine with nothing installed as well as on one with poppler.
	_ = popplerCommand("-h")
}
