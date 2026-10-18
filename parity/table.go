// Copyright (c) the go-pdfkit authors.
// SPDX-License-Identifier: BSD-3-Clause

package parity

// BentoPDF maps BentoPDF's own tool names onto what this fleet can be SHOWN to
// do. Every entry is checked; absence is the default.
//
// Measured against BentoPDF v2.8.8 at commit 6ce0d69 (its own last commit
// 2026-10-17, read 2026-10-18): 114 tools.
//
// ⛔ The commit is part of the figure. This line named 3a5f146, and that
// commit is not reachable in the clone the measurement actually reads — so
// the denominator was attributed to something nobody here can check. The
// count happens to be the same 114 at both readings, which is why it went
// unnoticed and why it is worth writing down: it would have gone on agreeing
// until the day BentoPDF added a tool.
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
	// ⛔ sign-pdf is a PICTURE, and a note of mine said otherwise. BentoPDF's
	// page offers drawing a signature, typing one, or uploading an image of
	// one, and then places it: there is no certificate anywhere in it. I had
	// recorded the three signature tools as one cryptographic job, which would
	// have kept this tool uncovered behind work it does not need —
	// digital-sign-pdf and validate-signature-pdf are the cryptographic pair,
	// and they are still uncovered.
	{Tool: "sign-pdf", Claim: "verb:place"},

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
	// ⛔ Both of these are the SAME detector, and it is an inference: a page
	// holds text put at places, and a table is what a reader sees when those
	// places line up. BentoPDF's own two pages are the same thing twice as
	// well — extract-tables writes one file per table, pdf-to-csv writes all
	// of them into one — and `pdfops tables` does both, with -to and without.
	//
	// What it does NOT do is read ruled lines, which is the strategy
	// PyMuPDF's find_tables uses by DEFAULT and falls back from. So a table
	// held apart by rules rather than by whitespace is not found, and the
	// claim is for the text strategy alone. Said in extract's README under
	// "What it still gets wrong", and in ops's beside the verb.
	{Tool: "extract-tables", Claim: "verb:tables"},
	{Tool: "pdf-to-csv", Claim: "verb:tables"},
	// ⛔ One verb for both of these, because they are the same thing twice
	// over: BentoPDF's prepare-pdf-for-ai is per-page MARKDOWN with the
	// document's metadata beside it — PyMuPDF4LLM's LlamaMarkdownReader —
	// which is `pdfops markdown -json`, and pdf-to-markdown is the same
	// reading written out as one document.
	//
	// What is claimed is the inference, not a conversion: a PDF has no
	// headings, and go-pdfkit/document says in its README what it will and
	// will not see. Ruled lines and multi-column layouts are NOT read, and
	// that bound is published rather than buried.
	{Tool: "pdf-to-markdown", Claim: "verb:markdown"},
	{Tool: "prepare-pdf-for-ai", Claim: "verb:markdown"},
	// ⛔ The same reading as markdown above, written by a different writer —
	// go-ooxml/ooxml, which is the fleet's first OOXML WRITER and sits on
	// opc.Build. So what is claimed is again the inference and not a
	// conversion: what comes out is the DOCUMENT, not the pages. The fonts,
	// the margins and the page breaks are the typesetter's work, and a .docx
	// reproducing them would be a picture of a PDF rather than something
	// anybody can edit.
	//
	// excel writes the tables only, one sheet each: a spreadsheet is a grid
	// of cells and a document is not.
	//
	// ⛔ And the judge is not our own reader. Where LibreOffice is installed
	// the writer's own tests ask it, because a package go-ooxml is delighted
	// with can be one Word refuses — and that is a failure nothing in this
	// repository could see.
	{Tool: "pdf-to-word", Claim: "verb:word"},
	{Tool: "pdf-to-excel", Claim: "verb:excel"},
	// ⛔ pdf-to-json is NOT the text. BentoPDF's own page says so plainly —
	// "the raw PDF object structure, not extracted text content" — and
	// points anyone wanting the text at prepare-pdf-for-ai instead. So the
	// claim is the object graph, which `pdfops objects` writes: the trailer
	// and everything reachable from it, a stream's dictionary and length
	// without its bytes.
	//
	// Reachable from the trailer, and not every object in the file: an
	// object no reference points at is in the bytes and in nobody's
	// document, and listing it beside the rest would say the two are alike.
	{Tool: "pdf-to-json", Claim: "verb:objects"},
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

	// ⛔ FictionBook, and the claim names Parse rather than the repository for
	// the reason every other converter claim does. It is the one ebook format
	// an `encoding/xml` reader can do on its own: a single XML file, no ZIP, no
	// package document, no spine, its images base64 inside the same file.
	{Tool: "fb2-to-pdf", Claim: "go-richdoc/fb2#func Parse"},

	// --- what the HTML reader unlocked ------------------------------------
	// ⛔ These three all needed the SAME missing piece, which is why the gap
	// was worth naming rather than working around: EPUB's content documents
	// are XHTML, MOBI's text is HTML, and the body of nearly every HTML email
	// is HTML. go-richdoc/html v0.1.0 reads it, delegating the parse to
	// golang.org/x/net/html, and two of the three followed the same afternoon.
	//
	// The claim for epub names readingOrder and not Parse, deliberately: the
	// hard part of an EPUB is not opening the ZIP, it is that the SPINE is the
	// only thing that says what order to read the chapters in. A claim on
	// Parse alone would go on saying yes after a refactor dropped the spine
	// and left an alphabetical walk of the archive — which is the defect every
	// naive reader of a container format has.
	{Tool: "epub-to-pdf", Claim: "go-richdoc/epub#func (p *opf) readingOrder"},

	// And email names the multipart walker for the same reason: an
	// alternative is the same message twice, and a reader that lost that
	// distinction would print every HTML mail once as text and once as itself.
	{Tool: "email-to-pdf", Claim: "go-richdoc/email#multipart/alternative"},

	// ⛔ And mobi names the TRAILING-ENTRY stripper, which is the one thing in
	// that format that fails silently. The bytes at the end of a text record
	// are not text and are not compressed — indexing data, and the overlap of
	// a multibyte character split across two records — and feeding them to the
	// decompressor does not error: it produces plausible rubbish at the end of
	// every record. A claim on Parse would go on saying yes after that was
	// dropped, and the book would still open.
	//
	// What this entry said until now — "the container and the two
	// decompressors are not there" — was true of the PalmDOC half and
	// deliberately remains true of HUFF/CDIC and KF8: go-richdoc/mobi refuses
	// both BY NAME. Measured against Project Gutenberg's own Pride and
	// Prejudice with Gutenberg's plain text of the same book as the judge:
	// 125 816 words against 126 630, and 3 638 of 4 215 long sentences found
	// verbatim.
	{Tool: "mobi-to-pdf", Claim: "go-richdoc/mobi#func stripTrailing"},

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
	// ⛔ The claim names the IMPORT, like the five above it, and that is the
	// whole wiring: convert decodes through image.Decode, so a codec that
	// registers itself is read without convert knowing its name. Remove the
	// import and the claim goes with it — which is the point, because the
	// capability leaves at the same moment.
	//
	// What go-images/psd reads is the flattened COMPOSITE, which is what
	// BentoPDF's own page describes ("renders the flattened composite image")
	// and what Photoshop itself shows. Measured against sips (ImageIO, and
	// nothing to do with our code): exact on 2 073 600 pixels of a real file.
	{Tool: "psd-to-pdf", Claim: "go-pdfkit/convert#go-images/psd"},
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
