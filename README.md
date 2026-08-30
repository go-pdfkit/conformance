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

What the two sides do **not** share is counted apart rather than as
disagreement. An **exact complement** is one of those: `pdfimages` writes a
**stencil** — an image with `/ImageMask true`, or a stream a picture names as
its `/Mask` — with the opposite polarity to the samples it holds. Ours is the
right reading: on a 644-byte document whose mask is half painted and half not,
poppler's own `pdftoppm` paints the half the spec says to paint, `pdfimages`
writes the other one, and ours agrees with the rendering. "Identical up to
inversion" and "wrong" look the same in a count of differing pixels, and only
one of them needs looking into.

The convention is about being a stencil and nothing else. It is **not** about
JBIG2: of 28 such masks in the `us-opm` population, 27 arrived in
`/CCITTFaxDecode` and one in `/FlateDecode`, and the minimal document carries
no filter at all. And it is **not** about soft masks, which are the one kind
`pdfimages` leaves alone — 0 of 734 `/SMask` streams in `fr-cerfa` and
`us-opm` came out inverted, against 13 of 13 `/ImageMask` and 2 of 2 `/Mask`.

A **remapped** picture is the other: `/Decode` maps the stored samples onto the
range the colour space wants; a viewer applies it and `pdfimages` writes the
samples as stored, so on a one-bit mask a `/Decode` of `[1 0]` inverts every
pixel. Of the 248 JBIG2 masks on the first pages of the medical population, 22
carry one, and all 22 are soft masks.

A filter with nothing comparable left is not reported as the worst thing in the
corpus: nothing to compare is not evidence of being wrong.

## What it comes to today

A number that is not written down cannot be regressed against. `baseline/`
holds a whole run of `images` over both corpora — the counts per population
per filter, and beside them the corpus, the poppler that judged it, every
module version it was built against and when it was taken, because a figure
that drops between two runs means a regression only if everything else held.

```
images -dir /Users/Shared/pdfscans -only ia-medical -json
```

[`baseline/README.md`](baseline/README.md) reads it out.

### Those numbers were taken at `render` v0.19.0, and this is v0.20.0

Every record under `baseline/` carries the versions it was built against in its
own `modules` block, and every one of them says **`render` v0.19.0**. This
repository is now built against **v0.20.0**, and a run taken here is across a
seam from them.

The seam is `render.Images`. Under v0.19.0 it answered per **draw**: a page
that stamped one logo five hundred times yielded five hundred entries decoded
from the same bytes, against `pdfimages`'s one, so this repository kept one
entry per resource name to compensate. v0.20.0 removed the cause — a page's
resources were being walked as a tree when they are a graph, so a picture
reached through two forms was decoded once per path — and each picture now
comes back once however many forms reach it.

**So the compensation is gone, and it had to go.** A resource name is unique
within one dictionary and not across them. Under v0.20.0, over the whole
`/Users/Shared/pdfforms` corpus, **40 of its 2268 documents draw two different
pictures sharing a name on their first page, and collapsing on the name would
have dropped 64 of them** — 28 in `gh-qpdf`, where qpdf's form-XObject fixtures
give two forms an `Im1` each, and 36 in `fr-impots`, where one issuer's real
forms do the same with `Im0`. The second half of that is the part that matters:
this is not only a property of a test corpus.

A later reader comparing a fresh run against `baseline/` without noticing the
seam would be reading a change in what is counted as a change in what is right.

## How it is checked

Exact 100% statement coverage including every error branch, `go vet`, `-race`,
and nine cross-compile targets. Nothing outside the standard library.
