package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

// ---- render comparison ----------------------------------------------------

// greyThumb decodes a PNG and returns it box-downsampled to width w as 8-bit
// grey rows.
func greyThumb(path string, w int) ([][]uint8, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return greyRows(img, w)
}

// greyRows composites an image over white and box-downsamples it to width w,
// as 8-bit grey rows. There is always at least one row and every row is w
// wide, which is what lets diffRows walk two of them without a guard.
func greyRows(img image.Image, w int) ([][]uint8, error) {
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return nil, fmt.Errorf("empty image")
	}
	h := b.Dy() * w / b.Dx()
	if h == 0 {
		h = 1
	}
	rows := make([][]uint8, h)
	for y := 0; y < h; y++ {
		rows[y] = make([]uint8, w)
		y0, y1 := b.Min.Y+y*b.Dy()/h, b.Min.Y+(y+1)*b.Dy()/h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0, x1 := b.Min.X+x*b.Dx()/w, b.Min.X+(x+1)*b.Dx()/w
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var sum, n int
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					// Composite over white first: Quartz (sips) renders a page
					// on a transparent background, and a transparent pixel
					// converted straight to grey is black — every such render
					// would read as a 99% mismatch against an opaque one.
					r, g, bl, a := img.At(xx, yy).RGBA()
					if a < 0xffff {
						r += 0xffff - a
						g += 0xffff - a
						bl += 0xffff - a
					}
					grey := (19595*r + 38470*g + 7471*bl + 1<<15) >> 24 // 0..255
					sum += int(grey)
					n++
				}
			}
			rows[y][x] = uint8(sum / n)
		}
	}
	return rows, nil
}

// thumbCache keeps each render's downsampled grey rows so the pairwise
// consensus doesn't decode the same PNG once per pair.
var thumbCache = map[string][][]uint8{}

func thumb(path string) ([][]uint8, error) {
	if t, ok := thumbCache[path]; ok {
		return t, nil
	}
	t, err := greyThumb(path, thumbWidth)
	if err == nil {
		thumbCache[path] = t
	}
	return t, err
}

// compareRenders returns the mean absolute grey difference and the share of
// pixels differing by more than diffThresh between two renders of the same
// page, over the height both cover.
func compareRenders(a, b string) (mean, pct float64, err error) {
	ra, err := thumb(a)
	if err != nil {
		return 0, 0, err
	}
	rb, err := thumb(b)
	if err != nil {
		return 0, 0, err
	}
	mean, pct = diffRows(ra, rb)
	return mean, pct, nil
}

// diffRows is compareRenders on rows already decoded: the mean absolute
// difference and the share, in %, of pixels whose difference passes
// diffThresh, over the area both cover.
func diffRows(ra, rb [][]uint8) (mean, pct float64) {
	h := min(len(ra), len(rb))
	var sum, big, n int
	for y := 0; y < h; y++ {
		w := min(len(ra[y]), len(rb[y]))
		for x := 0; x < w; x++ {
			d := int(ra[y][x]) - int(rb[y][x])
			if d < 0 {
				d = -d
			}
			sum += d
			if d > diffThresh {
				big++
			}
			n++
		}
	}
	if n == 0 {
		return 0, 0
	}
	return float64(sum) / float64(n), 100 * float64(big) / float64(n)
}

// score fills each verdict's per-page distance to poppler and its worst
// page, then the per-page consensus: the mean pairwise distance between every
// two judges' renders of that page, poppler included as just one of them.
func score(fr *fileResult) {
	var ref *verdict
	for i := range fr.Verdicts {
		if fr.Verdicts[i].Judge == "poppler" {
			ref = &fr.Verdicts[i]
		}
	}
	for i := range fr.Verdicts {
		v := &fr.Verdicts[i]
		if ref == nil || v == ref || len(v.Renders) == 0 {
			continue
		}
		v.Diffs = map[int]pageDiff{}
		for p, path := range v.Renders {
			rp, ok := ref.Renders[p]
			if !ok {
				continue
			}
			m, pct, err := compareRenders(rp, path)
			if err != nil {
				v.Err = "compare: " + err.Error()
				continue
			}
			v.Diffs[p] = pageDiff{m, pct}
			if pct >= v.WorstPct {
				v.WorstPct, v.WorstPage = pct, p
			}
		}
	}
	fr.Consensus = map[int]float64{}
	for _, p := range fr.SampledPages {
		var paths []string
		for _, v := range fr.Verdicts {
			if path, ok := v.Renders[p]; ok && v.OK {
				paths = append(paths, path)
			}
		}
		var sum float64
		var n int
		for i := 0; i < len(paths); i++ {
			for j := i + 1; j < len(paths); j++ {
				if _, pct, err := compareRenders(paths[i], paths[j]); err == nil {
					sum += pct
					n++
				}
			}
		}
		if n > 0 {
			fr.Consensus[p] = sum / float64(n)
			if fr.Consensus[p] >= fr.ConsensusMax {
				fr.ConsensusMax, fr.ConsensusPg = fr.Consensus[p], p
			}
		}
	}
}
