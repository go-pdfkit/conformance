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

One difference from `compare` is deliberate. It asks **`pdfimages`, not
`pdftoppm`**: extracting puts no rasteriser between the codec and the answer,
and poppler's own resampling otherwise shows up as every decoder disagreeing
with it slightly.

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

## Bit-exact where the format promises it, a tolerance where it does not

**The threshold is not the same for every filter, because the formats do not
promise the same thing.**

`CCITTFaxDecode`, `JBIG2Decode` and raw samples are **lossless**. The bytes
determine the pixels, so two implementations of one of those either agree bit
for bit or one of them is wrong. **They are judged by exact equality**, and the
corpus reads 100% for both JBIG2 columns, which is what a lossless format looks
like when it is right.

`DCTDecode` and `JPXDecode` are not. **Neither JPEG nor JPEG 2000 requires a
bit-exact decoder**: the standards fix the inverse transform to a tolerance and
not to a value, so two conformant implementations may legitimately differ in
the last bits of every pixel and neither of them is wrong. Asking those two for
bit-exactness measures conformance *to poppler* rather than conformance to the
format, and it made a correct decoder look like a broken one — the baseline
reads 15.4% for `JPXDecode` and 33.2% for `DCTDecode` behind medians of 0.0020
and 0.000135 — one pixel in five hundred, and one in seven thousand.

> **A `DCTDecode` or `JPXDecode` picture agrees when at most 1% of its pixels
> differ. Every other filter must be exact.**

Raw samples read 84.9%, which is **not** an argument for widening their
threshold too. What disagrees there is colour: the comparison asks only whether
a pixel is ink, so two implementations doing different ICC conversion can cross
that one bisection on a quarter of the pixels while both are right — `fr-cerfa`
differs on 535 `(samples)` pictures at a median of 0.286, which no tolerance
narrow enough to be worth having would absorb. Comparing per channel instead of
bisecting is the fix for that one, not a wider band, and finding 4 of the
baseline says so.

### Where 1% comes from

The number is read off the landed measurement, not chosen for looking tidy.
`baseline/` records, for every population and every filter, the median and the
worst of the shares that differed. Eighteen of those rows are a lossy filter
with something to say, and sorted by their median they fall in two groups with
nothing in between:

| | population medians of the differing shares |
|---|---|
| **rounding** | 0.000011, 0.000013, 0.000017, 0.000086, 0.000095, 0.000135, 0.000178, 0.000228, 0.000248, 0.000296, 0.000619, 0.000898, 0.000909, **0.001966** |
| *(empty)* | |
| **not rounding** | **0.047161**, 0.159758, 0.188452, 0.898821 |

The band between 0.001966 (`ia-medical` JPX) and 0.047161 (`gh-pypdf` DCT) is
empty and spans a factor of 24. **A cut anywhere inside it separates the same
fourteen populations from the same four**, so nothing in this table decides
where inside it to cut, and the middle is where a number with no other reason
belongs. The geometric centre of the band is 0.0096, and 1% is the round number
4% away from it: **5.1× above the widest median that is rounding, 4.7× below
the narrowest that is not.** Two further readings below move the floor of the
band up and then argue against spending the room that frees.

There is no arithmetic that derives the number instead. `difference` asks only
whether a pixel is ink — luminance under 128 — so a decoder's rounding flips
the verdict for exactly those pixels whose true luminance sits within the
rounding of the threshold, and how many of those a picture has is a fact about
the picture. A flat mid-grey field could disagree everywhere from rounding
alone. **The tolerance is therefore empirical, and it is only as good as the
populations it was read off.**

### There is no confirmed decode defect here to calibrate the narrow end against

It would be tidy to say the tolerance is set narrow enough to keep catching a
known bad picture. **No such picture is known in this corpus, and the number
was not chosen against one.**

The two concentrations that looked like candidates are not.
[conformance#13](https://github.com/go-pdfkit/conformance/issues/13) chased
both to a cause and neither is a decoder:

- Of the **173 `(samples)` complements in `fr-cerfa`**, 144 are **this
  instrument's own matcher**. Those pages set coloured text by stretching a
  uniform 2×2 swatch under a large `/SMask`, `match` pairs a picture with the
  first unclaimed picture of the same size, and a black swatch of ours meets a
  white swatch of theirs. The 29 that survive identity matching are `pdfimages`
  writing a one-bit `/Indexed` picture black only when the palette entry is
  exactly `#000000`.
- The **28 in `us-opm`** are the stencil polarity convention, and `pdftoppm` —
  poppler's own renderer — agrees with `go-pdfkit` against `pdfimages`.

They are on a lossless filter, so the tolerance never reaches them; an exact
complement is a share of 1.0, counted in its own column before any threshold is
applied. But that is now **bookkeeping rather than fidelity**, and calling them
defects the tolerance must keep catching would be claiming a signal that was
measured away.

What is left above the cut is a set of **unexplained disagreements**, which is
not the same as defects. Taking the worst picture of each population that has
one: `gh-pdfbox` JPX 0.899, `fr-impots` DCT 0.880, `ia-medical` JPX 0.371,
`gh-qpdf` DCT 0.195, `gh-pypdf` DCT 0.047, `fr-cerfa` DCT 0.095. Nobody has
established that any of them is ours to fix. **The narrow end of the tolerance
is therefore set by the shape of the distribution and by nothing else** — by
where a population's disagreements stop looking like rounding, not by where a
known error begins.

**And one of those six was audited and moved.** `fr-cerfa`'s worst DCT picture
reads 0.750 in `baseline/`; the section below shows it is the same matcher
artefact, and the population's worst *unambiguously paired* picture is 0.095.
The other five have not been audited for pairing. If a later audit removes one
of them, the band the cut sits in gets **wider**, not narrower, so 1% would
become more conservative rather than wrong — but the ceiling of that band is
`gh-pypdf`'s 0.047 and it rests on two unaudited pictures.

### Checked once at picture level, on one population

A median says half, so the table above is a statement about *populations* and
not about pictures. One population was therefore re-measured with every
picture's share written out rather than summarised — **`fr-cerfa` alone**, 450
documents, first page, at `render` v0.20.0, chosen because it is the largest
DCT sample the landed record cannot split. Its 105 differing `DCTDecode`
pictures land like this:

| | |
|---|---|
| 100 pictures | 0.00000046 … **0.0086996** |
| *(empty)* | |
| 5 pictures | **0.094990**, 0.250000, 0.375000, 0.571429, 0.750000 |

**The gap is there at picture level too**, spanning a factor of 10.9, and 1%
falls inside it. That is an independent confirmation and not a restatement: it
is one population's pictures rather than twenty populations' medians, and the
two bands overlap between 0.0087 and 0.047.

**Then the tail was audited for pairing, and four fifths of it is not
evidence.** `conformance#13` showed `match` manufactures disagreements when a
page draws many pictures of one size, so each of the 105 was checked against
what `pdfimages -list` says that page holds. Ten had a size `pdfimages` gave it
more than one picture of; ninety-five did not:

| | pictures | ≤ 0.0087 | above 1% |
|---|---:|---:|---|
| **pairing unambiguous** | 95 | 94 | **0.094990** |
| pairing ambiguous | 10 | 6 | 0.250000, 0.375000, 0.571429, 0.750000 |

All four of the largest are 7×1 and 8×1 slivers on page 1 of
`cerfa_12626.pdf` — where `pdfimages` extracts **2122 pictures of size 8×1 and
81 of size 7×1** — so they are paired at random among thousands of identical
sizes and their shares are two, three, four and six differing pixels out of
eight. **The one picture above the tolerance whose pairing is unambiguous is
`cerfa_12711.pdf`'s single 418×357 `Im0`, at 0.094990.**

That does not move the cut; it improves it. Removing the artefacts leaves the
band running 0.0087 → 0.0950 with evidence on both sides of it rather than
evidence below and noise above, and 1% still falls inside.

Under 1% that population's DCT agreement goes from **45.6%** (88 exact of 193
comparable) to **97.4%** (188 of 193), and the one picture worth looking at
stays on the list.

The same population's raw samples have **no such gap** — 535 differing pictures
running smoothly from 0.000185 to 0.857143, 7 of them under 1% and 111 under
10%. That is what a distribution looks like when a tolerance would be a dial
rather than a threshold, and it is the reason the lossless filters keep exact
equality rather than getting a wider band of their own.

### Where it is thin, and why it was not widened

**1% is near the *low* end of the picture-level band, not its middle.** It is
1.15× above `fr-cerfa`'s largest rounding picture and 9.5× below its smallest
non-rounding one. Both bands together would also admit 2%, which is their
common log centre — and 2% was not taken.

The reason is `uk-govuk`. Its DCT median is 0.000013 and its worst is
**0.013940**: one picture, in a population that is otherwise pure rounding,
sitting between 1% and 2%. Nobody has looked at it, and its pairing has not
been audited either, so it may be rounding, a real disagreement, or the same
matcher artefact `fr-cerfa`'s tail turned out to be. A tolerance of 2% decides
the question; 1% leaves it on the list. **When the data does not say which of
the three it is, the cut that keeps a picture visible costs a look and the
other costs whatever the picture was**, so the tolerance sits below it.

That is also the honest limit of the number. It sorts populations with a factor
of five to spare either way and sorts individual pictures with almost none near
the line. **A picture near 1% is a picture to look at, not a verdict** — and
after `conformance#13`, looking at it means checking the pairing before the
codec.

### What a landed record can and cannot say under it

**The instrument still counts exact matches.** It records, per filter, how many
were exact and the median and worst of the shares that were not — which is
enough to place the cut, as above, and *not* enough to recompute an agreement
rate under it: a median says half, and half of `ia-medical`'s 426 differing JPX
pictures is all the landed record can tell about how many of them are under 1%.
Recording the count under the tolerance is what a run must do next.

What can be said from the landed record is a floor and a ceiling. Eleven of the
eighteen lossy rows have a **worst** below 1%, so all **81** of their differing
pictures are inside the tolerance whatever their distribution. Four rows have a
**median** above it. Three are split by it in a proportion the record does not
give — `ia-medical` JPX (426), `fr-cerfa` DCT (105), `uk-govuk` DCT (18),
together 549 of the corpus's 658 differing lossy pictures. The re-measurement
above settles the second of those three, at 100 of 105; the other two would
need a run of their own.

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
