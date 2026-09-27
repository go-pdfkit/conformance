package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	was := map[string]previous{
		key("a", "regressed.pdf", "1"):    {ms(30), 0.001},
		key("a", "same.pdf", "1"):         {ms(30), 0.001},
		key("a", "wasblank.pdf", "1"):     {ms(10), -1},   // not comparable before
		key("a", "wasdiffering.pdf", "1"): {ms(10), 0.45}, // disagreed before
		key("a", "faster.pdf", "1"):       {ms(30), 0.001},
		key("a", "untimed.pdf", "1"):      {0, 0.001},
	}
	now := []timedPage{
		{key("a", "regressed.pdf", "1"), ms(360), 0.001},
		{key("a", "same.pdf", "1"), ms(33), 0.001},
		{key("a", "wasblank.pdf", "1"), ms(500), 0.001},
		{key("a", "wasdiffering.pdf", "1"), ms(500), 0.001},
		{key("a", "faster.pdf", "1"), ms(10), 0.001},
		{key("a", "untimed.pdf", "1"), ms(500), 0.001},
		{key("a", "new.pdf", "1"), ms(999), 0.001},
	}
	got := regressions(was, now, 2, 0)
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
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	was := map[string]previous{
		key("a", "b.pdf", "1"): {ms(10), 0},
		key("a", "a.pdf", "1"): {ms(10), 0},
	}
	now := []timedPage{
		{key("a", "b.pdf", "1"), ms(30), 0},
		{key("a", "a.pdf", "1"), ms(30), 0},
	}
	got := regressions(was, now, 2, 0)
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
	reportRegressions(&out, []slower{{key("a", "x.pdf", "1"), time.Millisecond, 12 * time.Millisecond, 12}}, 2)
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
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	was := map[string]previous{
		key("a", "zzz.pdf", "1"): {ms(10), 0},
		key("a", "mmm.pdf", "1"): {ms(10), 0},
		key("a", "aaa.pdf", "1"): {ms(10), 0},
	}
	now := []timedPage{
		{key("a", "aaa.pdf", "1"), ms(25), 0},  // 2.5x
		{key("a", "mmm.pdf", "1"), ms(100), 0}, // 10x
		{key("a", "zzz.pdf", "1"), ms(50), 0},  // 5x
	}
	got := regressions(was, now, 2, 0)
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
	us := func(n int) time.Duration { return time.Duration(n) * time.Microsecond }
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	was := map[string]previous{
		key("a", "tiny.pdf", "1"): {us(300), 0},
		key("a", "real.pdf", "1"): {ms(30), 0},
	}
	now := []timedPage{
		{key("a", "tiny.pdf", "1"), us(1500), 0},
		{key("a", "real.pdf", "1"), ms(120), 0},
	}
	got := regressions(was, now, 2, 10*time.Millisecond)
	if len(got) != 1 || !strings.Contains(got[0].key, "real.pdf") {
		t.Errorf("reported %+v", got)
	}
	// With no floor, both are reported -- which is what the floor is for.
	if len(regressions(was, now, 2, 0)) != 2 {
		t.Error("without a floor the tiny page should still be reported")
	}
}
