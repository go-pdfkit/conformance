package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestATransparentPixelIsWhiteNotBlack(t *testing.T) {
	// Quartz renders a page on a transparent background, and a transparent
	// pixel converted straight to grey is black — every such render would
	// read as a 99% mismatch against an opaque one.
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{0, 0, 0, 0})
	img.SetNRGBA(1, 0, color.NRGBA{0, 0, 0, 255})
	rows, err := greyRows(img, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0][0] != 255 || rows[0][1] != 0 {
		t.Errorf("got %v", rows)
	}
}

func TestGreyRowsAlwaysHasARowAndFullWidth(t *testing.T) {
	// A page much wider than tall downsamples to no rows at all unless one
	// is kept; a page smaller than the thumbnail is sampled up rather than
	// dividing by nothing. Both are what lets diffRows walk two results
	// without guards.
	wide := image.NewGray(image.Rect(0, 0, 1000, 1))
	rows, err := greyRows(wide, 400)
	if err != nil || len(rows) != 1 || len(rows[0]) != 400 {
		t.Errorf("a 1000×1 page: %d rows, %v", len(rows), err)
	}
	tiny := image.NewGray(image.Rect(0, 0, 1, 2))
	tiny.SetGray(0, 1, color.Gray{Y: 255})
	rows, err = greyRows(tiny, 4)
	if err != nil || len(rows) != 8 || rows[0][3] != 0 || rows[7][0] != 255 {
		t.Errorf("a 1×2 page: %d rows, %v", len(rows), err)
	}
	if _, err := greyRows(image.NewGray(image.Rect(0, 0, 0, 0)), 4); err == nil {
		t.Error("an empty image was downsampled")
	}
}

func TestGreyThumbReadsAPNGAndNothingElse(t *testing.T) {
	dir := t.TempDir()
	if _, err := greyThumb(filepath.Join(dir, "absent.png"), 4); err == nil {
		t.Error("an absent file was read")
	}
	notPNG := filepath.Join(dir, "not.png")
	os.WriteFile(notPNG, []byte("%PDF-1.7"), 0o644)
	if _, err := greyThumb(notPNG, 4); err == nil {
		t.Error("a PDF was decoded as a PNG")
	}
	p := filepath.Join(dir, "page.png")
	writePNG(t, p, 8, 8, image.Rect(0, 0, 4, 8))
	rows, err := greyThumb(p, 2)
	if err != nil || len(rows) != 2 || rows[0][0] != 0 || rows[0][1] != 255 {
		t.Errorf("got %v, %v", rows, err)
	}
}

func TestThumbsAreDecodedOnce(t *testing.T) {
	// The pairwise consensus would otherwise decode the same PNG once per
	// pair.
	dir := t.TempDir()
	p := filepath.Join(dir, "page.png")
	writePNG(t, p, 8, 8, image.Rect(0, 0, 8, 8))
	if _, err := thumb(p); err != nil {
		t.Fatal(err)
	}
	os.Remove(p)
	if rows, err := thumb(p); err != nil || rows[0][0] != 0 {
		t.Errorf("the cache did not answer: %v", err)
	}
	absent := filepath.Join(dir, "absent.png")
	if _, err := thumb(absent); err == nil {
		t.Fatal("an absent file was read")
	}
	if _, ok := thumbCache[absent]; ok {
		t.Error("a failure was cached")
	}
}

func TestDiffRowsCountsWhatPassesTheThreshold(t *testing.T) {
	a := [][]uint8{{0, 0, 255, 255, 100}}
	b := [][]uint8{{0, 255, 255, 0, 140}}
	// Differences 0, 255, 0, 255, 40: two of five pass 48, and the mean is
	// 550/5.
	mean, pct := diffRows(a, b)
	if mean != 110 || pct != 40 {
		t.Errorf("mean %v pct %v", mean, pct)
	}
	// Only the area both cover is compared.
	if mean, pct := diffRows(a, [][]uint8{{0, 0}, {9, 9}}); mean != 0 || pct != 0 {
		t.Errorf("over the overlap: mean %v pct %v", mean, pct)
	}
	if mean, pct := diffRows(nil, a); mean != 0 || pct != 0 {
		t.Errorf("nothing to compare: mean %v pct %v", mean, pct)
	}
}

func TestCompareRendersNamesTheFileItCouldNotRead(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.png"), filepath.Join(dir, "b.png")
	writePNG(t, a, 100, 100, image.Rect(0, 0, 50, 100))
	writePNG(t, b, 100, 100, image.Rect(0, 0, 100, 100))
	if _, _, err := compareRenders(filepath.Join(dir, "absent.png"), b); err == nil {
		t.Error("an absent first render was compared")
	}
	if _, _, err := compareRenders(a, filepath.Join(dir, "absent.png")); err == nil {
		t.Error("an absent second render was compared")
	}
	// Half of a is white where b is black.
	mean, pct, err := compareRenders(a, b)
	if err != nil || pct != 50 || mean != 127.5 {
		t.Errorf("mean %v pct %v, %v", mean, pct, err)
	}
}

// pages writes a render per page for one judge, with the ink at off.
func pages(t *testing.T, dir, judge string, off int, ps ...int) map[int]string {
	t.Helper()
	m := map[int]string{}
	for _, p := range ps {
		m[p] = filepath.Join(dir, judge+"_p"+string(rune('0'+p))+".png")
		writePNG(t, m[p], 100, 100, image.Rect(off, 0, off+50, 100))
	}
	return m
}

func TestScoreMeasuresAgainstPopplerAndThenAmongEveryone(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "gs_p1.png")
	os.WriteFile(bad, []byte("not a png"), 0o644)
	fr := fileResult{SampledPages: []int{1, 2}, Verdicts: []verdict{
		{Judge: "qpdf", OK: true},
		{Judge: "poppler", OK: true, Renders: pages(t, dir, "poppler", 0, 1, 2)},
		// A page the reference did not render is not measured.
		{Judge: "mupdf", OK: true, Renders: pages(t, dir, "mupdf", 10, 1, 2, 3)},
		// A render that cannot be read is said so, and does not stop the rest.
		{Judge: "gs", OK: true, Renders: map[int]string{1: bad}},
		// A judge that failed is left out of the consensus even if it drew.
		{Judge: "pdfium", OK: false, Renders: pages(t, dir, "pdfium", 50, 1, 2)},
		{Judge: "quartz", OK: true, Renders: pages(t, dir, "quartz", 20, 1)},
	}}
	score(&fr)
	mupdf, gs, pdfium, quartz := fr.Verdicts[2], fr.Verdicts[3], fr.Verdicts[4], fr.Verdicts[5]
	if len(mupdf.Diffs) != 2 || mupdf.Diffs[1].DiffPct != 20 || mupdf.WorstPct != 20 {
		t.Errorf("mupdf: %+v", mupdf)
	}
	if !strings.HasPrefix(gs.Err, "compare: ") || len(gs.Diffs) != 0 {
		t.Errorf("gs: %+v", gs)
	}
	if pdfium.WorstPct != 100 || quartz.WorstPct != 40 {
		t.Errorf("pdfium worst %v, quartz worst %v", pdfium.WorstPct, quartz.WorstPct)
	}
	// Page 1: poppler, mupdf and quartz can be paired (gs cannot be read,
	// pdfium failed): (20 + 40 + 20) / 3. Page 2: poppler and mupdf: 20.
	if fr.Consensus[1] != 80.0/3 || fr.Consensus[2] != 20 || fr.ConsensusPg != 1 {
		t.Errorf("consensus %v, worst page %d", fr.Consensus, fr.ConsensusPg)
	}
}

func TestScoreWithoutPopplerMeasuresNothingAgainstIt(t *testing.T) {
	dir := t.TempDir()
	fr := fileResult{SampledPages: []int{1}, Verdicts: []verdict{
		{Judge: "mupdf", OK: true, Renders: pages(t, dir, "mupdf", 10, 1)},
		{Judge: "quartz", OK: true, Renders: pages(t, dir, "quartz", 20, 1)},
	}}
	score(&fr)
	if len(fr.Verdicts[0].Diffs) != 0 || len(fr.Verdicts[1].Diffs) != 0 {
		t.Errorf("a distance to an absent reference: %+v", fr.Verdicts)
	}
	// But the readers that are there still disagree by a measurable amount.
	if fr.Consensus[1] != 20 {
		t.Errorf("consensus %v", fr.Consensus)
	}
}
