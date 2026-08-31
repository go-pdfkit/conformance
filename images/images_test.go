package images

import (
	"fmt"
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

// standIn replaces poppler with a function writing the pictures the test
// wants, and a listing that says every one of them was greyscale.
//
// Both halves have to be stood in: a picture whose colour space could not be
// read is counted among the converted, so a test that stood in only the
// pictures would be measuring the converted bucket without meaning to.
func standIn(t *testing.T, pics ...image.Image) {
	t.Helper()
	standInSpaces(t, "gray", pics...)
}

// standInSpaces is standIn with the colour space each picture is listed under.
func standInSpaces(t *testing.T, space string, pics ...image.Image) {
	t.Helper()
	wasPictures, wasList := popplerCommand, listCommand
	t.Cleanup(func() { popplerCommand, listCommand = wasPictures, wasList })
	popplerCommand = func(args ...string) error {
		stem := args[len(args)-1]
		for i, pic := range pics {
			f, err := os.Create(fmt.Sprintf("%s-%03d.png", stem, i))
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
	listCommand = func(...string) ([]byte, error) {
		var sb strings.Builder
		sb.WriteString("page   num  type   width height color comp bpc  enc interp  object ID x-ppi y-ppi size ratio\n")
		sb.WriteString("------\n")
		for i := range pics {
			fmt.Fprintf(&sb, "   1  %4d image      2     1  %s    1   8  image  no   7  0  72  72 9B 50%%\n", i, space)
		}
		return []byte(sb.String()), nil
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

// twoPixelsShifted is that picture with both its levels moved n towards the
// middle, which is the kind of failure the bisection this replaced could not
// see at all.
func twoPixelsShifted(n int) image.Image {
	im := image.NewRGBA(image.Rect(0, 0, 2, 1))
	v := uint8(n)
	im.Set(0, 0, color.RGBA{v, v, v, 255})
	im.Set(1, 0, color.RGBA{255 - v, 255 - v, 255 - v, 255})
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
	if got[0].Share != 0 || got[0].Peak != 0 || got[0].MSE != 0 || got[0].Mean != 0 {
		t.Errorf("got %+v", got[0])
	}
	if got[0].Note != "" || got[0].Name != "I" || got[0].W != 2 || got[0].H != 1 {
		t.Errorf("got %+v", got[0])
	}
	if got[0].Space != "gray" || got[0].Converted {
		t.Errorf("a greyscale picture was not counted as direct: %+v", got[0])
	}
}

func TestALevelShiftUnderTheGateIsStillAgreement(t *testing.T) {
	// Two conformant JPEG decoders may sit one IDCT level either side of the
	// reference, so two levels is what they may legitimately differ by and
	// Gate is 2. A picture inside that agrees.
	standIn(t, twoPixelsShifted(Gate))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Share != 0 || got[0].Peak != Gate {
		t.Fatalf("a picture within the gate came out as %+v", got)
	}
	// The magnitude is still recorded, and its sign says which way it ran:
	// ours is darker than theirs at the first pixel and lighter at the
	// second, so the bias cancels and the squared error does not.
	if got[0].MSE != 4 || got[0].Mean != 0 {
		t.Errorf("the aggregate terms are %v and %v, want 4 and 0", got[0].MSE, got[0].Mean)
	}
}

func TestAUniformLevelShiftIsSeen(t *testing.T) {
	// THE defect this measure exists for. The bisection it replaced asked only
	// whether a pixel was ink, so a decoder a hundred levels off on every
	// pixel scored 0.000 — perfect agreement. Every term must catch it, and
	// the signed mean must say which way it ran.
	shifted := image.NewRGBA(image.Rect(0, 0, 2, 1))
	shifted.Set(0, 0, color.RGBA{0, 0, 0, 255})
	shifted.Set(1, 0, color.RGBA{155, 155, 155, 255})
	standIn(t, shifted)
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 {
		t.Fatalf("%d results", len(got))
	}
	d := got[0].Difference
	if d.Share != 0.5 || d.Peak != 100 || d.Inverted {
		t.Errorf("a hundred-level shift on half the pixels came out as %+v", d)
	}
	// Ours is lighter than theirs, so the bias is positive: 100 levels on
	// three of six samples.
	if d.Mean != 50 || d.MSE != 5000 {
		t.Errorf("the aggregate terms are %v and %v, want 50 and 5000", d.Mean, d.MSE)
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
	if got[0].Peak != 255 {
		t.Errorf("the peak of a black-for-white pixel is %d", got[0].Peak)
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
	// So an exact complement is a convention and not a disagreement. Under a
	// magnitude measure it would otherwise read as maximal error at every
	// pixel, which is why polarity stays its own signal.
	standIn(t, twoPixels(false))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || !got[0].Inverted || got[0].Share != 1 || got[0].Peak != 255 {
		t.Fatalf("got %+v", got)
	}
	by := Tally(got)
	c := by["(samples)"]
	if c == nil || c.Direct.Inverted != 1 || len(c.Direct.Diffs) != 0 {
		t.Fatalf("tallied as %+v", c)
	}
	if r := Report(by); !strings.Contains(r, "1 inverted") {
		t.Errorf("the report does not say so: %q", r)
	}
}

func TestANearComplementIsStillAComplement(t *testing.T) {
	// The gate applies to the complement too, so a polarity convention
	// carried through a codec that rounds is still recognised as a convention
	// rather than reported as the worst disagreement in the corpus.
	standIn(t, twoPixelsShifted(255-Gate))
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || !got[0].Inverted {
		t.Fatalf("a complement within the gate came out as %+v", got)
	}
}

func TestAPictureThatAgreesIsNeverCalledAComplement(t *testing.T) {
	// A uniform mid-grey is within the gate of its own complement, so the
	// direct comparison has to be tried first or a picture that agrees would
	// be filed as a polarity convention and left out of the rate.
	mid := &raster.Image{W: 1, H: 1, Pix: []uint8{127, 127, 127, 255}}
	got := difference(mid, &raster.Image{W: 1, H: 1, Pix: []uint8{128, 128, 128, 255}}, false)
	if got.Share != 0 || got.Inverted {
		t.Errorf("a picture that agrees came out as %+v", got)
	}
}

func TestAPictureMostlyWrongIsNotAComplement(t *testing.T) {
	// Only EVERY pixel being the other way round is a convention. Nearly every
	// pixel is a defect, and must stay in the differing count where it is read.
	got := Tally([]Result{{Name: "a", Filter: "F", Difference: Difference{Share: 0.99}}})
	if c := got["F"]; c == nil || c.Direct.Inverted != 0 || len(c.Direct.Diffs) != 1 {
		t.Errorf("got %+v", c)
	}
}

func TestAColourConvertedPictureIsCountedApart(t *testing.T) {
	// A CMYK or ICC picture reaches RGB through two different sets of colour
	// arithmetic, and per channel that difference is large and is not a
	// decoder disagreeing. It must not be absorbed by widening the gate, and
	// it must not be averaged into the rate.
	standInSpaces(t, "icc", threeQuartersWrong())
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": wide(w)}
	}), Options{})
	if len(got) != 1 || got[0].Space != "icc" || !got[0].Converted {
		t.Fatalf("got %+v", got)
	}
	by := Tally(got)
	c := by["(samples)"]
	if c == nil || c.Direct.Pictures != 0 || c.Converted.Pictures != 1 {
		t.Fatalf("tallied as %+v", c)
	}
	// The magnitudes are still recorded, so the claim that this is colour
	// arithmetic can be checked rather than assumed.
	if len(c.Converted.Diffs) != 1 || c.Converted.Diffs[0].Share != 0.75 {
		t.Errorf("the converted bucket kept %+v", c.Converted.Diffs)
	}
	if r := Report(by); !strings.Contains(r, "converted") {
		t.Errorf("the report does not name the bucket: %q", r)
	}
}

func TestEveryColourSpacePopplerNamesIsClassified(t *testing.T) {
	// The set is ImageOutputDev.cc:152-190. Only the three that reach RGB
	// without a conversion are direct, and a space that could not be read is
	// counted with the converted ones so that a failure of pdfimages -list is
	// loud rather than silently generous.
	for _, space := range []string{"gray", "rgb", "-"} {
		if converted(space) {
			t.Errorf("%q was counted as converted", space)
		}
	}
	for _, space := range []string{"cmyk", "lab", "icc", "index", "sep", "devn", ""} {
		if !converted(space) {
			t.Errorf("%q was counted as direct", space)
		}
	}
}

func TestAPictureWithNoListingRowIsNotCreditedAsAgreeing(t *testing.T) {
	// A picture the listing said nothing about cannot be classified, and is
	// counted among the converted rather than credited to the rate.
	standIn(t, twoPixels(true))
	was := listCommand
	defer func() { listCommand = was }()
	listCommand = func(...string) ([]byte, error) { return nil, os.ErrPermission }
	got := Judge(pageOfPictures(t, func(w *reader.Writer) reader.Dict {
		return reader.Dict{"I": grey(w)}
	}), Options{})
	if len(got) != 1 || got[0].Space != "" || !got[0].Converted {
		t.Fatalf("got %+v", got)
	}
}

func TestAFileTheJudgeNumberedNothing(t *testing.T) {
	// pdfimages numbers the file it writes with the row it listed the picture
	// on, so a name with no digits after the dash cannot be classified.
	if got := number("i-.png"); got != -1 {
		t.Errorf("a name with no number came to %d", got)
	}
	if got := number("i-007.png"); got != 7 {
		t.Errorf("i-007.png came to %d", got)
	}
}

func TestTheListingIsReadPastItsHeader(t *testing.T) {
	// The header and the rule under it do not parse as rows, and a row too
	// short to hold a colour space is not one either.
	was := listCommand
	defer func() { listCommand = was }()
	listCommand = func(...string) ([]byte, error) {
		return []byte("page   num  type   width height color comp bpc\n" +
			"-----------------\n" +
			"   1     0 image     2     1  index   1   8\n" +
			"   1     1\n"), nil
	}
	got := listing("whatever.pdf", 1)
	if len(got) != 1 || got[0] != "index" {
		t.Errorf("the listing read as %v", got)
	}
}

func TestPicturesAreOrderedByTheirNumberAndNotTheirName(t *testing.T) {
	// A page with more than a thousand pictures numbers one of them 1000,
	// which sorts before 999 lexically. Matching is by size and order, so a
	// lexical order pairs the wrong pictures.
	wasPictures, wasList := popplerCommand, listCommand
	defer func() { popplerCommand, listCommand = wasPictures, wasList }()
	popplerCommand = func(args ...string) error {
		stem := args[len(args)-1]
		for i, n := range []string{"0999", "1000"} {
			f, err := os.Create(stem + "-" + n + ".png")
			if err != nil {
				return err
			}
			err = png.Encode(f, twoPixels(i == 0))
			f.Close()
			if err != nil {
				return err
			}
		}
		return nil
	}
	listCommand = func(...string) ([]byte, error) { return nil, os.ErrPermission }
	got, err := poppler("whatever.pdf", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].num != 999 || got[1].num != 1000 {
		t.Fatalf("ordered as %+v", got)
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
	if got[0].Share != 0 || !got[1].Inverted {
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
	if got := difference(a, &raster.Image{W: 1, H: 1, Pix: make([]uint8, 4)}, false); got.Share != -1 {
		t.Errorf("two sizes compared to %+v", got)
	}
	empty := &raster.Image{W: 0, H: 0}
	if got := difference(empty, empty, false); got.Share != -1 {
		t.Errorf("two empty pictures compared to %+v", got)
	}
}

func TestAMaskIsComparedInInkCoverage(t *testing.T) {
	// render puts a mask in one of two places and poppler in a third, so one
	// symmetric formula reduces all three; the layouts were read out of the
	// buffers and are cited in the package comment.
	//
	// A /ImageMask true stencil: black, with the shape in the alpha channel.
	// Painted then seen through.
	stencil := &raster.Image{W: 2, H: 1, Pix: []uint8{0, 0, 0, 255, 0, 0, 0, 0}}
	// poppler writes the same mask as opaque grey, black where it paints.
	theirs := &raster.Image{W: 2, H: 1, Pix: []uint8{0, 0, 0, 255, 255, 255, 255, 255}}
	if got := difference(stencil, theirs, true); got.Share != 0 || got.Peak != 0 {
		t.Errorf("a stencil both sides agree on came out as %+v", got)
	}
	// A /SMask: opaque, with its levels in RGB. The same formula reduces it
	// to the same coverage, so it compares against poppler's grey directly —
	// and reading it as a stencil's alpha would have said 255 everywhere.
	smask := &raster.Image{W: 2, H: 1, Pix: []uint8{0, 0, 0, 255, 255, 255, 255, 255}}
	if got := difference(smask, theirs, true); got.Share != 0 || got.Peak != 0 {
		t.Errorf("a soft mask both sides agree on came out as %+v", got)
	}
	// Read as three colour channels instead, the stencil disagrees with
	// poppler on everything — which is what the reduction exists to prevent.
	if got := difference(stencil, theirs, false); got.Peak != 255 {
		t.Errorf("read as colour, the same pair came out as %+v", got)
	}
	// And the polarity convention still reads as one.
	flipped := &raster.Image{W: 2, H: 1, Pix: []uint8{255, 255, 255, 255, 0, 0, 0, 255}}
	if got := difference(stencil, flipped, true); !got.Inverted {
		t.Errorf("a stencil written the other way round came out as %+v", got)
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
		{Name: "a", Filter: "JBIG2Decode", Stencil: true, Decoded: true,
			Difference: Difference{Share: 1}},
		{Name: "b", Filter: "JBIG2Decode", Stencil: true},
	})
	c := by["JBIG2Decode mask"]
	if c == nil || c.Remapped != 1 || c.Direct.Exact != 1 || len(c.Direct.Diffs) != 0 {
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
		{Name: "a", Filter: "RemappedDecode", Decoded: true, Difference: Difference{Share: 1}},
		{Name: "b", Filter: "WrongDecode", Difference: Difference{Share: 0.5}},
		{Name: "c", Filter: "WrongDecode"},
	}))
	if !strings.HasPrefix(got, "WrongDecode") {
		t.Errorf("the report reads %q", got)
	}
}

func TestTheTallyGroupsByWhatReadThePicture(t *testing.T) {
	by := Tally([]Result{
		{Name: "a", Filter: "JBIG2Decode", Stencil: true},
		{Name: "b", Filter: "JBIG2Decode", Stencil: true, Difference: Difference{Share: 0.5}},
		{Name: "c", Filter: "JBIG2Decode", Stencil: true, Difference: Difference{Share: -1}},
		{Name: "d", Filter: ""},
		{Name: "", Difference: Difference{Share: -1}, Note: "a whole document"},
	})
	if len(by) != 2 {
		t.Fatalf("grouped into %d: %v", len(by), by)
	}
	jb := by["JBIG2Decode mask"]
	if jb == nil || jb.Pictures != 3 || jb.Unmatched != 1 {
		t.Errorf("got %+v", jb)
	}
	if jb.Direct.Exact != 1 || len(jb.Direct.Diffs) != 1 {
		t.Errorf("the direct bucket came out as %+v", jb.Direct)
	}
	if s := by["(samples)"]; s == nil || s.Direct.Exact != 1 {
		t.Errorf("plain samples came out as %+v", s)
	}
}

func TestTheWorstFilterIsReportedFirst(t *testing.T) {
	// A filter that is right everywhere is not the one to read about.
	got := Report(Tally([]Result{
		{Name: "a", Filter: "GoodDecode"},
		{Name: "b", Filter: "BadDecode", Difference: Difference{Share: 0.5, Peak: 40, MSE: 9, Mean: -3}},
		{Name: "c", Filter: "BadDecode", Difference: Difference{Share: 0.25, Peak: 8, MSE: 1, Mean: -1}},
	}))
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 4 {
		t.Fatalf("report is %q", got)
	}
	if !strings.HasPrefix(lines[0], "BadDecode") {
		t.Errorf("the report leads with %q", lines[0])
	}
	// The spread of every term is on the bucket's line, and the signed mean
	// keeps its sign so that a reader can see which way the bias ran.
	for _, want := range []string{"share 0.5000/0.5000", "peak 40/40",
		"mse 9.0000/9.0000", "mean -1.0000/-3.0000"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("no %q in %q", want, lines[1])
		}
	}
	if strings.Contains(lines[3], "share") {
		t.Errorf("a filter with nothing differing reported a spread: %q", lines[3])
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
		{Name: "a", Filter: "ZDecode"},
		{Name: "b", Filter: "ADecode"},
	}))
	if !strings.HasPrefix(got, "ADecode") {
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

func TestTheRealCommandsAreTheOnesThatAreRun(t *testing.T) {
	// The defaults reach for pdfimages twice: once for the pictures and once
	// for what it says about them. Whether it is installed decides the error,
	// not whether the statement runs — so this covers the wiring on a machine
	// with nothing installed as well as on one with poppler.
	_ = popplerCommand("-h")
	_, _ = listCommand("-h")
}

func TestSummarizeKeepsWhatTheReportSays(t *testing.T) {
	// The record and the report are the same measurement, so a filter that
	// reads worst in one must come first in the other.
	got := Summarize("ia-medical", 4, []Result{
		{Name: "A", Filter: "CCITTFaxDecode", Missing: Judged},
		{Name: "B", Filter: "JBIG2Decode", Difference: Difference{Share: 0.5, Peak: 9, MSE: 4, Mean: -2}},
		{Name: "C", Filter: "JBIG2Decode", Difference: Difference{Share: 0.25, Peak: 3, MSE: 1, Mean: 1}},
		{Name: "D", Filter: "JBIG2Decode", Difference: Difference{Share: -1}},
		{Name: "E", Filter: "DCTDecode", Difference: Difference{Share: 1, Inverted: true}},
		{Name: "F", Filter: "DCTDecode", Decoded: true},
		{Name: "G", Filter: "DCTDecode", Converted: true,
			Difference: Difference{Share: 0.3, Peak: 20, MSE: 30, Mean: 5}},
		{Difference: Difference{Share: -1}, Missing: Ours, Note: "refused: x"},
		{Difference: Difference{Share: -1}, Missing: Neither, Note: "refused: y"},
		{Difference: Difference{Share: -1}, Missing: Theirs, Note: "they took nothing out: x"},
		{Difference: Difference{Share: -1}, Missing: Theirs, Note: "they took nothing out: y"},
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
	if j.Pictures != 3 || j.Unmatched != 1 || j.Direct == nil || j.Direct.Differing != 2 {
		t.Fatalf("JBIG2 counted as %+v / %+v", j, j.Direct)
	}
	// The middle is the middle of the sorted values, and the far end is the
	// one furthest from zero — which for the signed mean is -2, not 1.
	if j.Direct.Terms.Peak.Worst != 9 || j.Direct.Terms.Mean.Worst != -2 {
		t.Errorf("the spread of two differing pictures is %+v", j.Direct.Terms)
	}
	// The colour-converted picture is in its own bucket with its own terms,
	// and is no part of the direct one.
	var dct FilterCounts
	for _, f := range got.Filters {
		if f.Filter == "DCTDecode" {
			dct = f
		}
	}
	if dct.Direct == nil || dct.Direct.Inverted != 1 || dct.Direct.Differing != 0 {
		t.Fatalf("the direct bucket came out as %+v", dct.Direct)
	}
	if dct.Converted == nil || dct.Converted.Pictures != 1 || dct.Converted.Differing != 1 {
		t.Fatalf("the converted bucket came out as %+v", dct.Converted)
	}
	if dct.Converted.Terms.MSE.Worst != 30 {
		t.Errorf("the converted terms are %+v", dct.Converted.Terms)
	}
	// A bucket where nothing differed carries no spread at all, so a reader
	// cannot mistake a nought for a measurement.
	ccitt := got.Filters[1]
	if ccitt.Direct == nil || ccitt.Direct.Terms != nil {
		t.Errorf("%s differed nowhere yet carries a spread", ccitt.Filter)
	}
}

func TestSummarizeSaysNothingAboutAnEmptyTally(t *testing.T) {
	// A population whose documents draw no pictures is not a population that
	// disagreed, and its record must not invent a filter to say so.
	if got := Summarize("empty", 0, nil); len(got.Filters) != 0 {
		t.Errorf("got %+v", got.Filters)
	}
}
