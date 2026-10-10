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
	{Tool: "fix-page-size", Claim: "go-pdfkit/ops#func (d *Doc) Resize("},
	{Tool: "organize-pdf", Claim: "go-pdfkit/ops#func (d *Doc) Move("},
	{Tool: "n-up-pdf", Claim: "verb:nup"},
	{Tool: "pdf-booklet", Claim: "verb:booklet"},
	{Tool: "alternate-merge", Claim: "go-pdfkit/ops#func (d *Doc) Interleave("},
	{Tool: "combine-single-page", Claim: "go-pdfkit/ops#func (d *Doc) OnePage("},
	{Tool: "posterize-pdf", Claim: "go-pdfkit/ops#func (d *Doc) Poster("},
	{Tool: "add-blank-page", Claim: "verb:blank"},
	{Tool: "page-dimensions", Claim: "verb:info"},

	// --- marks on the page -------------------------------------------------
	{Tool: "add-watermark", Claim: "verb:watermark"},
	{Tool: "add-stamps", Claim: "verb:stamp"},
	{Tool: "page-numbers", Claim: "verb:number"},
	{Tool: "bates-numbering", Claim: "verb:bates"},

	// --- the file itself ---------------------------------------------------
	{Tool: "edit-metadata", Claim: "go-pdfkit/ops#func (d *Doc) SetInfo("},
	{Tool: "view-metadata", Claim: "go-pdfkit/ops#func (d *Doc) Info("},
	{Tool: "remove-metadata", Claim: "verb:strip"},
	{Tool: "sanitize-pdf", Claim: "verb:sanitize"},
	{Tool: "compress-pdf", Claim: "verb:compress"},
	{Tool: "flatten-pdf", Claim: "verb:flatten"},
	{Tool: "bookmark", Claim: "go-pdfkit/ops#func (d *Doc) SetOutline("},
	{Tool: "table-of-contents", Claim: "go-pdfkit/ops#func (d *Doc) SetOutline("},

	// --- locks -------------------------------------------------------------
	{Tool: "protect-pdf", Claim: "verb:encrypt"},
	{Tool: "unlock-pdf", Claim: "verb:decrypt"},
	{Tool: "change-permissions", Claim: "verb:permissions"},
	{Tool: "remove-restrictions", Claim: "verb:permissions"},

	// --- what comes OUT of a PDF -------------------------------------------
	{Tool: "pdf-to-text", Claim: "verb:text"},
	{Tool: "extract-images", Claim: "verb:images"},
	{Tool: "extract-attachments", Claim: "go-pdfkit/ops#func (d *Doc) Detach("},
	{Tool: "add-attachments", Claim: "go-pdfkit/ops#func (d *Doc) Attach("},
	{Tool: "edit-attachments", Claim: "go-pdfkit/ops#func (d *Doc) Attachments("},

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
	{Tool: "rtf-to-pdf", Claim: "go-rtf/rtf#func Parse"},
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
}
