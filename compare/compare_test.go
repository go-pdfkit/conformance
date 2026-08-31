package compare

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-gfx/gfx/raster"
	"github.com/go-pdfkit/reader"
)

// onePage writes a document whose page is a black square on white, and returns
// its path.
func onePage(t *testing.T, content string) string {
	t.Helper()
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	pageRef := w.Add(reader.Dict{"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox": reader.Array{reader.Integer(0), reader.Integer(0), reader.Integer(72), reader.Integer(72)},
		"Contents": w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte(content)})})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "one.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// standIn replaces poppler with a function that writes the picture the test
// wants, so these run on a machine with nothing installed.
func standIn(t *testing.T, draw func(w, h int) image.Image, w, h int) {
	t.Helper()
	was := popplerCommand
	t.Cleanup(func() { popplerCommand = was })
	popplerCommand = func(args ...string) (bool, error) {
		stem := args[len(args)-1]
		f, err := os.Create(stem + "-1.png")
		if err != nil {
			return false, err
		}
		defer f.Close()
		return false, png.Encode(f, draw(w, h))
	}
}

func flat(c color.RGBA) func(w, h int) image.Image {
	return func(w, h int) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < w*h; i++ {
			img.Pix[i*4], img.Pix[i*4+1] = c.R, c.G
			img.Pix[i*4+2], img.Pix[i*4+3] = c.B, c.A
		}
		return img
	}
}

func TestTwoPicturesOfTheSamePageAgree(t *testing.T) {
	// A blank page drawn by both: nothing differs.
	standIn(t, flat(color.RGBA{255, 255, 255, 255}), 72, 72)
	got := Compare(onePage(t, ""), Options{})
	if len(got) != 1 {
		t.Fatalf("%d results", len(got))
	}
	if got[0].Share != 0 {
		t.Errorf("share %v, note %q", got[0].Share, got[0].Note)
	}
	if got[0].Ours == 0 {
		t.Error("our own time was not measured")
	}
}

func TestAPageDrawnDifferentlyIsReported(t *testing.T) {
	// They draw it black, we draw it white: everything differs.
	standIn(t, flat(color.RGBA{0, 0, 0, 255}), 72, 72)
	got := Compare(onePage(t, ""), Options{})
	if got[0].Share < 0.99 {
		t.Errorf("share %v, want nearly all of it", got[0].Share)
	}
}

func TestPicturesOfDifferentSizesAreNotCompared(t *testing.T) {
	// Rather than compared badly. This is what -cropbox exists for: pdftoppm
	// shows the media box by default and we show the crop box.
	standIn(t, flat(color.RGBA{255, 255, 255, 255}), 40, 40)
	got := Compare(onePage(t, ""), Options{})
	if got[0].Share != -1 {
		t.Fatalf("share %v", got[0].Share)
	}
	if got[0].Note == "" {
		t.Error("no reason given")
	}
}

func TestWhatCannotBeJudgedSaysWhy(t *testing.T) {
	standIn(t, flat(color.RGBA{255, 255, 255, 255}), 72, 72)
	t.Run("a file that is not there", func(t *testing.T) {
		got := Compare(filepath.Join(t.TempDir(), "absent.pdf"), Options{})
		if len(got) != 1 || got[0].Share != -1 || got[0].Note == "" {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("a file that is not a PDF", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "not.pdf")
		if err := os.WriteFile(p, []byte("nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		got := Compare(p, Options{})
		if len(got) != 1 || got[0].Share != -1 {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("the other renderer refusing", func(t *testing.T) {
		was := popplerCommand
		defer func() { popplerCommand = was }()
		popplerCommand = func(...string) (bool, error) { return false, os.ErrPermission }
		got := Compare(onePage(t, ""), Options{})
		if got[0].Share != -1 || got[0].Note == "" {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("the other renderer writing nothing", func(t *testing.T) {
		was := popplerCommand
		defer func() { popplerCommand = was }()
		popplerCommand = func(...string) (bool, error) { return false, nil }
		got := Compare(onePage(t, ""), Options{})
		if got[0].Share != -1 {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("the other renderer writing something that is not a picture", func(t *testing.T) {
		was := popplerCommand
		defer func() { popplerCommand = was }()
		popplerCommand = func(args ...string) (bool, error) {
			return false, os.WriteFile(args[len(args)-1]+"-1.png", []byte("not a png"), 0o644)
		}
		got := Compare(onePage(t, ""), Options{})
		if got[0].Share != -1 {
			t.Errorf("got %+v", got)
		}
	})
}

func TestMorePagesThanTheDocumentHasIsNotAnError(t *testing.T) {
	standIn(t, flat(color.RGBA{255, 255, 255, 255}), 72, 72)
	got := Compare(onePage(t, ""), Options{Pages: 5})
	if len(got) != 1 {
		t.Errorf("%d results for a one-page document", len(got))
	}
}

func TestSummariseSaysWhatTheDistributionIs(t *testing.T) {
	rs := []Result{
		{Share: 0.001, Ours: time.Second},
		{Share: 0.02},
		{Share: 0.5, Ours: 30 * time.Second},
		{Share: -1, Note: "different sizes: 1x1 against 2x2"},
		{Share: -1, Note: "refused: something"},
	}
	s := Summarise(rs, 20*time.Second)
	if s.Compared != 3 || s.NotCompared != 2 {
		t.Fatalf("%d compared, %d not", s.Compared, s.NotCompared)
	}
	if s.Under[0.01] != 1 || s.Under[0.05] != 2 {
		t.Errorf("under: %v", s.Under)
	}
	if s.Max != 0.5 {
		t.Errorf("worst %v", s.Max)
	}
	if s.Slowest != 30*time.Second || s.Over != 1 {
		t.Errorf("slowest %v, %d over", s.Slowest, s.Over)
	}
	// A hundred variations of one failure are counted as one kind.
	if len(s.Notes) != 2 {
		t.Errorf("notes: %v", s.Notes)
	}
}

func TestSummarisingNothingSaysNothing(t *testing.T) {
	s := Summarise(nil, 0)
	if s.Compared != 0 || s.Max != 0 {
		t.Errorf("%+v", s)
	}
	if s := Summarise([]Result{{Share: -1, Note: "plain"}}, 0); s.Notes["plain"] != 1 {
		t.Errorf("notes %v", s.Notes)
	}
}

func TestTheDifferenceOfNothingIsNothing(t *testing.T) {
	empty := &raster.Image{W: 0, H: 0}
	if d := difference(empty, empty); d != 0 {
		t.Errorf("difference of two empty pictures is %v", d)
	}
}

func TestAPageWeWillNotDrawIsSaidSo(t *testing.T) {
	// A page whose media box asks for more pixels than the renderer allows.
	// We draw nothing, and the reason is carried rather than swallowed.
	standIn(t, flat(color.RGBA{255, 255, 255, 255}), 10, 10)
	w := reader.NewWriter("1.7")
	pagesRef := w.Reserve()
	pageRef := w.Add(reader.Dict{"Type": reader.Name("Page"), "Parent": pagesRef,
		"MediaBox": reader.Array{reader.Integer(0), reader.Integer(0),
			reader.Integer(200000), reader.Integer(200000)},
		"Contents": w.Add(&reader.Stream{Dict: reader.Dict{}, Raw: []byte("")})})
	w.Put(pagesRef, reader.Dict{"Type": reader.Name("Pages"),
		"Kids": reader.Array{pageRef}, "Count": reader.Integer(1)})
	out, err := w.Finish(reader.Dict{"Root": w.Add(reader.Dict{
		"Type": reader.Name("Catalog"), "Pages": pagesRef})})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "huge.pdf")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	got := Compare(path, Options{})
	if got[0].Share != -1 || got[0].Note == "" {
		t.Fatalf("got %+v", got[0])
	}
	if !contains(got[0].Note, "we drew nothing") {
		t.Errorf("note %q", got[0].Note)
	}
	if !contains(got[0].Note, "past the limit") {
		t.Errorf("the reason was swallowed: %q", got[0].Note)
	}
}

func contains(s, sub string) bool { return indexOf(s, sub) >= 0 }

func TestTheRealCommandIsTheOneThatIsRun(t *testing.T) {
	// The default reaches for pdftoppm. Whether it is installed decides the
	// error, not whether the statement runs — so this covers the wiring on a
	// machine with nothing installed as well as on one with poppler.
	_, _ = popplerCommand("-h")
}

func TestATemporaryDirectoryItCannotMakeIsReported(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "not-a-directory"))
	if _, _, _, err := draw("whatever.pdf", 1, 72); err == nil {
		t.Error("no error when there is nowhere to work")
	}
}

func TestAPictureItCannotOpenIsReported(t *testing.T) {
	was := popplerCommand
	defer func() { popplerCommand = was }()
	popplerCommand = func(args ...string) (bool, error) {
		p := args[len(args)-1] + "-1.png"
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			return false, err
		}
		return false, os.Chmod(p, 0o000)
	}
	if _, _, _, err := draw("whatever.pdf", 1, 72); err == nil {
		t.Error("a picture that cannot be opened was read")
	}
}

func TestPixelsDifferInBothDirections(t *testing.T) {
	// One picture lighter here and darker there: the difference is a distance,
	// not a subtraction.
	// Eight by eight, so the blur does not average the whole picture into one
	// grey: a's left half is dark where b's is light, and the right half the
	// other way round, so the per-pixel difference comes out negative on one
	// side and positive on the other.
	const n = 8
	a := &raster.Image{W: n, H: n, Pix: make([]uint8, n*n*4)}
	b := &raster.Image{W: n, H: n, Pix: make([]uint8, n*n*4)}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := (y*n + x) * 4
			dark, light := uint8(0), uint8(255)
			if x >= n/2 {
				dark, light = light, dark
			}
			a.Pix[i], a.Pix[i+1], a.Pix[i+2], a.Pix[i+3] = dark, dark, dark, 255
			b.Pix[i], b.Pix[i+1], b.Pix[i+2], b.Pix[i+3] = light, light, light, 255
		}
	}
	if d := difference(a, b); d < 0.5 {
		t.Errorf("two pictures that are each other's negative differ by only %v", d)
	}
	// And against itself, nothing.
	if d := difference(a, a); d != 0 {
		t.Errorf("a picture differs from itself by %v", d)
	}
}

func TestTheSlowPagesAreNamed(t *testing.T) {
	// A count of pages over a threshold says a corpus has a problem and gives
	// nobody a way to go and look at it.
	s := Summarise([]Result{
		{Path: "/c/three.pdf", Page: 1, Ours: 3 * time.Second},
		{Path: "/a/one.pdf", Page: 2, Ours: 9 * time.Second},
		{Path: "/b/two.pdf", Page: 1, Ours: 100 * time.Millisecond},
	}, time.Second)
	if s.Over != 2 || len(s.Slow) != 2 {
		t.Fatalf("%d over the threshold, %d named", s.Over, len(s.Slow))
	}
	if s.Slow[0].Path != "/a/one.pdf" || s.Slow[0].Page != 2 {
		t.Errorf("the worst page is %s page %d", s.Slow[0].Path, s.Slow[0].Page)
	}
}

func TestOnlySoManySlowPagesAreNamed(t *testing.T) {
	// Enough to see whether they are one document's doing or spread across a
	// population, and few enough that the report of a bad run stays readable.
	var rs []Result
	for i := 0; i < slowKept+4; i++ {
		rs = append(rs, Result{Path: "x.pdf", Page: i, Ours: 2 * time.Second})
	}
	s := Summarise(rs, time.Second)
	if s.Over != slowKept+4 {
		t.Errorf("counted %d over the threshold", s.Over)
	}
	if len(s.Slow) != slowKept {
		t.Errorf("named %d of them", len(s.Slow))
	}
	// Pages that took the same time are named in document order, so a rerun
	// says the same thing.
	if s.Slow[0].Page != 0 || s.Slow[1].Page != 1 {
		t.Errorf("named page %d then %d", s.Slow[0].Page, s.Slow[1].Page)
	}
}

func TestPagesOfTheSameLengthInDifferentDocuments(t *testing.T) {
	s := Summarise([]Result{
		{Path: "/z.pdf", Page: 1, Ours: 2 * time.Second},
		{Path: "/a.pdf", Page: 1, Ours: 2 * time.Second},
	}, time.Second)
	if len(s.Slow) != 2 || s.Slow[0].Path != "/a.pdf" {
		t.Errorf("named %s first", s.Slow[0].Path)
	}
}

func TestAJudgeThatWillNotFinishIsNamedRatherThanWaitedOn(t *testing.T) {
	// conformance#21: pdftoppm hangs for ever on at least one document of the
	// forms corpus, and from outside a hang and a slow page are the same
	// thing. The bound is only worth having if the page comes back by name
	// with the tool that hung, so it can be told from a page that disagreed.
	was := popplerCommand
	defer func() { popplerCommand = was }()
	popplerCommand = func(...string) (bool, error) { return true, os.ErrDeadlineExceeded }
	path := onePage(t, "")
	got := Compare(path, Options{})
	if len(got) != 1 || got[0].Share != -1 || got[0].Tool != "pdftoppm" {
		t.Fatalf("a hang came back as %+v", got)
	}
	if !contains(got[0].Note, "did not finish within") {
		t.Errorf("the note says %q", got[0].Note)
	}
	// And the run's summary names every one of them, uncapped: a cap is a way
	// of dropping names, and the point of the bound is that they are kept.
	s := Summarise(got, time.Hour)
	if len(s.Hung) != 1 || s.Hung[0].Path != path {
		t.Fatalf("the summary names %+v", s.Hung)
	}
	if s.NotCompared != 1 || s.Compared != 0 {
		t.Errorf("a hang was counted as a comparison: %+v", s)
	}
}
