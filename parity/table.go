// Copyright (c) the go-pdfkit authors.
// SPDX-License-Identifier: BSD-3-Clause

package parity

// BentoPDF maps BentoPDF's own tool names onto what this fleet can be SHOWN to
// do. Every entry is checked; absence is the default.
//
// Measured against BentoPDF v2.8.8 (commit 3a5f146, 2026-10-03): 114 tools.
var BentoPDF = []Claim{
	// --- the page verbs, all read off the pdfops binary --------------------
	{Tool: "merge-pdf", Claim: "verb:merge"},
	{Tool: "split-pdf", Claim: "verb:split"},
	{Tool: "extract-pages", Claim: "verb:select"},
	{Tool: "delete-pages", Claim: "verb:delete"},
	{Tool: "reverse-pages", Claim: "verb:reverse"},
	{Tool: "rotate-pdf", Claim: "verb:rotate"},
	{Tool: "rotate-custom", Claim: "verb:rotate"},
	{Tool: "crop-pdf", Claim: "verb:crop"},
	{Tool: "fix-page-size", Claim: "verb:resize"},
	{Tool: "organize-pdf", Claim: "verb:move"},
	{Tool: "n-up-pdf", Claim: "verb:nup"},
	{Tool: "pdf-booklet", Claim: "verb:booklet"},
	{Tool: "alternate-merge", Claim: "verb:interleave"},
	{Tool: "combine-single-page", Claim: "verb:onepage"},
	{Tool: "posterize-pdf", Claim: "verb:poster"},
	{Tool: "add-blank-page", Claim: "verb:blank"},
	{Tool: "page-dimensions", Claim: "verb:info"},

	// --- marks on the page -------------------------------------------------
	{Tool: "add-watermark", Claim: "verb:watermark"},
	{Tool: "add-stamps", Claim: "verb:stamp"},
	{Tool: "page-numbers", Claim: "verb:number"},
	{Tool: "add-page-labels", Claim: "verb:labels"},
	{Tool: "pdf-layers", Claim: "verb:layers"},
	// ⛔ text-color recolours the text in each page's OWN content stream.
	// BentoPDF's page says the tool "identifies text regions on each page and
	// recolors them"; text drawn through a Form XObject is not reached here,
	// because an XObject is SHARED and recolouring one would change pages the
	// caller did not name. The verb COUNTS what it could not reach and says
	// so, which is why this is a claim rather than a refusal: the one failure
	// that would make it dishonest — a document coming back unchanged while
	// looking converted — is the one the tool reports out loud.
	{Tool: "text-color", Claim: "verb:textcolour"},
	{Tool: "bates-numbering", Claim: "verb:bates"},

	// --- the file itself ---------------------------------------------------
	{Tool: "edit-metadata", Claim: "verb:metadata"},
	{Tool: "view-metadata", Claim: "verb:metadata"},
	{Tool: "remove-metadata", Claim: "verb:strip"},
	{Tool: "sanitize-pdf", Claim: "verb:sanitize"},
	{Tool: "compress-pdf", Claim: "verb:compress"},
	{Tool: "flatten-pdf", Claim: "verb:flatten"},
	{Tool: "bookmark", Claim: "verb:outline"},
	{Tool: "table-of-contents", Claim: "verb:outline"},

	// --- locks -------------------------------------------------------------
	{Tool: "protect-pdf", Claim: "verb:encrypt"},
	{Tool: "unlock-pdf", Claim: "verb:decrypt"},
	{Tool: "change-permissions", Claim: "verb:permissions"},
	{Tool: "remove-restrictions", Claim: "verb:permissions"},

	// --- what comes OUT of a PDF -------------------------------------------
	{Tool: "pdf-to-text", Claim: "verb:text"},
	{Tool: "extract-images", Claim: "verb:images"},
	{Tool: "extract-attachments", Claim: "verb:detach"},
	{Tool: "add-attachments", Claim: "verb:attach"},
	{Tool: "edit-attachments", Claim: "verb:attachments"},

	// --- forms -------------------------------------------------------------
	{Tool: "form-filler", Claim: "verb:fill"},

	// --- rasterising, and what can be done once a page is pixels ------------
	// ⛔ Every one of these RASTERISES: what comes out has no text in it. That
	// is what BentoPDF's own tools do for the same jobs, so the claims are
	// honest — but it is why rotate, crop and stamp are NOT here. They keep
	// the text, and they are verbs of pdfops above.
	{Tool: "rasterize-pdf", Claim: "go-pdfkit/render#func Page"},
	{Tool: "invert-colors", Claim: "go-pdfkit/convert#images.Invert"},
	{Tool: "pdf-to-greyscale", Claim: "go-pdfkit/convert#images.Grayscale"},
	{Tool: "adjust-colors", Claim: "go-pdfkit/convert#images.AdjustContrast"},
	{Tool: "background-color", Claim: "go-pdfkit/convert#Background: bg"},
	{Tool: "scanner-effect", Claim: "go-pdfkit/convert#type ScannerEffect"},

	// --- the browser workbench ---------------------------------------------
	{Tool: "edit-pdf", Claim: "go-pdfkit/app#func main"},
	{Tool: "pdf-multi-tool", Claim: "go-pdfkit/app#func main"},

	// --- archives, and the one tool that converts nothing -------------------
	// ⛔ pdf-to-zip is NOT "a PDF's pages in a zip". BentoPDF's own page says
	// "bundle multiple PDF FILES into a single ZIP archive — no conversion",
	// which is a different tool entirely; reading the fiche is what stopped a
	// claim that would have been plainly false, and a verifier that only
	// checks a symbol exists could never have caught it.
	{Tool: "cbz-to-pdf", Claim: "go-pdfkit/convert#func ArchiveToPDF"},
	{Tool: "pdf-to-cbz", Claim: "go-pdfkit/convert#func writeComicInfo"},
	{Tool: "pdf-to-zip", Claim: "go-pdfkit/convert#func Bundle"},
	{Tool: "svg-to-pdf", Claim: "go-pdfkit/convert#func RasterizeSVG"},

	// --- document converters, each through richdoc then a PDF writer -------
	{Tool: "odt-to-pdf", Claim: "go-odf/odf#func Parse"},
	{Tool: "ods-to-pdf", Claim: "go-odf/odf#office:spreadsheet"},
	{Tool: "odp-to-pdf", Claim: "go-odf/odf#office:presentation"},

	// ⛔ odg-to-pdf is NOT claimed, and the reason is the DIFFERENCE between
	// the two formats rather than a gap in the reader: the same code reads
	// both. A presentation IS largely its words, so a deck whose text comes
	// through under a heading per slide is the document. A drawing IS its
	// geometry — BentoPDF's own page says "diagrams, flowcharts, and vector
	// illustrations" — and a PDF holding a flowchart's labels with no boxes or
	// arrows is not that drawing. Absence is the conservative direction, and
	// this must never make the fleet look better than it is.
	{Tool: "rtf-to-pdf", Claim: "go-rtf/rtf#func Parse"},

	// --- Word, through go-ooxml -------------------------------------------
	// ⛔ BentoPDF's own page says this tool takes "DOC, DOCX, ODT, and RTF".
	// Three of the four are read here — DOCX by go-ooxml/ooxml, ODT by
	// go-odf/odf, RTF by go-rtf/rtf — and the fourth, DOC, is NOT: the
	// pre-2007 binary format is an OLE compound file and nothing in the fleet
	// reads one. The gap is named rather than passed over, the way cbz-to-pdf
	// names CBR, because a claim whose shortfall is written down can be
	// closed by whoever reads it.
	//
	// The claim names the WordprocessingML reader rather than the repository:
	// a "go-ooxml/ooxml exists" claim would go on saying yes after the Word
	// reader had been dropped in favour of the other two.
	{Tool: "word-to-pdf", Claim: "go-ooxml/ooxml#func ParseWord"},
	{Tool: "excel-to-pdf", Claim: "go-ooxml/ooxml#func ParseWorkbook"},
	{Tool: "powerpoint-to-pdf", Claim: "go-ooxml/ooxml#func ParsePresentation"},

	// ⛔ THE SAME SHORTFALL IN ALL THREE, named rather than passed over: each
	// of BentoPDF's three Office tools takes the pre-2007 BINARY format as
	// well, and nothing in the fleet reads one. They are OLE compound files —
	// a different container entirely from OPC, with its own FAT and its own
	// record streams — so this is a gap of one missing library and not of
	// three missing readers.
	//
	//	word-to-pdf        DOC  (and DOCX ✓, ODT ✓, RTF ✓)
	//	excel-to-pdf       XLS  (and XLSX ✓, ODS ✓, CSV ✓)
	//	powerpoint-to-pdf  PPT  (and PPTX ✓, ODP ✓)
	//
	// Writing it down the way cbz-to-pdf names CBR: a claim whose shortfall is
	// written down can be closed by whoever reads it, and the three share one.
	{Tool: "markdown-to-pdf", Claim: "go-richdoc/markdown#func Parse"},

	// --- the data formats, through go-richdoc/data -------------------------
	// ⛔ Each claim names that format's own DECISION — the function that does
	// the thing the format needs — rather than the repository. A single
	// "go-richdoc/data exists" claim would go on saying yes after a reader had
	// been dropped.
	{Tool: "csv-to-pdf", Claim: "go-richdoc/data#func Sniff"},
	{Tool: "json-to-pdf", Claim: "go-richdoc/data#func uniformObjects"},
	{Tool: "xml-to-pdf", Claim: "go-richdoc/data#uniformKids"},
	{Tool: "txt-to-pdf", Claim: "go-richdoc/data#func laidOut"},

	// --- pictures both ways, through go-pdfkit/convert ---------------------
	// ⛔ Each claim names the FORMAT, so an entry survives only while that
	// format is still named in the source. A single "convert exists" claim
	// would go on saying yes after a codec had been dropped.
	{Tool: "image-to-pdf", Claim: "go-pdfkit/convert#func ToPDF"},
	{Tool: "png-to-pdf", Claim: "go-pdfkit/convert#image/png"},
	{Tool: "jpg-to-pdf", Claim: "go-pdfkit/convert#image/jpeg"},
	{Tool: "bmp-to-pdf", Claim: "go-pdfkit/convert#x/image/bmp"},
	{Tool: "tiff-to-pdf", Claim: "go-pdfkit/convert#x/image/tiff"},
	{Tool: "webp-to-pdf", Claim: "go-pdfkit/convert#x/image/webp"},
	{Tool: "pdf-to-png", Claim: `go-pdfkit/convert#"png"`},
	{Tool: "pdf-to-jpg", Claim: `go-pdfkit/convert#"jpeg"`},
	{Tool: "pdf-to-bmp", Claim: `go-pdfkit/convert#"bmp"`},
	{Tool: "pdf-to-tiff", Claim: `go-pdfkit/convert#"tiff"`},

	// --- two more that go-pdfkit/ops already had under another name --------
	{Tool: "remove-annotations", Claim: "go-pdfkit/ops#func (d *Doc) RemoveAnnotations("},
	// ⛔ The claim names the SUBSTITUTION, not the variable. A plain "{pages}"
	// is in a doc comment two lines above, and a claim satisfied by prose is
	// exactly what this table must not contain: a README saying a function
	// exists is a claim, not the function. Stamp has all nine positions, which
	// covers the six this tool offers, and {page}/{pages} are what make
	// "Page 3 of 10" update per page.
	{Tool: "header-footer", Claim: `go-pdfkit/ops#"{pages}", strconv.Itoa(pages)`},

	// duplex-collate was in the list below as "the library can express it but
	// no single entry point does the job". Writing the refusal down is what
	// got it closed: go-pdfkit/ops v0.12.0 has DuplexCollate and a `duplex`
	// verb, so the claim asks the binary.
	{Tool: "duplex-collate", Claim: "verb:duplex"},

	// ⛔⛔ THREE REFUSALS BECAME VERBS. Each of these was written down here as
	// "not covered", with the reason — and each time the reason turned out to
	// be the SPECIFICATION. Writing down why a thing cannot be claimed is what
	// got all three built, which is an argument for the practice and not just
	// for the entries:
	//
	//   - divide-pages said "Poster MAGNIFIES; divide-pages cuts a page into
	//     halves that stay half-size". go-pdfkit/ops v0.13.0 has Divide.
	//   - remove-blank-pages said "it needs a blank DETECTOR over rendered
	//     pixels, and a detector is a measurement, not a wiring job". It is
	//     now one, and the measurement moved the default resolution from 50
	//     DPI to 150: at 50 an eight-point full stop covers exactly ONE pixel,
	//     which is the mark a speck of dust makes.
	//   - duplex-collate said "no single entry point does the job". It has one.
	{Tool: "divide-pages", Claim: "verb:divide"},
	{Tool: "remove-blank-pages", Claim: "verb:deblank"},

	// repair-pdf is reader's repair, given a name. ⛔ The claim is the VERB
	// and not reader's symbol, because the repair itself was always there and
	// silent: every ops verb has been repairing files for as long as reader
	// has, and a claim on the symbol would have been true and useless. What
	// this table can now show is a tool that SAYS so and keeps the result.
	{Tool: "repair-pdf", Claim: "verb:repair"},

	// ⛔ ONE that looks close and is not, kept with its reason so that "not
	// covered" does not invite the next person to re-derive it:
	//
	//   - compare-pdfs. conformance/compare already renders two pages, reduces
	//     both to grey, blurs them and counts the pixels that differ — which
	//     is the measurement this tool needs. But it compares OUR drawing
	//     against poppler's, to judge us; it needs poppler installed, and it
	//     pairs page i with page i rather than "intelligently". A claim on it
	//     would be a claim that we can diff two files a caller brings, and we
	//     cannot. Generalising it is real work, not a table entry.
}
