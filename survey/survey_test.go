package survey

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pdfkit/reader"
)

// page builds a one-page document whose content is what is given and whose
// resources hold the images named.
func page(t *testing.T, content string, images map[string]reader.Object) string {
	t.Helper()
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	res := reader.Dict{}
	if len(images) > 0 {
		xo := reader.Dict{}
		for name, o := range images {
			xo[reader.Name(name)] = o
		}
		res["XObject"] = xo
	}
	pageRef := w.Add(reader.Dict{
		"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte(content)}),
		"Resources": res,
	})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// image is an image XObject in the filter named.
func image(w *reader.Writer, filter reader.Object) reader.Object {
	d := reader.Dict{"Type": reader.Name("XObject"), "Subtype": reader.Name("Image"),
		"Width": reader.Integer(1), "Height": reader.Integer(1)}
	if filter != nil {
		d["Filter"] = filter
	}
	return w.Add(&reader.Stream{Dict: d, Raw: []byte{0}})
}

func TestAPageOfNothingButAPictureIsBlankWithoutIt(t *testing.T) {
	// This is the distinction the whole package exists for: a filter that
	// APPEARS is not the same amount of harm as a filter whose absence leaves
	// a page with nothing on it.
	w := reader.NewWriter("1.7")
	_ = w
	var path string
	{
		w := reader.NewWriter("1.7")
		pagesRef := w.Reserve()
		img := image(w, reader.Name("JPXDecode"))
		pageRef := w.Add(reader.Dict{
			"Type": reader.Name("Page"), "Parent": pagesRef,
			"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
			"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("q 50 0 0 50 0 0 cm /I Do Q")}),
			"Resources": reader.Dict{"XObject": reader.Dict{"I": img}},
		})
		w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
			"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
		out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
			"Type": reader.Name("Catalog"), "Pages": pagesRef})})
		if err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(t.TempDir(), "scan.pdf")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := Survey([]string{path}, 0)
	if c.Documents != 1 || c.Pages != 1 {
		t.Fatalf("%d documents, %d pages", c.Documents, c.Pages)
	}
	if c.UsedBy["JPXDecode"] != 1 || c.Images["JPXDecode"] != 1 {
		t.Errorf("used by %d, %d images", c.UsedBy["JPXDecode"], c.Images["JPXDecode"])
	}
	if c.BlankWithout["JPXDecode"] != 1 {
		t.Errorf("a page of nothing but a JPX image is not counted as blank without it")
	}
}

func TestAPageWithWordsOnItIsNotBlankWithoutItsPicture(t *testing.T) {
	w := reader.NewWriter("1.7")
	img := image(w, reader.Name("JPXDecode"))
	_ = img
	var path string
	{
		w := reader.NewWriter("1.7")
		pagesRef := w.Reserve()
		img := image(w, reader.Name("JPXDecode"))
		pageRef := w.Add(reader.Dict{
			"Type": reader.Name("Page"), "Parent": pagesRef,
			"MediaBox": reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
			"Contents": w.Add(&reader.Stream{Dict: reader.Dict{},
				Raw: []byte("0 g 0 0 10 10 re f q 50 0 0 50 0 0 cm /I Do Q")}),
			"Resources": reader.Dict{"XObject": reader.Dict{"I": img}},
		})
		w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
			"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
		out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
			"Type": reader.Name("Catalog"), "Pages": pagesRef})})
		if err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(t.TempDir(), "mixed.pdf")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := Survey([]string{path}, 0)
	if c.UsedBy["JPXDecode"] != 1 {
		t.Fatal("the picture was not counted")
	}
	if c.BlankWithout["JPXDecode"] != 0 {
		t.Error("a page with a filled rectangle on it was called blank without its picture")
	}
}

func TestARefusalIsCountedWithItsReason(t *testing.T) {
	// A population's refusals are part of what it is, and their reasons decide
	// whether they are ours to fix.
	dir := t.TempDir()
	notPDF := filepath.Join(dir, "not.pdf")
	if err := os.WriteFile(notPDF, []byte("this is not a PDF"), 0o644); err != nil {
		t.Fatal(err)
	}
	absent := filepath.Join(dir, "absent.pdf")

	c := Survey([]string{notPDF, absent}, 0)
	if c.Files != 2 || c.Documents != 0 {
		t.Fatalf("%d files, %d documents", c.Files, c.Documents)
	}
	if len(c.Refused) != 2 {
		t.Errorf("refusals: %v", c.Refused)
	}
	if len(c.Reasons()) != 2 {
		t.Errorf("reasons: %v", c.Reasons())
	}
}

func TestTheFilterIsTheLastOneInTheChain(t *testing.T) {
	// A picture is Flate-then-DCT: what matters is what it ends up encoded in.
	var path string
	{
		w := reader.NewWriter("1.7")
		pagesRef := w.Reserve()
		chained := image(w, reader.Array{reader.Name("FlateDecode"), reader.Name("DCTDecode")})
		plain := image(w, nil)
		notAnImage := w.Add(&reader.Stream{Dict: reader.Dict{
			"Type": reader.Name("XObject"), "Subtype": reader.Name("Form"),
			"BBox": reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(1), reader.Integer(1)}}})
		pageRef := w.Add(reader.Dict{
			"Type": reader.Name("Page"), "Parent": pagesRef,
			"MediaBox": reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
			"Contents": w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("")}),
			"Resources": reader.Dict{"XObject": reader.Dict{
				"A": chained, "B": plain, "C": notAnImage}},
		})
		w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
			"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
		out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
			"Type": reader.Name("Catalog"), "Pages": pagesRef})})
		if err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(t.TempDir(), "chain.pdf")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := Survey([]string{path}, 0)
	if c.Images["DCTDecode"] != 1 {
		t.Errorf("the chain's last filter was not counted: %v", c.Images)
	}
	if c.Images["none"] != 1 {
		t.Errorf("an unfiltered picture was not counted: %v", c.Images)
	}
	// The form XObject is not an image and must not be counted as one.
	if total := c.Images["DCTDecode"] + c.Images["none"]; total != 2 {
		t.Errorf("counted %d images, want 2: %v", total, c.Images)
	}
}

func TestOnlySoManyPagesAreLookedAt(t *testing.T) {
	path := page(t, "0 g 0 0 10 10 re f", nil)
	if c := Survey([]string{path}, 0); c.Pages != 1 {
		t.Errorf("all pages: %d", c.Pages)
	}
	// A cap below the document's length stops early, which is how a survey of
	// a hundred thousand files stays affordable.
	if c := Survey([]string{path}, 1); c.Pages != 1 {
		t.Errorf("capped: %d", c.Pages)
	}
}

func TestTheNamesComeBackSorted(t *testing.T) {
	// So that two runs print the same thing.
	c := newCounts()
	c.UsedBy["Z"], c.UsedBy["A"] = 1, 1
	c.Refused["z"], c.Refused["a"] = 1, 1
	if got := c.Filters(); len(got) != 2 || got[0] != "A" {
		t.Errorf("filters %v", got)
	}
	if got := c.Reasons(); len(got) != 2 || got[0] != "a" {
		t.Errorf("reasons %v", got)
	}
}

func TestAPageThatCannotBeReadIsSkippedRatherThanFatal(t *testing.T) {
	// A corpus is somebody else's bytes: one document that will not answer must
	// not end a survey of a hundred thousand.
	var path string
	{
		w := reader.NewWriter("1.7")
		pagesRef := w.Reserve()
		// Resources that are not a dictionary, and no contents at all.
		pageRef := w.Add(reader.Dict{
			"Type": reader.Name("Page"), "Parent": pagesRef,
			"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(9), reader.Integer(9)},
			"Resources": reader.Integer(7),
		})
		w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
			"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
		out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
			"Type": reader.Name("Catalog"), "Pages": pagesRef})})
		if err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(t.TempDir(), "odd.pdf")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := Survey([]string{path}, 0)
	if c.Documents != 1 {
		t.Fatalf("the document did not open: %v", c.Refused)
	}
	if len(c.Images) != 0 {
		t.Errorf("images found where there are none: %v", c.Images)
	}
}

func TestAnXObjectThatIsNotThereIsNotAnImage(t *testing.T) {
	var path string
	{
		w := reader.NewWriter("1.7")
		pagesRef := w.Reserve()
		pageRef := w.Add(reader.Dict{
			"Type": reader.Name("Page"), "Parent": pagesRef,
			"MediaBox": reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(9), reader.Integer(9)},
			"Contents": w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("")}),
			// An XObject entry that is not a stream, and one that is not a dict.
			"Resources": reader.Dict{"XObject": reader.Dict{"A": reader.Integer(3)}},
		})
		w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
			"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
		out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
			"Type": reader.Name("Catalog"), "Pages": pagesRef})})
		if err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(t.TempDir(), "empty.pdf")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if c := Survey([]string{path}, 0); len(c.Images) != 0 {
		t.Errorf("images %v", c.Images)
	}
}

func TestTheCapStopsPartWayThroughADocument(t *testing.T) {
	// How a survey of a hundred thousand scanned books stays affordable: the
	// first pages say which codec a document is in, and the last four hundred
	// say it again.
	var path string
	{
		w := reader.NewWriter("1.7")
		pagesRef := w.Reserve()
		kids := reader.Array{}
		for i := 0; i < 3; i++ {
			kids = append(kids, w.Add(reader.Dict{
				"Type": reader.Name("Page"), "Parent": pagesRef,
				"MediaBox": reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(9), reader.Integer(9)},
				"Contents": w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("")}),
			}))
		}
		w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
			"Kids": kids, "Count": reader.Integer(len(kids))})
		out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
			"Type": reader.Name("Catalog"), "Pages": pagesRef})})
		if err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(t.TempDir(), "three.pdf")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if c := Survey([]string{path}, 0); c.Pages != 3 {
		t.Errorf("uncapped: %d pages, want 3", c.Pages)
	}
	if c := Survey([]string{path}, 2); c.Pages != 2 {
		t.Errorf("capped at 2: %d pages", c.Pages)
	}
}

func TestAMaskIsCountedByItsOwnFilter(t *testing.T) {
	// The metric that was missing, and its absence cost a release. A scanned
	// page's ink layer is shaped by a JBIG2 stencil in /Mask; counting filters
	// by what they encode as CONTENT said JBIG2 was almost nowhere, while it
	// shapes the text of every page those scanners produce.
	for _, key := range []string{"Mask", "SMask"} {
		t.Run(key, func(t *testing.T) {
			var path string
			w := reader.NewWriter("1.7")
			pagesRef := w.Reserve()
			mask := w.Add(&reader.Stream{Dict: reader.Dict{
				"Type": reader.Name("XObject"), "Subtype": reader.Name("Image"),
				"Width": reader.Integer(4), "Height": reader.Integer(4),
				"ImageMask": reader.Bool(true), "Filter": reader.Name("JBIG2Decode"),
			}, Raw: []byte{0}})
			img := w.Add(&reader.Stream{Dict: reader.Dict{
				"Type": reader.Name("XObject"), "Subtype": reader.Name("Image"),
				"Width": reader.Integer(4), "Height": reader.Integer(4),
				"Filter": reader.Name("JPXDecode"), reader.Name(key): mask,
			}, Raw: []byte{0}})
			pageRef := w.Add(reader.Dict{"Type": reader.Name("Page"), "Parent": pagesRef,
				"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(9), reader.Integer(9)},
				"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("")}),
				"Resources": reader.Dict{"XObject": reader.Dict{"I": img}}})
			w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
				"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
			out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
				"Type": reader.Name("Catalog"), "Pages": pagesRef})})
			if err != nil {
				t.Fatal(err)
			}
			path = filepath.Join(t.TempDir(), "scan.pdf")
			if err := os.WriteFile(path, out, 0o644); err != nil {
				t.Fatal(err)
			}
			c := Survey([]string{path}, 0)
			// The picture is counted by what IT is in, and the mask by what
			// the MASK is in. Both matter, and they are not the same question.
			if c.Images["JPXDecode"] != 1 {
				t.Errorf("the picture was counted as %v", c.Images)
			}
			if c.MaskedBy["JBIG2Decode"] != 1 {
				t.Errorf("the mask was counted as %v", c.MaskedBy)
			}
			if len(c.Masks()) != 1 {
				t.Errorf("masks %v", c.Masks())
			}
		})
	}
}

func TestAnImageWithNoMaskIsCountedAsHavingNone(t *testing.T) {
	path := page(t, "", nil)
	if c := Survey([]string{path}, 0); len(c.MaskedBy) != 0 {
		t.Errorf("masks found where there are none: %v", c.MaskedBy)
	}
}

func TestAMaskThatIsNotAnImageIsNotOne(t *testing.T) {
	// /Mask may also be an array — a range of colours to treat as absent —
	// which is not a stream and has no filter of its own.
	var path string
	{
		w := reader.NewWriter("1.7")
		pagesRef := w.Reserve()
		img := w.Add(&reader.Stream{Dict: reader.Dict{
			"Type": reader.Name("XObject"), "Subtype": reader.Name("Image"),
			"Width": reader.Integer(4), "Height": reader.Integer(4),
			"Filter": reader.Name("JPXDecode"),
			"Mask":   reader.Array{reader.Integer(0), reader.Integer(10)},
		}, Raw: []byte{0}})
		pageRef := w.Add(reader.Dict{"Type": reader.Name("Page"), "Parent": pagesRef,
			"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(9), reader.Integer(9)},
			"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("")}),
			"Resources": reader.Dict{"XObject": reader.Dict{"I": img}}})
		w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
			"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
		out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
			"Type": reader.Name("Catalog"), "Pages": pagesRef})})
		if err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(t.TempDir(), "keyed.pdf")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := Survey([]string{path}, 0)
	if len(c.MaskedBy) != 0 {
		t.Errorf("a colour-key mask was counted as an image: %v", c.MaskedBy)
	}
	if c.Images["JPXDecode"] != 1 {
		t.Errorf("the picture itself was not counted: %v", c.Images)
	}
}

// iccProfileBytes writes a real ICC profile, so these tests exercise the
// reading rather than a stand-in for it. tags is signature to tag data.
func iccProfileBytes(space, pcs string, tags [][2]any) []byte {
	head := make([]byte, 132)
	head[8] = 2
	copy(head[12:16], "prtr")
	copy(head[16:20], space)
	copy(head[20:24], pcs)
	copy(head[36:40], "acsp")
	binary.BigEndian.PutUint32(head[128:], uint32(len(tags)))
	table := make([]byte, len(tags)*12)
	off := len(head) + len(table)
	var body []byte
	for i, t := range tags {
		sig, data := t[0].(string), t[1].([]byte)
		copy(table[i*12:], sig)
		binary.BigEndian.PutUint32(table[i*12+4:], uint32(off+len(body)))
		binary.BigEndian.PutUint32(table[i*12+8:], uint32(len(data)))
		body = append(body, data...)
	}
	out := append(append(head, table...), body...)
	binary.BigEndian.PutUint32(out[0:4], uint32(len(out)))
	return out
}

func iccXYZ(x, y, z float64) []byte {
	d := make([]byte, 20)
	copy(d, "XYZ ")
	for i, v := range []float64{x, y, z} {
		binary.BigEndian.PutUint32(d[8+i*4:], uint32(int32(math.Round(v*65536))))
	}
	return d
}

func iccGamma(g float64) []byte {
	d := make([]byte, 14)
	copy(d, "curv")
	binary.BigEndian.PutUint32(d[8:], 1)
	binary.BigEndian.PutUint16(d[12:], uint16(math.Round(g*256)))
	return d
}

func iccOpaque(typ string, n int) []byte {
	d := make([]byte, 8+n)
	copy(d, typ)
	return d
}

// iccLutTag writes an mft2 lookup table of four inputs on a grid of two, which
// is the shape of a press profile and the shape no matrix describes.
func iccLutTag() []byte {
	d := make([]byte, 52)
	copy(d, "mft2")
	d[8], d[9], d[10] = 4, 3, 2
	for i := range 3 {
		binary.BigEndian.PutUint32(d[12+i*16:], 0x00010000)
	}
	binary.BigEndian.PutUint16(d[48:], 2)
	binary.BigEndian.PutUint16(d[50:], 2)
	put := func(v uint16) {
		var u [2]byte
		binary.BigEndian.PutUint16(u[:], v)
		d = append(d, u[:]...)
	}
	ramp := func() { put(0); put(0xffff) }
	for range 4 {
		ramp()
	}
	for range 16 * 3 {
		put(0x4000)
	}
	for range 3 {
		ramp()
	}
	return d
}

func rgbProfile() []byte {
	return iccProfileBytes("RGB ", "XYZ ", [][2]any{
		{"rXYZ", iccXYZ(0.4, 0.2, 0)}, {"gXYZ", iccXYZ(0.3, 0.7, 0.1)},
		{"bXYZ", iccXYZ(0.2, 0.1, 0.7)},
		{"rTRC", iccGamma(2.2)}, {"gTRC", iccGamma(2.2)}, {"bTRC", iccGamma(2.2)},
	})
}

func greyProfile() []byte {
	return iccProfileBytes("GRAY", "XYZ ", [][2]any{
		{"wtpt", iccXYZ(0.9642, 1.0, 0.8249)}, {"kTRC", iccGamma(2.2)},
	})
}

func pressProfile() []byte {
	return iccProfileBytes("CMYK", "Lab ", [][2]any{{"A2B1", iccLutTag()}})
}

// spacePage builds a one-page document whose /Resources /ColorSpace names the
// space given, under the name "CS".
func spacePage(t *testing.T, build func(w *reader.Writer) reader.Object) Counts {
	t.Helper()
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	pageRef := w.Add(reader.Dict{
		"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("0 0 1 rg 0 0 5 5 re f")}),
		"Resources": reader.Dict{"ColorSpace": reader.Dict{"CS": build(w)}},
	})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return Survey([]string{path}, 0)
}

func iccArray(w *reader.Writer, profile []byte) reader.Object {
	return reader.Array{reader.Name("ICCBased"),
		w.Add(&reader.Stream{Dict: reader.Dict{"N": reader.Integer(3)}, Raw: profile})}
}

func TestEveryProfileShapeIsCountedAsWhatGfxMakesOfIt(t *testing.T) {
	for _, c := range []struct {
		name    string
		profile []byte
		want    string
	}{
		{"three curves and a matrix", rgbProfile(), "matrix and curves"},
		{"one curve", greyProfile(), "one curve"},
		{"a press", pressProfile(), "lookup table, 4 channels"},
		{"a version 4 lookup table", iccProfileBytes("CMYK", "Lab ", [][2]any{{"A2B0", iccOpaque("mAB ", 64)}}),
			`a "mAB " lookup table`},
		{"a parametric curve", iccProfileBytes("RGB ", "XYZ ", [][2]any{
			{"rXYZ", iccXYZ(0.4, 0.2, 0)}, {"gXYZ", iccXYZ(0.3, 0.7, 0.1)},
			{"bXYZ", iccXYZ(0.2, 0.1, 0.7)}, {"rTRC", iccOpaque("para", 12)},
			{"gTRC", iccOpaque("para", 12)}, {"bTRC", iccOpaque("para", 12)}}),
			"a parametric curve"},
		{"not a profile", []byte("these are not profile bytes at all"), "malformed"},
	} {
		got := spacePage(t, func(w *reader.Writer) reader.Object { return iccArray(w, c.profile) })
		if got.Profiles[c.want] != 1 {
			t.Errorf("%s: counted %v, want one %q", c.name, got.Profiles, c.want)
		}
		if len(got.Shapes()) != 1 {
			t.Errorf("%s: shapes = %v, want exactly one", c.name, got.Shapes())
		}
	}
}

// TestAProfileIsFoundThroughTheSpaceThatNamesIt is the case the census was
// extended for: the press profile that carried the corpus's largest mean
// squared error is not named by the page, it is named by a Separation's
// ALTERNATE. A census of the spaces a page names directly sees nothing.
func TestAProfileIsFoundThroughTheSpaceThatNamesIt(t *testing.T) {
	for name, wrap := range map[string]func(w *reader.Writer, inner reader.Object) reader.Object{
		"a Separation's alternate": func(w *reader.Writer, inner reader.Object) reader.Object {
			return reader.Array{reader.Name("Separation"), reader.Name("PANTONE 293 U"), inner,
				w.Add(reader.Dict{"FunctionType": reader.Integer(2), "N": reader.Integer(1)})}
		},
		"a DeviceN's alternate": func(w *reader.Writer, inner reader.Object) reader.Object {
			return reader.Array{reader.Name("DeviceN"), reader.Array{reader.Name("Black")}, inner,
				w.Add(reader.Dict{"FunctionType": reader.Integer(2), "N": reader.Integer(1)})}
		},
		"an Indexed base": func(w *reader.Writer, inner reader.Object) reader.Object {
			return reader.Array{reader.Name("Indexed"), inner, reader.Integer(1), reader.String("ab")}
		},
		"a Pattern base": func(w *reader.Writer, inner reader.Object) reader.Object {
			return reader.Array{reader.Name("Pattern"), inner}
		},
	} {
		got := spacePage(t, func(w *reader.Writer) reader.Object {
			return wrap(w, iccArray(w, pressProfile()))
		})
		if got.Profiles["lookup table, 4 channels"] != 1 {
			t.Errorf("%s: counted %v, want the press profile", name, got.Profiles)
		}
	}
}

// TestASpaceThatNamesItselfDoesNotRunForever is the depth bound. A corpus is
// somebody else's bytes and a colour space may point at itself.
func TestASpaceThatNamesItselfDoesNotRunForever(t *testing.T) {
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	csRef := w.Reserve()
	w.Put(csRef, reader.Array{reader.Name("Pattern"), csRef})
	pageRef := w.Add(reader.Dict{
		"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("0 0 5 5 re f")}),
		"Resources": reader.Dict{"ColorSpace": reader.Dict{"CS": csRef}},
	})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Survey([]string{path}, 0); len(got.Profiles) != 0 {
		t.Errorf("counted %v, want nothing", got.Profiles)
	}
}

func TestASpaceWithNoProfileBehindItIsNotCounted(t *testing.T) {
	for name, build := range map[string]func(w *reader.Writer) reader.Object{
		"a name where a stream should be": func(w *reader.Writer) reader.Object {
			return reader.Array{reader.Name("ICCBased"), reader.Name("nonsense")}
		},
		"a stream that decodes to nothing": func(w *reader.Writer) reader.Object {
			return reader.Array{reader.Name("ICCBased"), w.Add(&reader.Stream{
				Dict: reader.Dict{"Filter": reader.Name("FlateDecode")}, Raw: []byte("not deflate")})}
		},
		"a space of one element": func(w *reader.Writer) reader.Object {
			return reader.Array{reader.Name("ICCBased")}
		},
		"a device space, which names no profile": func(w *reader.Writer) reader.Object {
			return reader.Name("DeviceRGB")
		},
		"a space this census does not follow": func(w *reader.Writer) reader.Object {
			return reader.Array{reader.Name("CalRGB"), w.Add(reader.Dict{})}
		},
	} {
		if got := spacePage(t, build); len(got.Profiles) != 0 {
			t.Errorf("%s: counted %v, want nothing", name, got.Profiles)
		}
	}
}

// TestAPicturesOwnColourSpaceIsCountedToo covers the other place a profile is
// named: not the page's resources but the image dictionary.
func TestAPicturesOwnColourSpaceIsCountedToo(t *testing.T) {
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	img := w.Add(&reader.Stream{Dict: reader.Dict{
		"Type": reader.Name("XObject"), "Subtype": reader.Name("Image"),
		"Width": reader.Integer(1), "Height": reader.Integer(1),
		"Filter":     reader.Name("DCTDecode"),
		"ColorSpace": iccArray(w, pressProfile()),
	}, Raw: []byte{0}})
	pageRef := w.Add(reader.Dict{
		"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("q 50 0 0 50 0 0 cm /I Do Q")}),
		"Resources": reader.Dict{"XObject": reader.Dict{"I": img}},
	})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Survey([]string{path}, 0); got.Profiles["lookup table, 4 channels"] != 1 {
		t.Errorf("counted %v, want the picture's own profile", got.Profiles)
	}
}

// TestAProfileInsideAFormIsFound. A page that draws all its content through a
// form XObject names no colour space of its own, and the form carries its own
// /Resources. Counting only the top level saw a third of this corpus's press
// profiles.
func TestAProfileInsideAFormIsFound(t *testing.T) {
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	form := w.Add(&reader.Stream{Dict: reader.Dict{
		"Type": reader.Name("XObject"), "Subtype": reader.Name("Form"),
		"BBox":      reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(5), reader.Integer(5)},
		"Resources": reader.Dict{"ColorSpace": reader.Dict{"CS": iccArray(w, pressProfile())}},
	}, Raw: []byte("/CS cs 0 0 5 5 re f")})
	pageRef := w.Add(reader.Dict{
		"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("/F Do")}),
		"Resources": reader.Dict{"XObject": reader.Dict{"F": form}},
	})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Survey([]string{path}, 0); got.Profiles["lookup table, 4 channels"] != 1 {
		t.Errorf("counted %v, want the profile inside the form", got.Profiles)
	}
}

// TestAFormThatNamesItselfDoesNotRunForever is the other depth bound: a form
// may name itself in its own resources, and a corpus is somebody else's bytes.
func TestAFormThatNamesItselfDoesNotRunForever(t *testing.T) {
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	formRef := w.Reserve()
	w.Put(formRef, &reader.Stream{Dict: reader.Dict{
		"Type": reader.Name("XObject"), "Subtype": reader.Name("Form"),
		"BBox":      reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(5), reader.Integer(5)},
		"Resources": reader.Dict{"XObject": reader.Dict{"F": formRef}},
	}, Raw: []byte("/F Do")})
	pageRef := w.Add(reader.Dict{
		"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("/F Do")}),
		"Resources": reader.Dict{"XObject": reader.Dict{"F": formRef}},
	})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Survey([]string{path}, 0); len(got.Profiles) != 0 {
		t.Errorf("counted %v, want nothing", got.Profiles)
	}
}

// TestSomethingThatIsNotAnXObjectIsSkipped covers the resource dictionary that
// names a thing which is not a stream at all.
func TestSomethingThatIsNotAnXObjectIsSkipped(t *testing.T) {
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	pageRef := w.Add(reader.Dict{
		"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox":  reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(50), reader.Integer(50)},
		"Contents":  w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("0 0 5 5 re f")}),
		"Resources": reader.Dict{"XObject": reader.Dict{"X": reader.Name("not a stream")}},
	})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Survey([]string{path}, 0); len(got.Profiles) != 0 {
		t.Errorf("counted %v, want nothing", got.Profiles)
	}
}
