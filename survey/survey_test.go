package survey

import (
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
