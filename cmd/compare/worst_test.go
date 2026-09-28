package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-pdfkit/conformance/compare"
)

// res is one row of the pool, at a ratio the arithmetic makes obvious.
func res(path string, ours, theirs time.Duration) compare.Result {
	return compare.Result{Path: path, Page: 1, Ours: ours, Theirs: theirs}
}

// TestAnInflatedWorstRowIsCorrectedByDrawingItAgain. This is the defect the
// mechanism exists for: 1 831 ms that is really 688.
func TestAnInflatedWorstRowIsCorrectedByDrawingItAgain(t *testing.T) {
	pool := []compare.Result{res("a.pdf", ms(1831), ms(672))}
	got := confirmWorst(pool, 2, 5, func(string, int) (time.Duration, time.Duration, bool) {
		return ms(688), ms(672), true
	})
	if len(got) != 1 {
		t.Fatalf("returned %d rows", len(got))
	}
	if got[0].Ours != ms(688) {
		t.Errorf("ours = %v, want the re-measured 688ms", got[0].Ours)
	}
	if got[0].Theirs != ms(672) {
		t.Errorf("theirs = %v, want 672ms", got[0].Theirs)
	}
}

// TestAPageWhoseSampleWasLuckyIsPromotedIntoTheList. The other half of the bias,
// and the reason Summarise hands back a pool: a page cannot be promoted out of a
// list that was cut to five before anyone measured it.
func TestAPageWhoseSampleWasLuckyIsPromotedIntoTheList(t *testing.T) {
	// Five pages that look bad and one, last in the pool, that looked fine.
	pool := []compare.Result{
		res("a.pdf", ms(500), ms(100)), // 5.0x by its sample
		res("b.pdf", ms(400), ms(100)),
		res("c.pdf", ms(300), ms(100)),
		res("d.pdf", ms(200), ms(100)),
		res("e.pdf", ms(150), ms(100)),
		res("lucky.pdf", ms(110), ms(100)), // 1.1x by its sample
	}
	got := confirmWorst(pool, 1, 5, func(path string, _ int) (time.Duration, time.Duration, bool) {
		if path == "lucky.pdf" {
			return ms(900), ms(100), true // it really costs 9x
		}
		return ms(120), ms(100), true // the other five really cost 1.2x
	})
	if len(got) != 5 {
		t.Fatalf("printed %d rows, want 5", len(got))
	}
	if got[0].Path != "lucky.pdf" {
		t.Errorf("worst row is %s, want lucky.pdf promoted to the top", got[0].Path)
	}
}

// TestTheWorstListIsTruncatedAfterConfirmingNotBefore.
func TestTheWorstListIsTruncatedAfterConfirmingNotBefore(t *testing.T) {
	pool := make([]compare.Result, 0, 9)
	for i := 0; i < 9; i++ {
		pool = append(pool, res(string(rune('a'+i))+".pdf", ms(100+i), ms(100)))
	}
	drawn := 0
	got := confirmWorst(pool, 1, 3, func(string, int) (time.Duration, time.Duration, bool) {
		drawn++
		return ms(100), ms(100), true
	})
	if drawn != 9 {
		t.Errorf("drew %d of the 9 pooled pages: truncation happened first", drawn)
	}
	if len(got) != 3 {
		t.Errorf("printed %d, want 3", len(got))
	}
}

// TestAPageThatCannotBeDrawnAgainKeepsItsReading. An unread answer is not a
// negative one, and dropping the row would hide the page that needs looking at.
func TestAPageThatCannotBeDrawnAgainKeepsItsReading(t *testing.T) {
	pool := []compare.Result{res("a.pdf", ms(900), ms(100))}
	got := confirmWorst(pool, 3, 5, func(string, int) (time.Duration, time.Duration, bool) {
		return 0, 0, false
	})
	if len(got) != 1 || got[0].Ours != ms(900) || got[0].Theirs != ms(100) {
		t.Errorf("got %+v, want the row unchanged", got)
	}
}

// TestZeroDurationsAreNotMeasurements. ok with a zero is a timing that was not
// taken, and letting it through would divide by zero or read as infinitely fast.
func TestZeroDurationsAreNotMeasurements(t *testing.T) {
	pool := []compare.Result{res("a.pdf", ms(900), ms(100))}
	for _, tc := range []struct {
		name         string
		ours, theirs time.Duration
	}{
		{"ours zero", 0, ms(100)},
		{"theirs zero", ms(100), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := confirmWorst(pool, 1, 5, func(string, int) (time.Duration, time.Duration, bool) {
				return tc.ours, tc.theirs, true
			})
			if got[0].Ours != ms(900) || got[0].Theirs != ms(100) {
				t.Errorf("got %v/%v, want the row unchanged", got[0].Ours, got[0].Theirs)
			}
		})
	}
}

// TestConfirmWorstTakesTheMinimumOfEachSide.
func TestConfirmWorstTakesTheMinimumOfEachSide(t *testing.T) {
	pool := []compare.Result{res("a.pdf", ms(900), ms(900))}
	ours := []time.Duration{ms(700), ms(200), ms(650)}
	theirs := []time.Duration{ms(300), ms(800), ms(100)}
	i := 0
	got := confirmWorst(pool, 3, 5, func(string, int) (time.Duration, time.Duration, bool) {
		o, th := ours[i], theirs[i]
		i++
		return o, th, true
	})
	if got[0].Ours != ms(200) {
		t.Errorf("ours = %v, want the minimum 200ms", got[0].Ours)
	}
	if got[0].Theirs != ms(100) {
		t.Errorf("theirs = %v, want the minimum 100ms", got[0].Theirs)
	}
}

// TestWithNoTriesTheListIsStillTruncated. -confirm 0 asks for the rows unchecked,
// and a report that then printed all fifteen would be a different change.
func TestWithNoTriesTheListIsStillTruncated(t *testing.T) {
	pool := make([]compare.Result, 0, 9)
	for i := 0; i < 9; i++ {
		pool = append(pool, res(string(rune('a'+i))+".pdf", ms(200-i), ms(100)))
	}
	got := confirmWorst(pool, 0, 5, func(string, int) (time.Duration, time.Duration, bool) {
		t.Error("it drew a page with tries = 0")
		return 0, 0, false
	})
	if len(got) != 5 || got[0].Path != "a.pdf" {
		t.Errorf("got %d rows starting at %s, want 5 starting at a.pdf", len(got), got[0].Path)
	}
}

// TestTwoRowsOfTheSameRatioKeepAStableOrder, so a rerun does not shuffle a report.
func TestTwoRowsOfTheSameRatioKeepAStableOrder(t *testing.T) {
	pool := []compare.Result{
		{Path: "b.pdf", Page: 1, Ours: ms(200), Theirs: ms(100)},
		{Path: "a.pdf", Page: 2, Ours: ms(200), Theirs: ms(100)},
		{Path: "a.pdf", Page: 1, Ours: ms(200), Theirs: ms(100)},
	}
	got := confirmWorst(pool, 0, 5, nil)
	want := []struct {
		path string
		page int
	}{{"a.pdf", 1}, {"a.pdf", 2}, {"b.pdf", 1}}
	for i, w := range want {
		if got[i].Path != w.path || got[i].Page != w.page {
			t.Errorf("row %d is %s page %d, want %s page %d", i, got[i].Path, got[i].Page, w.path, w.page)
		}
	}
}

// TestTheTimingsFileCarriesTheConfirmedDuration.
//
// The file and the report disagreed, and the FILE was wrong: the walk records one
// timing per page, confirmWorst draws the named rows again, and nothing carried the
// second reading back. At render v0.60.0 the walk read bulletindedepart23dutc.pdf at
// 1 219 ms -- 7.7 times what the page costs -- the report dropped it, and §24's
// "Where we lose" table, which is built from the file, would have published it as the
// corpus's worst page at 3.05x.
func TestTheTimingsFileCarriesTheConfirmedDuration(t *testing.T) {
	rs := []compare.Result{
		{Path: "/c/noisy.pdf", Page: 1, Ours: ms(1219), Theirs: ms(399), Share: 0.000005},
		{Path: "/c/plain.pdf", Page: 1, Ours: ms(100), Theirs: ms(200), Share: 0.001},
	}
	// The confirmation drew the noisy one again and found what it really costs.
	confirmed := []compare.Result{
		{Path: "/c/noisy.pdf", Page: 1, Ours: ms(158), Theirs: ms(399), Share: 0.000005},
	}
	var out strings.Builder
	writeTimingRows(&out, "pop", rs, confirmed)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("wrote %d rows, want 2:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], fmt.Sprint(int64(ms(158)))) {
		t.Errorf("the confirmed row kept the walk's reading:\n%s", lines[0])
	}
	if strings.Contains(lines[0], fmt.Sprint(int64(ms(1219)))) {
		t.Errorf("the walk's 1219ms survived into the file:\n%s", lines[0])
	}
	// A row nothing confirmed keeps what the walk read, which is all there is.
	if !strings.Contains(lines[1], fmt.Sprint(int64(ms(100)))) {
		t.Errorf("an unconfirmed row lost its reading:\n%s", lines[1])
	}
	// The order of the walk is kept, so a file can be diffed against another.
	if !strings.HasPrefix(lines[0], "pop\tnoisy.pdf\t1\t") {
		t.Errorf("row 0 is %q", lines[0])
	}
}

// TestAHungToolIsNamedInTheFile, because a page the judge did not finish has a
// duration that is the bound rather than a measurement, and a reader has to be able
// to tell.
func TestAHungToolIsNamedInTheFile(t *testing.T) {
	var out strings.Builder
	writeTimingRows(&out, "pop", []compare.Result{
		{Path: "/c/a.pdf", Page: 1, Ours: ms(1), Theirs: ms(2), Share: -1, Tool: "pdftoppm"},
		{Path: "/c/b.pdf", Page: 1, Ours: ms(1), Theirs: ms(2), Share: 0},
	}, nil)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if !strings.HasSuffix(lines[0], "\tpdftoppm") {
		t.Errorf("the hung tool is not named: %q", lines[0])
	}
	// "-" rather than an empty field, so the last column cannot be mistaken for a
	// missing one.
	if !strings.HasSuffix(lines[1], "\t-") {
		t.Errorf("a row with no hang should end in a dash: %q", lines[1])
	}
}
