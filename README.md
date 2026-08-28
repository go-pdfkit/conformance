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

## How it is checked

Exact 100% statement coverage including every error branch, `go vet`, `-race`,
and nine cross-compile targets. Nothing outside the standard library.
