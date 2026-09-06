package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// run judges every PDF the globs name with every reader the machine has.
func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("judges", flag.ContinueOnError)
	fs.SetOutput(errOut)
	glob := fs.String("pdfs", "", "comma-separated globs of the PDFs to judge")
	outDir := fs.String("out", "out/judges", "directory for per-judge renders")
	resultsPath := fs.String("results", "judges.json", "JSON results path")
	reportPath := fs.String("report", "JUDGES.md", "Markdown report path")
	fs.StringVar(&pdfiumBin, "pdfium", envOr("PDFIUM_TEST", ""), "pdfium_test binary (Chrome's engine); PDFIUM_TEST env")
	fs.StringVar(&nodeDir, "nodedir", "judges", "directory with pdfjs-*.mjs and node_modules")
	fs.DurationVar(&judgeTimeout, "timeout", 180*time.Second, "how long one judge may take on one file before it is called a hang")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *glob == "" {
		fmt.Fprintln(errOut, "judges: -pdfs is needed")
		return 2
	}

	judges := []judge{
		{"qpdf", func() bool { return have("qpdf") }, judgeQpdf},
		{"poppler", func() bool { return have("pdfinfo") && have("pdftoppm") && have("pdftotext") }, judgePoppler},
		{"mupdf", func() bool { return have("mutool") }, judgeMupdf},
		{"gs", func() bool { return have("gs") }, judgeGs},
		{"pdfium", func() bool { _, err := os.Stat(pdfiumBin); return pdfiumBin != "" && err == nil }, judgePdfium},
		{"pdfjs", func() bool {
			_, err := os.Stat(filepath.Join(nodeDir, "node_modules", "pdfjs-dist"))
			return have("node") && err == nil
		}, judgePdfjs},
		{"quartz", func() bool { return have("sips") }, judgeQuartz},
	}

	var files []string
	for _, g := range strings.Split(*glob, ",") {
		m, _ := filepath.Glob(strings.TrimSpace(g))
		files = append(files, m...)
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintln(errOut, "judges: no PDFs matched")
		return 1
	}
	nodeDir, _ = filepath.Abs(nodeDir)

	var results []fileResult
	for _, f := range files {
		abs, _ := filepath.Abs(f)
		fr := fileResult{File: f}
		if st, err := os.Stat(abs); err == nil {
			fr.Bytes = st.Size()
		}
		// Absolute: pdfium and pdf.js run with another working directory.
		dir, _ := filepath.Abs(filepath.Join(*outDir, strings.TrimSuffix(filepath.Base(f), ".pdf")))
		os.MkdirAll(dir, 0o755)
		fmt.Fprintf(errOut, "%s\n", f)
		// Page count for the sample comes from poppler, which runs before the
		// renderers; until then only page 1 is known to exist.
		pages := []int{1}
		for _, j := range judges {
			if !j.avail() {
				fr.Verdicts = append(fr.Verdicts, verdict{Judge: j.name, Skipped: true, TextChar: -1})
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), judgeTimeout)
			t0 := time.Now()
			v := j.run(ctx, abs, dir, pages)
			v.Ms = time.Since(t0).Milliseconds()
			cancel()
			if v.Judge == "poppler" && len(pages) == 1 && v.Pages > 1 {
				// Re-render poppler on the full sample now that the count is known.
				pages = samplePages(v.Pages)
				fr.SampledPages = pages
				ctx, cancel := context.WithTimeout(context.Background(), judgeTimeout)
				v = j.run(ctx, abs, dir, pages)
				cancel()
				// Both runs are poppler's time; the record used to say 0.
				v.Ms = time.Since(t0).Milliseconds()
			}
			if fr.SampledPages == nil {
				fr.SampledPages = pages
			}
			fmt.Fprintf(errOut, "  %-8s ok=%v pages=%d text=%d warn=%d %dms %s\n",
				v.Judge, v.OK, v.Pages, v.TextChar, v.Warnings, v.Ms, v.Err)
			fr.Verdicts = append(fr.Verdicts, v)
		}
		score(&fr)
		for _, v := range fr.Verdicts {
			if len(v.Diffs) > 0 {
				fmt.Fprintf(errOut, "  %-8s worst Δ%.1f%% on p%d\n", v.Judge, v.WorstPct, v.WorstPage)
			}
		}
		fmt.Fprintf(errOut, "  consensus max %.1f%% on p%d (pages %v)\n", fr.ConsensusMax, fr.ConsensusPg, fr.SampledPages)
		results = append(results, fr)
	}

	// A run whose record could not be written has not been made: the table is
	// read by people and the JSON by the next run, and losing either silently
	// is a gap nobody can tell from a run that never happened.
	if err := writeResults(*resultsPath, results); err != nil {
		fmt.Fprintf(errOut, "judges: results: %v\n", err)
		return 1
	}
	if err := writeReport(*reportPath, results, judges); err != nil {
		fmt.Fprintf(errOut, "judges: report: %v\n", err)
		return 1
	}
	fmt.Fprintf(out, "%d PDFs judged; report %s; results %s\n", len(results), *reportPath, *resultsPath)
	return 0
}
