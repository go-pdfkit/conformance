package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdfkit/conformance/internal/mdreport"
)

// now is a variable so a test can know what date the report will carry.
var now = time.Now

// writeReport writes the Markdown table above mdreport's marker, keeping
// whatever a reader wrote beneath it on the last run.
func writeReport(path string, results []fileResult, judges []judge) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Judged by every reader on this machine — %s\n\n", now().UTC().Format("2006-01-02"))
	b.WriteString("Judges: ")
	var names []string
	for _, j := range judges {
		if j.avail() {
			names = append(names, j.name)
		}
	}
	b.WriteString(strings.Join(names, ", "))
	fmt.Fprintf(&b, ". Versions: poppler %s; mupdf %s; gs %s; qpdf %s; pdf.js %s; macOS %s.\n\n",
		version("pdftoppm", "-v"), version("mutool", "-v"), version("gs", "--version"), version("qpdf", "--version"),
		pdfjsVersion(), version("sw_vers", "-productVersion"))
	fmt.Fprintf(&b, "Cell format: `pages · text ratio · Δworst (page)` — pages the judge reports (– if none), "+
		"its extracted-text length as a ratio of poppler's (– if it extracts none), and the largest distance of its "+
		"renders from poppler's over the sampled pages (first, middle, last, at %d dpi: share of pixels whose grey "+
		"level moves by more than %d/255 after both are downsampled to %d px wide), with the page it happened on. "+
		"`consensus` is the mean pairwise distance between all judges' renders of a page, worst page — no reader "+
		"privileged. ⚠n = n warning lines on stderr; ❌ = the judge failed to process the file.\n\n",
		renderDPI, diffThresh, thumbWidth)

	b.WriteString("| PDF | Bytes |")
	for _, n := range names {
		b.WriteString(" " + n + " |")
	}
	b.WriteString(" consensus |\n|---|---|")
	for range names {
		b.WriteString("---|")
	}
	b.WriteString("---|\n")
	for _, fr := range results {
		fmt.Fprintf(&b, "| %s | %s |", fr.File, fmtBytes(fr.Bytes))
		var popplerText int
		for _, v := range fr.Verdicts {
			if v.Judge == "poppler" {
				popplerText = v.TextChar
			}
		}
		for _, v := range fr.Verdicts {
			if v.Skipped {
				continue
			}
			b.WriteString(" " + cell(v, popplerText) + " |")
		}
		fmt.Fprintf(&b, " %.1f%% (p%d) |\n", fr.ConsensusMax, fr.ConsensusPg)
	}
	b.WriteString("\n")
	return mdreport.Write(path, b.String())
}

// cell is one judge's verdict on one file, as the table reads it.
func cell(v verdict, popplerText int) string {
	if !v.OK {
		return "❌ " + v.Err
	}
	var parts []string
	if v.Pages > 0 {
		parts = append(parts, strconv.Itoa(v.Pages)+"p")
	} else {
		parts = append(parts, "–")
	}
	switch {
	case v.TextChar < 0:
		parts = append(parts, "–")
	case popplerText > 0:
		parts = append(parts, fmt.Sprintf("%.3f", float64(v.TextChar)/float64(popplerText)))
	default:
		parts = append(parts, strconv.Itoa(v.TextChar))
	}
	switch {
	case v.Judge == "poppler":
		parts = append(parts, "ref")
	case len(v.Diffs) > 0 && v.Judge == "quartz":
		parts = append(parts, fmt.Sprintf("Δ%.1f%% (p1 only)", v.WorstPct))
	case len(v.Diffs) > 0:
		parts = append(parts, fmt.Sprintf("Δ%.1f%% (p%d)", v.WorstPct, v.WorstPage))
	default:
		parts = append(parts, "–")
	}
	s := "✅ " + strings.Join(parts, " · ")
	if v.Warnings > 0 {
		s += fmt.Sprintf(" ⚠%d", v.Warnings)
	}
	return s
}

func fmtBytes(n int64) string {
	switch {
	case n >= 1e6:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	case n >= 1e3:
		return fmt.Sprintf("%.0f KB", float64(n)/1e3)
	}
	return fmt.Sprintf("%d B", n)
}

// version is the first line a tool prints about itself, on whichever stream
// it chose: pdftoppm and mutool answer -v on stderr, gs and qpdf on stdout.
// The judge is half the measurement, so the report says which one it was.
func version(bin string, args ...string) string {
	o, e, err := runCmd(context.Background(), "", bin, args...)
	out := o + e
	if err != nil && out == "" {
		return "?"
	}
	l := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	if len(l) > 60 {
		l = l[:60]
	}
	return l
}

// pdfjsVersion is read off the installed package, since pdf.js has no binary
// to ask.
func pdfjsVersion() string {
	b, err := os.ReadFile(filepath.Join(nodeDir, "node_modules", "pdfjs-dist", "package.json"))
	if err != nil {
		return "?"
	}
	var p struct{ Version string }
	if json.Unmarshal(b, &p) != nil {
		return "?"
	}
	return p.Version
}

// writeResults writes every verdict as JSON, for whatever wants to read the
// run back without parsing a table.
func writeResults(path string, results []fileResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
