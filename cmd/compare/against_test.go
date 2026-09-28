package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ms and us keep a table of durations readable. Every test in this file needed
// them, so they live here once rather than in each.
func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }
func us(n int) time.Duration { return time.Duration(n) * time.Microsecond }

func writeTimings(t *testing.T, rows string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "timings.tsv")
	head := "population\tdocument\tpage\tours_ns\ttheirs_ns\tshare\thung\n"
	if err := os.WriteFile(p, []byte(head+rows), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestAnEarlierRunIsReadBack covers the file format both directions of this
// feature depend on.
func TestAnEarlierRunIsReadBack(t *testing.T) {
	p := writeTimings(t, "alpha\tone.pdf\t1\t1000000\t2000000\t0.000100\t-\n"+
		"alpha\ttwo.pdf\t1\t3000000\t4000000\t-1.000000\t-\n")
	was, err := readTimings(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(was) != 2 {
		t.Fatalf("read %d rows, want 2", len(was))
	}
	got := was[key("alpha", "one.pdf", "1")]
	if got.ours != time.Millisecond || got.share != 0.0001 {
		t.Errorf("row one came back as %+v", got)
	}
	if was[key("alpha", "two.pdf", "1")].share != -1 {
		t.Error("a page that was never compared lost its -1")
	}
	if _, err := readTimings(filepath.Join(t.TempDir(), "absent.tsv")); err == nil {
		t.Error("a missing file was read without complaint")
	}
	// A row with too few fields, or an unparseable number, is skipped rather
	// than fatal: half a file is better than refusing to look at any of it.
	p2 := writeTimings(t, "short\trow\n"+"alpha\tbad.pdf\t1\tnotanumber\t1\t0.0\t-\n"+
		"alpha\tworse.pdf\t1\t1\t1\tnotafloat\t-\n"+"alpha\tgood.pdf\t1\t500\t600\t0.0\t-\n")
	was2, err := readTimings(p2)
	if err != nil {
		t.Fatal(err)
	}
	if len(was2) != 1 {
		t.Errorf("kept %d rows, want only the good one", len(was2))
	}
}

// TestOnlyARealSlowdownIsReported is the whole point: a page that was blank and
// now draws is slower and is NOT a regression, and a page whose timing moved
// under the threshold is not news.
func TestOnlyARealSlowdownIsReported(t *testing.T) {
	was := map[string]previous{
		key("a", "regressed.pdf", "1"):    {ms(30), 0.001},
		key("a", "same.pdf", "1"):         {ms(30), 0.001},
		key("a", "wasblank.pdf", "1"):     {ms(10), -1},   // not comparable before
		key("a", "wasdiffering.pdf", "1"): {ms(10), 0.45}, // disagreed before
		key("a", "faster.pdf", "1"):       {ms(30), 0.001},
		key("a", "untimed.pdf", "1"):      {0, 0.001},
	}
	now := []timedPage{
		{key: key("a", "regressed.pdf", "1"), ours: ms(360), share: 0.001},
		{key: key("a", "same.pdf", "1"), ours: ms(33), share: 0.001},
		{key: key("a", "wasblank.pdf", "1"), ours: ms(500), share: 0.001},
		{key: key("a", "wasdiffering.pdf", "1"), ours: ms(500), share: 0.001},
		{key: key("a", "faster.pdf", "1"), ours: ms(10), share: 0.001},
		{key: key("a", "untimed.pdf", "1"), ours: ms(500), share: 0.001},
		{key: key("a", "new.pdf", "1"), ours: ms(999), share: 0.001},
	}
	got := regressions(was, now, 2, 0, 0)
	if len(got) != 1 {
		t.Fatalf("reported %d pages, want 1: %+v", len(got), got)
	}
	if !strings.Contains(got[0].key, "regressed.pdf") {
		t.Errorf("reported %q", got[0].key)
	}
	if got[0].factor < 11 || got[0].factor > 13 {
		t.Errorf("factor %v, want about 12", got[0].factor)
	}
}

// TestTiedSlowdownsAreOrderedStably keeps two runs of this report comparable.
func TestTiedSlowdownsAreOrderedStably(t *testing.T) {
	was := map[string]previous{
		key("a", "b.pdf", "1"): {ms(10), 0},
		key("a", "a.pdf", "1"): {ms(10), 0},
	}
	now := []timedPage{
		{key: key("a", "b.pdf", "1"), ours: ms(30), share: 0},
		{key: key("a", "a.pdf", "1"), ours: ms(30), share: 0},
	}
	got := regressions(was, now, 2, 0, 0)
	if len(got) != 2 || !strings.Contains(got[0].key, "a.pdf") {
		t.Errorf("order is %v", got)
	}
}

// TestTheReportSaysWhatMayBeConcluded: the rows are candidates, and a report that
// does not say so invites a chase after noise. Three pages in the run that found
// the memo defect moved by more than 7x for no reason a re-measurement could
// reproduce.
func TestTheReportSaysWhatMayBeConcluded(t *testing.T) {
	var out bytes.Buffer
	reportRegressions(&out, nil, 2)
	if !strings.Contains(out.String(), "no page is 2.0x slower") {
		t.Errorf("a clean run says %q", out.String())
	}
	out.Reset()
	reportRegressions(&out, []slower{{key: key("a", "x.pdf", "1"), was: time.Millisecond, is: 12 * time.Millisecond, factor: 12}}, 2)
	s := out.String()
	for _, want := range []string{"1 page(s) at least 2.0x slower", "CANDIDATES", "Re-measure", "12.00x", "x.pdf page 1"} {
		if !strings.Contains(s, want) {
			t.Errorf("the report does not say %q:\n%s", want, s)
		}
	}
}

// TestAFileThatCannotBeScannedIsAnError: os.Open succeeds on a directory and the
// scan then fails, which is the same shape as a truncated or unreadable file. A
// run that silently compares against nothing is worse than one that stops.
func TestAFileThatCannotBeScannedIsAnError(t *testing.T) {
	if _, err := readTimings(t.TempDir()); err == nil {
		t.Error("a directory was read as a timings file")
	}
}

// TestSlowdownsAreOrderedByFactorFirst, before the tie-break on the name.
func TestSlowdownsAreOrderedByFactorFirst(t *testing.T) {
	was := map[string]previous{
		key("a", "zzz.pdf", "1"): {ms(10), 0},
		key("a", "mmm.pdf", "1"): {ms(10), 0},
		key("a", "aaa.pdf", "1"): {ms(10), 0},
	}
	now := []timedPage{
		{key: key("a", "aaa.pdf", "1"), ours: ms(25), share: 0},  // 2.5x
		{key: key("a", "mmm.pdf", "1"), ours: ms(100), share: 0}, // 10x
		{key: key("a", "zzz.pdf", "1"), ours: ms(50), share: 0},  // 5x
	}
	got := regressions(was, now, 2, 0, 0)
	if len(got) != 3 {
		t.Fatalf("reported %d", len(got))
	}
	for i, want := range []string{"mmm.pdf", "zzz.pdf", "aaa.pdf"} {
		if !strings.Contains(got[i].key, want) {
			t.Errorf("at %d got %q, want %s", i, got[i].key, want)
		}
	}
}

// TestAFactorOnANothingPageIsNotAFinding: the minimum duration exists because
// 0.3 ms becoming 1.5 ms reads as five times slower and says nothing.
func TestAFactorOnANothingPageIsNotAFinding(t *testing.T) {
	was := map[string]previous{
		key("a", "tiny.pdf", "1"): {us(300), 0},
		key("a", "real.pdf", "1"): {ms(30), 0},
	}
	now := []timedPage{
		{key: key("a", "tiny.pdf", "1"), ours: us(1500), share: 0},
		{key: key("a", "real.pdf", "1"), ours: ms(120), share: 0},
	}
	got := regressions(was, now, 2, 10*time.Millisecond, 0)
	if len(got) != 1 || !strings.Contains(got[0].key, "real.pdf") {
		t.Errorf("reported %+v", got)
	}
	// With no floor, both are reported -- which is what the floor is for.
	if len(regressions(was, now, 2, 0, 0)) != 2 {
		t.Error("without a floor the tiny page should still be reported")
	}
}

// TestAFactorWithoutAnIncreaseIsNoise is the second floor, and it is measured
// rather than chosen. Run against a reference taken at a different machine load,
// a factor of 2 with a 10ms duration floor produced thirteen candidates and every
// one was noise -- 0.92x to 1.01x when re-measured in one interleaved run, with
// increases of at most +88ms. The regression this check exists for was +337ms.
func TestAFactorWithoutAnIncreaseIsNoise(t *testing.T) {
	was := map[string]previous{
		key("a", "noise.pdf", "1"): {ms(11), 0}, // 11 -> 49: 4.45x, +38ms
		key("a", "real.pdf", "1"):  {ms(32), 0}, // 32 -> 369: 11.5x, +337ms
	}
	now := []timedPage{
		{key: key("a", "noise.pdf", "1"), ours: ms(49), share: 0},
		{key: key("a", "real.pdf", "1"), ours: ms(369), share: 0},
	}
	got := regressions(was, now, 2, 10*time.Millisecond, 100*time.Millisecond)
	if len(got) != 1 || !strings.Contains(got[0].key, "real.pdf") {
		t.Errorf("reported %+v, want only real.pdf", got)
	}
	// Without the increase floor, both are reported -- which is what it is for.
	if len(regressions(was, now, 2, 10*time.Millisecond, 0)) != 2 {
		t.Error("without the increase floor the noisy page should still be reported")
	}
}

// TestACandidateThatIsFasterWhenDrawnAgainIsDropped. This is the whole reason
// confirm exists: thirteen candidates from one real run were all noise, and no
// arithmetic on the pair (was, is) could say so.
func TestACandidateThatIsFasterWhenDrawnAgainIsDropped(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(100), is: ms(400), factor: 4, path: "x.pdf", page: 1}}
	got := confirm(in, 2, 2, 0, func(string, int) (time.Duration, bool) { return ms(98), true })
	if len(got) != 1 {
		t.Fatalf("confirm returned %d rows, want 1", len(got))
	}
	if !got[0].dropped || got[0].confirmed {
		t.Errorf("dropped=%v confirmed=%v, want dropped", got[0].dropped, got[0].confirmed)
	}
	// The row now carries what was measured, not what was read from the file.
	if got[0].is != ms(98) {
		t.Errorf("is = %v, want the re-measured 98ms", got[0].is)
	}
	if got[0].factor != 0.98 {
		t.Errorf("factor = %v, want 0.98", got[0].factor)
	}
}

// TestACandidateStillSlowerWhenDrawnAgainIsConfirmed.
func TestACandidateStillSlowerWhenDrawnAgainIsConfirmed(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(32), is: ms(369), factor: 11.5, path: "x.pdf", page: 1}}
	got := confirm(in, 2, 2, 0, func(string, int) (time.Duration, bool) { return ms(360), true })
	if !got[0].confirmed || got[0].dropped {
		t.Errorf("confirmed=%v dropped=%v, want confirmed", got[0].confirmed, got[0].dropped)
	}
}

// TestConfirmTakesTheMinimumOfItsTries. A minimum is used rather than a mean
// because the accident this check is built to survive only ever adds time. A mean
// would carry a single 700ms outlier into the verdict.
func TestConfirmTakesTheMinimumOfItsTries(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(100), is: ms(400), factor: 4}}
	seq := []time.Duration{ms(700), ms(90), ms(650)}
	i := 0
	got := confirm(in, 3, 2, 0, func(string, int) (time.Duration, bool) {
		d := seq[i]
		i++
		return d, true
	})
	if got[0].is != ms(90) {
		t.Errorf("is = %v, want the minimum 90ms", got[0].is)
	}
	if !got[0].dropped {
		t.Error("a page whose best try beats the reference is noise")
	}
}

// TestAPageNothingCouldDrawAgainIsKept. An unread answer must not read as a
// negative: a candidate nobody could re-measure stays a candidate, unmarked, so a
// reader sees it and knows nothing confirmed it.
func TestAPageNothingCouldDrawAgainIsKept(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(100), is: ms(400), factor: 4}}
	got := confirm(in, 3, 2, 0, func(string, int) (time.Duration, bool) { return 0, false })
	if len(got) != 1 {
		t.Fatalf("confirm returned %d rows, want the candidate kept", len(got))
	}
	if got[0].confirmed || got[0].dropped {
		t.Error("a page nothing drew again is neither confirmed nor dropped")
	}
	if got[0].is != ms(400) {
		t.Errorf("is = %v, want the original reading left alone", got[0].is)
	}
}

// TestADrawThatReturnsZeroDoesNotCount. ok true with a zero duration is a timing
// that was not taken, and averaging it in would drop every candidate.
func TestADrawThatReturnsZeroDoesNotCount(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(100), is: ms(400), factor: 4}}
	got := confirm(in, 1, 2, 0, func(string, int) (time.Duration, bool) { return 0, true })
	if got[0].confirmed || got[0].dropped {
		t.Error("a zero duration is not a measurement")
	}
}

// TestConfirmWithNoTriesChangesNothing. -confirm 0 is how a reader asks for the
// candidates unchecked, which is what the tool did before this existed.
func TestConfirmWithNoTriesChangesNothing(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(100), is: ms(400), factor: 4}}
	got := confirm(in, 0, 2, 0, func(string, int) (time.Duration, bool) {
		t.Error("confirm drew a page with tries = 0")
		return 0, false
	})
	if len(got) != 1 || got[0].confirmed || got[0].dropped || got[0].is != ms(400) {
		t.Errorf("got %+v, want the input unchanged", got)
	}
}

// TestConfirmPassesThePageItWasAskedAbout. A confirmation that drew a different
// page would answer its own assertion.
func TestConfirmPassesThePageItWasAskedAbout(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "7"), was: ms(100), is: ms(400), factor: 4, path: "/c/x.pdf", page: 7}}
	var gotPath string
	var gotPage int
	confirm(in, 1, 2, 0, func(p string, n int) (time.Duration, bool) {
		gotPath, gotPage = p, n
		return ms(500), true
	})
	if gotPath != "/c/x.pdf" || gotPage != 7 {
		t.Errorf("drew %s page %d, want /c/x.pdf page 7", gotPath, gotPage)
	}
}

// TestTheReportSaysWhichRowsSurvivedReMeasurement.
func TestTheReportSaysWhichRowsSurvivedReMeasurement(t *testing.T) {
	var out strings.Builder
	reportRegressions(&out, []slower{
		{key: key("a", "real.pdf", "1"), was: ms(32), is: ms(369), factor: 11.5, confirmed: true},
		{key: key("a", "noise.pdf", "1"), was: ms(160), is: ms(157), factor: 0.98, dropped: true},
	}, 2)
	got := out.String()
	for _, want := range []string{"! ", "~ ", "real.pdf", "noise.pdf", "1 dropped by re-measurement", "1 confirmed"} {
		if !strings.Contains(got, want) {
			t.Errorf("report does not contain %q:\n%s", want, got)
		}
	}
	// The headline counts what survived, not what was proposed.
	if !strings.Contains(got, "1 page(s) at least") {
		t.Errorf("headline should count the 1 survivor, not 2 candidates:\n%s", got)
	}
}

// TestARowThatComesBackAtTheSameSpeedIsNotAConfirmedRegression.
//
// The defect this pins was shipped and then found by the tool's own output: at
// render v0.58.0 the check drew a page again, got 358ms against the reference's
// 350ms, and marked it CONFIRMED -- under a headline that said "at least 2.0x
// slower". `still a little slower` is not the question. The question is whether the
// re-measurement clears the SAME bars that made the row a candidate.
func TestARowThatComesBackAtTheSameSpeedIsNotAConfirmedRegression(t *testing.T) {
	// Selected on a reading of 4x; drawn again it is 1.02x.
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(350), is: ms(1400), factor: 4}}
	got := confirm(in, 1, 2, 100*time.Millisecond, func(string, int) (time.Duration, bool) {
		return ms(358), true
	})
	if got[0].confirmed {
		t.Errorf("1.02x was reported as a confirmed regression (%v -> %v)", got[0].was, got[0].is)
	}
	if !got[0].dropped {
		t.Error("it should be marked as noise")
	}
}

// TestARowThatClearsTheRatioButNotTheIncreaseIsDropped. Both bars, not either: a
// 12ms page at 3x has gained 8ms, which is the small-page accident -slower exists
// for, and a confirmation that ignored the floor would let it back in.
func TestARowThatClearsTheRatioButNotTheIncreaseIsDropped(t *testing.T) {
	in := []slower{{key: key("a", "x.pdf", "1"), was: ms(4), is: ms(400), factor: 100}}
	got := confirm(in, 1, 2, 100*time.Millisecond, func(string, int) (time.Duration, bool) {
		return ms(12), true // 3x, but only +8ms
	})
	if got[0].confirmed {
		t.Errorf("+8ms on a 4ms page was confirmed at %.2fx", got[0].factor)
	}
}

// TestARealRegressionStillSurvivesConfirmation, so the stricter rule has not made
// the check blind: the one real regression this project found was 32ms -> 369ms.
func TestARealRegressionStillSurvivesConfirmation(t *testing.T) {
	in := []slower{{key: key("a", "cerfa.pdf", "1"), was: ms(32), is: ms(369), factor: 11.5}}
	got := confirm(in, 1, 2, 100*time.Millisecond, func(string, int) (time.Duration, bool) {
		return ms(360), true
	})
	if !got[0].confirmed {
		t.Errorf("32ms -> 360ms was dropped at %.2fx", got[0].factor)
	}
}

// TestARowThatClearsTheIncreaseButNotTheRatioIsDropped. The other half of "both
// bars": a page that already took a second and comes back 150ms slower has cleared
// the absolute floor and is still only 1.15x, which is inside the spread two takes
// of the SAME version show. The floor exists to remove small-page accidents, not to
// admit large-page ones.
func TestARowThatClearsTheIncreaseButNotTheRatioIsDropped(t *testing.T) {
	in := []slower{{key: key("a", "big.pdf", "1"), was: ms(1000), is: ms(3000), factor: 3}}
	got := confirm(in, 1, 2, 100*time.Millisecond, func(string, int) (time.Duration, bool) {
		return ms(1150), true // +150ms, but 1.15x
	})
	if got[0].confirmed {
		t.Errorf("+150ms at %.2fx was confirmed; the ratio bar is not being applied", got[0].factor)
	}
	if !got[0].dropped {
		t.Error("it should be marked as noise")
	}
}
