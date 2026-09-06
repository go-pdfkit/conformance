// judges runs every reference PDF reader present on this machine over a set
// of PDFs and reports whether each one opens, paginates, renders and extracts
// text the same way — conformance per channel. A file that is the smallest
// is worth nothing if one reader in the field disagrees with the others about
// it.
//
//	judges -pdfs 'out/*.pdf,out/bench/*.pdf' -pdfium /path/to/pdfium_test
//
// Judges, each skipped with a note when its binary is absent, never faked:
//
//	qpdf     structural check (qpdf --check)          — exit 0 clean, 3 warnings, 2 errors
//	poppler  pdfinfo / pdftoppm / pdftotext             — the reference the per-judge Δ is taken against
//	mupdf    mutool draw (png + txt), mutool info
//	gs       Ghostscript png16m + txtwrite
//	pdfium   pdfium_test --png --txt (Chrome's engine; PDFIUM_TEST=/path or -pdfium)
//	pdfjs    pdf.js under node (judges/pdfjs-*.mjs)     — Firefox's engine
//	quartz   sips (macOS ImageIO/Quartz, Preview's engine) — page 1 render only
//
// For each PDF and judge it records: pages reported, extracted-text length
// as a ratio of poppler's, and — on a sample of pages: the first, the middle
// and the last — how far its render is from poppler's at 96 dpi (share of
// pixels differing noticeably after both are box-downsampled to the same
// width), worst page reported. Because poppler is itself just one reader, a
// per-page consensus is computed too: the mean pairwise distance between all
// judges' renders of that page, so no single reader is privileged.
//
// A producer should judge a control beside its own output — html2pdf judges
// Chrome's PDFs of the same pages — because a judge that disagrees on the
// control too is judge noise, not a defect of ours.
package main

import "os"

// osExit is a variable so the tests can reach the exit path without ending the
// test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }
