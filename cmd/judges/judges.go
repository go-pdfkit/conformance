package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	renderDPI   = 96
	pdfiumScale = "1.3333" // 96/72, so pdfium's render matches the others' size
	thumbWidth  = 400      // renders are compared after downsampling to this width
	diffThresh  = 48       // a pixel "differs" when its grey level moves by more than this, of 255
)

// pageDiff is one judge's render of one page measured against poppler's.
type pageDiff struct {
	MeanDiff float64 `json:"mean_diff"` // mean grey Δ, 0..255
	DiffPct  float64 `json:"diff_pct"`  // share of pixels with |Δ| > diffThresh, in %
}

// verdict is one judge's reading of one PDF.
type verdict struct {
	Judge     string           `json:"judge"`
	Skipped   bool             `json:"skipped,omitempty"`
	OK        bool             `json:"ok"`
	Err       string           `json:"err,omitempty"`
	Warnings  int              `json:"warnings"`
	Pages     int              `json:"pages"`      // 0 when the judge reports none
	TextChar  int              `json:"text_chars"` // -1 when the judge extracts none
	Renders   map[int]string   `json:"renders,omitempty"`
	Diffs     map[int]pageDiff `json:"diffs,omitempty"` // vs poppler, per sampled page
	WorstPct  float64          `json:"worst_pct"`
	WorstPage int              `json:"worst_page"`
	Ms        int64            `json:"ms"`
}

// fileResult is every judge's reading of one PDF, and how far they are from
// one another.
type fileResult struct {
	File         string          `json:"file"`
	Bytes        int64           `json:"bytes"`
	SampledPages []int           `json:"sampled_pages"`
	Verdicts     []verdict       `json:"verdicts"`
	Consensus    map[int]float64 `json:"consensus"` // per page: mean pairwise diff% among all judges' renders
	ConsensusMax float64         `json:"consensus_max"`
	ConsensusPg  int             `json:"consensus_page"`
}

// A judge is one reader: whether this machine has it, and how to ask it.
type judge struct {
	name  string
	avail func() bool
	run   func(ctx context.Context, pdf, outDir string, pages []int) verdict
}

var (
	pdfiumBin    string        // pdfium_test, Chrome's engine; empty when it is not built
	nodeDir      string        // directory holding pdfjs-*.mjs and node_modules
	judgeTimeout time.Duration // how long one judge may take on one file
)

// lookPath is a variable so a test can decide which readers this machine has.
var lookPath = exec.LookPath

func have(bin string) bool { _, err := lookPath(bin); return err == nil }

// runCmd runs one tool under the judge's deadline and returns what it said on
// each stream. It is a variable so every judge can be exercised on a machine
// that has none of their binaries — the CI runner is one.
//
// It does not go through internal/poppler, because it needs what that
// deliberately leaves out: stderr on its own, since a judge's warning count is
// the number of lines it complained on; and a working directory, since
// pdfium_test writes beside its input and pdf.js resolves node_modules from
// where it is run. What it keeps is poppler.Run's one rule — a deadline that
// passed is read off the CONTEXT and not off the error, because a killed
// process reports a signal, and "signal: killed" in a report cell is a hang
// nobody can tell from a crash.
var runCmd = func(ctx context.Context, dir string, name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err = cmd.Run()
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		err = &hang{tool: name}
	}
	return o.String(), e.String(), err
}

// A hang is a tool that did not answer within the judge's bound.
type hang struct{ tool string }

func (h *hang) Error() string {
	return "hung: " + h.tool + " did not finish within " + judgeTimeout.String()
}

func countLines(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// textLen counts the non-whitespace runes of an extraction. Whitespace is
// left out because judges disagree wildly on it for reasons that are not
// about the file — Ghostscript's txtwrite pads lines to reproduce column
// layout, pdfium separates every glyph run — while the glyphs they recover
// are what the comparison is about.
func textLen(s string) int {
	n := 0
	for _, r := range s {
		if r > ' ' {
			n++
		}
	}
	return n
}

var rePages = regexp.MustCompile(`(?mi)^Pages:\s+(\d+)`)

func pagesFrom(s string) int {
	if m := rePages.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// samplePages picks the pages whose renders are compared: the first, the
// middle and the last. Text and page counts cover every page regardless;
// this bounds the render work while still looking past page 1.
func samplePages(n int) []int {
	if n <= 1 {
		return []int{1}
	}
	set := map[int]bool{1: true, (n + 1) / 2: true, n: true}
	var out []int
	for p := range set {
		out = append(out, p)
	}
	sort.Ints(out)
	return out
}

// exitCode is the status a tool ended with, or -1 when it did not end with
// one.
func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

// ---- judges -------------------------------------------------------------

func judgeQpdf(ctx context.Context, pdf, outDir string, _ []int) verdict {
	v := verdict{Judge: "qpdf", TextChar: -1}
	out, errs, err := runCmd(ctx, "", "qpdf", "--check", pdf)
	all := out + errs
	v.Warnings = strings.Count(all, "WARNING")
	switch {
	case err == nil:
		v.OK = true
	case exitCode(err) == 3:
		v.OK = true // qpdf's "warnings only"
	default:
		v.Err = firstLine(all, err)
	}
	return v
}

// judgePoppler is the reference. It is not privileged as the truth — the
// consensus column exists so that it is not — but every other judge's Δ is
// taken against it, because a distance needs a second point and poppler is
// the reader that is on every machine this runs on.
func judgePoppler(ctx context.Context, pdf, outDir string, pages []int) verdict {
	v := verdict{Judge: "poppler", Renders: map[int]string{}}
	info, e1, err := runCmd(ctx, "", "pdfinfo", pdf)
	if err != nil {
		v.Err = firstLine(e1, err)
		return v
	}
	v.Pages = pagesFrom(info)
	txt, e2, err := runCmd(ctx, "", "pdftotext", pdf, "-")
	if err != nil {
		v.Err = firstLine(e2, err)
		return v
	}
	v.TextChar = textLen(txt)
	v.Warnings = countLines(e1) + countLines(e2)
	for _, p := range pages {
		base := filepath.Join(outDir, fmt.Sprintf("poppler_p%d", p))
		ps := strconv.Itoa(p)
		_, e3, err := runCmd(ctx, "", "pdftoppm", "-png", "-r", strconv.Itoa(renderDPI), "-f", ps, "-l", ps, "-singlefile", pdf, base)
		if err != nil {
			v.Err = firstLine(e3, err)
			return v
		}
		v.Renders[p] = base + ".png"
		v.Warnings += countLines(e3)
	}
	v.OK = true
	return v
}

func judgeMupdf(ctx context.Context, pdf, outDir string, pages []int) verdict {
	v := verdict{Judge: "mupdf", Renders: map[int]string{}}
	info, e0, _ := runCmd(ctx, "", "mutool", "info", pdf)
	v.Pages = pagesFrom(info)
	txt, e1, err := runCmd(ctx, "", "mutool", "draw", "-q", "-F", "txt", "-o", "-", pdf)
	if err != nil {
		v.Err = firstLine(e1, err)
		return v
	}
	v.TextChar = textLen(txt)
	v.Warnings = countLines(e0) + countLines(e1)
	for _, p := range pages {
		out := filepath.Join(outDir, fmt.Sprintf("mupdf_p%d.png", p))
		_, e2, err := runCmd(ctx, "", "mutool", "draw", "-q", "-r", strconv.Itoa(renderDPI), "-o", out, pdf, strconv.Itoa(p))
		if err != nil {
			v.Err = firstLine(e2, err)
			return v
		}
		v.Renders[p] = out
		v.Warnings += countLines(e2)
	}
	v.OK = true
	return v
}

// judgeGs reports no page count: Ghostscript has no "info" verb, and a count
// read off txtwrite's output would be a count of what it chose to emit.
func judgeGs(ctx context.Context, pdf, outDir string, pages []int) verdict {
	v := verdict{Judge: "gs", Renders: map[int]string{}}
	txtFile := filepath.Join(outDir, "gs.txt")
	o1, e1, err := runCmd(ctx, "", "gs", "-q", "-dNOPAUSE", "-dBATCH", "-dSAFER", "-sDEVICE=txtwrite", "-sOutputFile="+txtFile, pdf)
	if err != nil {
		v.Err = firstLine(o1+e1, err)
		return v
	}
	if b, err := os.ReadFile(txtFile); err == nil {
		v.TextChar = textLen(string(b))
	}
	v.Warnings = countLines(o1 + e1)
	for _, p := range pages {
		out := filepath.Join(outDir, fmt.Sprintf("gs_p%d.png", p))
		ps := strconv.Itoa(p)
		o2, e2, err := runCmd(ctx, "", "gs", "-q", "-dNOPAUSE", "-dBATCH", "-dSAFER", "-sDEVICE=png16m", "-r"+strconv.Itoa(renderDPI),
			"-dFirstPage="+ps, "-dLastPage="+ps, "-sOutputFile="+out, pdf)
		if err != nil {
			v.Err = firstLine(o2+e2, err)
			return v
		}
		v.Renders[p] = out
		v.Warnings += countLines(o2 + e2)
	}
	v.OK = true
	return v
}

// pdfiumNoise is what pdfium_test narrates on stderr as it goes ("Processing
// PDF file x.", "Processed N pages.") — progress, not warnings; only anything
// else counts as one.
var pdfiumNoise = regexp.MustCompile(`(?m)^(Processing PDF file .*|Processed \d+ pages\.)\n?`)

func judgePdfium(ctx context.Context, pdf, outDir string, pages []int) verdict {
	v := verdict{Judge: "pdfium", Renders: map[int]string{}}
	// pdfium_test writes <file>.<page>.png / .txt beside the input; work on a
	// copy in outDir so the corpus tree stays clean.
	work := filepath.Join(outDir, "pdfium.pdf")
	b, err := os.ReadFile(pdf)
	if err != nil {
		v.Err = err.Error()
		return v
	}
	if err := os.WriteFile(work, b, 0o644); err != nil {
		v.Err = err.Error()
		return v
	}
	defer os.Remove(work)
	o1, e1, err := runCmd(ctx, outDir, pdfiumBin, "--txt", "pdfium.pdf")
	if err != nil {
		v.Err = firstLine(o1+e1, err)
		return v
	}
	txts, _ := filepath.Glob(filepath.Join(outDir, "pdfium.pdf.*.txt"))
	v.Pages = len(txts)
	total := 0
	for _, t := range txts {
		if tb, err := os.ReadFile(t); err == nil {
			total += textLen(utf32leToString(tb))
		}
		os.Remove(t)
	}
	v.TextChar = total
	v.Warnings = countLines(pdfiumNoise.ReplaceAllString(e1, ""))
	for _, p := range pages {
		o2, e2, err := runCmd(ctx, outDir, pdfiumBin, "--png", "--scale="+pdfiumScale, "--pages="+strconv.Itoa(p-1), "pdfium.pdf")
		if err != nil {
			v.Err = firstLine(o2+e2, err)
			return v
		}
		src := filepath.Join(outDir, fmt.Sprintf("pdfium.pdf.%d.png", p-1))
		out := filepath.Join(outDir, fmt.Sprintf("pdfium_p%d.png", p))
		if err := os.Rename(src, out); err != nil {
			v.Err = "no render for page " + strconv.Itoa(p)
			return v
		}
		v.Renders[p] = out
		v.Warnings += countLines(pdfiumNoise.ReplaceAllString(e2, ""))
	}
	v.OK = true
	return v
}

// utf32leToString decodes pdfium_test's --txt output — UTF-32LE with a
// byte-order mark (FF FE 00 00; verified with xxd, four bytes per character)
// — so its length is counted in characters like every other judge's, not in
// bytes, which would read as four times the text.
func utf32leToString(b []byte) string {
	if len(b) >= 4 && b[0] == 0xFF && b[1] == 0xFE && b[2] == 0 && b[3] == 0 {
		b = b[4:]
	}
	r := make([]rune, 0, len(b)/4)
	for i := 0; i+3 < len(b); i += 4 {
		r = append(r, rune(uint32(b[i])|uint32(b[i+1])<<8|uint32(b[i+2])<<16|uint32(b[i+3])<<24))
	}
	return string(r)
}

var rePdfjsPages = regexp.MustCompile(`(?m)^pages (\d+)`)

func judgePdfjs(ctx context.Context, pdf, outDir string, pages []int) verdict {
	v := verdict{Judge: "pdfjs", Renders: map[int]string{}}
	out, e1, err := runCmd(ctx, nodeDir, "node", "pdfjs-text.mjs", pdf)
	if err != nil {
		v.Err = firstLine(e1, err)
		return v
	}
	if m := rePdfjsPages.FindStringSubmatch(out); m != nil {
		v.Pages, _ = strconv.Atoi(m[1])
		out = out[len(m[0]):]
	}
	v.TextChar = textLen(out)
	v.Warnings = countLines(e1)
	for _, p := range pages {
		render := filepath.Join(outDir, fmt.Sprintf("pdfjs_p%d.png", p))
		_, e2, err := runCmd(ctx, nodeDir, "node", "pdfjs-render.mjs", pdf, render, strconv.Itoa(p), pdfiumScale)
		if err != nil {
			v.Err = firstLine(e2, err)
			return v
		}
		v.Renders[p] = render
		v.Warnings += countLines(e2)
	}
	v.OK = true
	return v
}

// judgeQuartz renders page 1 only: sips has no page selection. Its render is
// on a transparent background, which is why greyRows composites over white.
func judgeQuartz(ctx context.Context, pdf, outDir string, _ []int) verdict {
	v := verdict{Judge: "quartz", TextChar: -1, Renders: map[int]string{}}
	out := filepath.Join(outDir, "quartz_p1.png")
	o, e, err := runCmd(ctx, "", "sips", "-s", "format", "png", pdf, "--out", out)
	if err != nil {
		v.Err = firstLine(o+e, err)
		return v
	}
	v.Renders[1] = out
	v.Warnings = strings.Count(o+e, "Error") + strings.Count(o+e, "Warning")
	v.OK = true
	return v
}

// firstLine is what a failed judge's cell reads: the first thing the tool
// said, or the error when it said nothing.
func firstLine(s string, err error) string {
	// A hang is said as a hang, whatever the tool managed to print before it
	// was killed: the stderr of a killed pdftoppm is its warnings so far, and
	// the first of those is not why the cell is red.
	var h *hang
	if errors.As(err, &h) {
		return err.Error()
	}
	for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if len(l) > 120 {
				l = l[:120] + "…"
			}
			return l
		}
	}
	return err.Error()
}
