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

	// --- rasterising --------------------------------------------------------
	{Tool: "rasterize-pdf", Claim: "go-pdfkit/render#func Page"},

	// --- the browser workbench ---------------------------------------------
	{Tool: "edit-pdf", Claim: "go-pdfkit/app#func main"},
	{Tool: "pdf-multi-tool", Claim: "go-pdfkit/app#func main"},

	// --- document converters, each through richdoc then a PDF writer -------
	{Tool: "odt-to-pdf", Claim: "go-odf/odf#func Parse"},
	{Tool: "ods-to-pdf", Claim: "go-odf/odf#office:spreadsheet"},
	{Tool: "rtf-to-pdf", Claim: "go-rtf/rtf#func Parse"},
	{Tool: "markdown-to-pdf", Claim: "go-richdoc/markdown#func Parse"},

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
