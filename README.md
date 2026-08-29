# conformance

[![CI](https://github.com/go-pdfkit/conformance/actions/workflows/ci.yml/badge.svg)](https://github.com/go-pdfkit/conformance/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-pdfkit/conformance.svg)](https://pkg.go.dev/github.com/go-pdfkit/conformance)
[![License: BSD-3-Clause](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](LICENSE)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](#how-it-is-checked)

Where [`go-pdfkit`](https://github.com/go-pdfkit) is judged by implementations
that are not its own, over corpora of real PDFs.

A file our own reader reads back perfectly can draw nothing anywhere else. That
has happened here — a rebuilt cross-reference stream that our reader was
delighted with and macOS drew as a blank page — which is why the tools in this
repository ask poppler rather than asking ourselves.

## A corpus is a measurement, not a directory

What they need in order to ask is a population of documents, and a figure is
only as honest as the population it was drawn from. Government forms and arXiv
figures between them hold almost no JBIG2 and almost no JPEG 2000, so
"six files in sixteen hundred use JBIG2" says what is in *those two*
populations — not what is in the world. Mass digitisation is where a scanned
page lives, and a scanned page is a fax.

So a corpus here is a directory of documents **and** a `MANIFEST.tsv` beside
them recording, for each one, the population it belongs to, the URL it came
from, when, how big it is and what its bytes hash to. That is what makes a
number reproducible and lets a document that changed underneath be noticed.

```
harvest -dir /Users/Shared/pdfscans -origin ia-americana \
        -query 'collection:americana AND format:"Text PDF"' -want 250
```

`harvest` is resumable by construction: what is already in the manifest is
skipped, so an interrupted run is continued by running it again and a corpus is
extended by asking for a larger `-want`.

It records the population per document because **a prevalence is per population
or it is not a prevalence**.

## A page hides its parts

`compare` draws a page twice and says how far apart the two pictures are, which
is the question a reader asks. But a page is a **composition**, and a
composition averages: a picture that is wholly wrong moves a page's number by a
few percent and passes. That is not a worry, it is a history —
`go-pdfkit/render` v0.12.0 shipped drawing scanned pages dark because what was
measured was that ink appeared and not that the *right* ink appeared.

So `images` takes the pictures out instead, one by one, and reports by the
filter each was stored in — because *"our CCITT is right and our JBIG2 is
wrong"* is the finding a per-page number cannot produce.

```
images -dir /Users/Shared/pdfscans -only ia-medical -pages 2
```

Two differences from `compare` are deliberate. It asks **`pdfimages`, not
`pdftoppm`**: extracting puts no rasteriser between the codec and the answer,
and poppler's own resampling otherwise shows up as every decoder disagreeing
with it slightly. And **exact equality is the question** — two rasterisers
disagree on every edge pixel, but two implementations of one image format
either agree bit for bit or one of them is wrong.

Measured that way, four JBIG2 decoders over 403 real streams: one was exact on
every stream it read, and one decoded *more* streams and was wrong on 91% of
them, with nothing in its output saying so.

## How it is checked

Exact 100% statement coverage including every error branch, `go vet`, `-race`,
and nine cross-compile targets. Nothing outside the standard library.
