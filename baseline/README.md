# What go-pdfkit comes to today

This is a baseline: the fidelity of `go-pdfkit` against poppler, per population
and per image filter, written down so that tomorrow's regression can be seen.
It is not an argument that anything is right. It is the number that a later run
is subtracted from.

The records beside this file are what `images -json` wrote, unedited. Each one
carries its own conditions — the corpus, the judge's version, every module
version it was built against, and when it was taken — because a figure that
falls between two runs means a regression **only if everything else held**, and
a drop is otherwise as likely to be a newer poppler or a corpus that grew.

## Conditions

| | |
|---|---|
| taken | 2026-08-30T16:34:29Z .. 2026-08-30T18:16:00Z (UTC) |
| judge | pdfimages version 26.04.0 |
| `go-pdfkit/render` | **v0.19.0** |
| `go-pdfkit/reader` | v0.6.0 |
| `go-gfx/gfx` | v0.16.0 |
| `tannevaled/gobig2` | v0.1.0 |
| `ajroetker/go-jpeg2000` | v0.0.2 |
| pages per document | 1 (the first page of each document) |
| corpora | `/Users/Shared/pdfscans` (MANIFEST.tsv), `/Users/Shared/pdfforms` (MANIFEST.tsv) |

**The threshold used for every filter in these records is exact equality**, and
that choice decides the answer, so it is stated rather than assumed. A picture
counts as agreeing only when every pixel agrees on whether it is ink. That is
defensible here and would not be for a page: `pdfimages` extracts rather than
renders, so there is no rasteriser between the codec and the pixels. `compare`,
which draws whole pages, cannot use this threshold and does not.

It is defensible for a **lossless** filter, where the bytes determine the
pixels and two implementations either agree bit for bit or one of them is
wrong. It is **not** defensible for `DCTDecode` and `JPXDecode`, and finding 2
below is the measurement that says so. The repository has since settled the
question the other way for those two — [the README](../README.md) states a
tolerance of 1% of pixels, read off the very numbers in this document — but
**these records predate it and every rate in them is an exact-equality rate**,
including the two lossy ones.

**There are deliberately no timings in this document.** Another job was running
on the machine throughout, so every duration measured here would be a
measurement of that job as much as of this one. Counts and pixel comparisons
are unaffected by load; wall-clock is not. The absence is a decision, not an
oversight.

## How to read the columns

A picture that could not be compared is not evidence, and is counted apart
rather than folded into a disagreement:

- **unopenable** — neither implementation would open the document. `pdfinfo` was
  asked about every document ours refused. This is a fact about the corpus. **A
  population's real size is its documents minus this.**
- **refused** — ours would not open or draw it *and poppler would*. This is the
  only one of these counts that means a defect.
- **declined** — ours drew pictures and `pdfimages` took none out, so there was
  nothing to compare against.
- **inverted** — ours and theirs are exact complements. `pdfimages` writes a
  **stencil** — `/ImageMask true`, or a stream named as a picture's `/Mask` —
  with the opposite polarity to its samples, whatever filter it arrived in;
  that is a convention, not a disagreement. Soft masks are not affected.
- **remapped** — the picture carries a `/Decode` array, which a viewer applies
  and `pdfimages` does not. The two sides were not asked the same question.
- **unmatched** — theirs had no picture of that size to pair with.
- **agreement** — exact ÷ comparable, where comparable is pictures minus
  remapped minus inverted. A filter with nothing comparable reads `n/a`, not
  0%: nothing to compare is not evidence of being wrong.

A population that was **not run** says so in the table, by name. It is never
omitted, because a reader who cannot tell a population that scored badly from
one that never ran has a report that misleads in the direction of comfort.

## The fleet, per population

Scanned pages — `/Users/Shared/pdfscans`:

| population | documents | unopenable | refused | declined | pictures | compared | exact | agreement |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `ia-medical` | 250 | 0 | 0 | 0 | 745 | 530 | 102 | 19.2% |
| `ia-biodiversity` | 250 | — | — | — | — | — | — | **not run** (killed by the memory watchdog at 24.9 GB, `rc=137` — finding 1) |
| `ia-americana` | 250 | — | — | — | — | — | — | **not run** (exceeded the 45-minute cap, `rc=124` — never returned, memory flat) |
| `ia-texts` | 12 | 7 | 0 | 0 | 13 | 10 | 4 | 40.0% |
| `ia-uscourts` | 250 | 0 | 0 | 0 | 134 | 95 | 70 | 73.7% |

Government and library forms — `/Users/Shared/pdfforms`:

| population | documents | unopenable | refused | declined | pictures | compared | exact | agreement |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `ca-cra` | 84 | 0 | 0 | 0 | 151 | 0 | 0 | n/a |
| `fr-cerfa` | 450 | 0 | 0 | 0 | 4390 | 4130 | 3446 | 83.4% |
| `fr-impots` | 50 | 0 | 0 | 0 | 102 | 73 | 28 | 38.4% |
| `gh-openpdf` | 56 | — | — | — | — | — | — | **not run** (killed by the memory watchdog at 27.5 GB, `rc=137` — finding 1) |
| `gh-pdfbox` | 157 | 8 | 0 | 1 | 41 | 39 | 20 | 51.3% |
| `gh-pdfcpu` | 147 | 0 | 0 | 0 | 696 | 696 | 653 | 93.8% |
| `gh-pypdf` | 34 | 1 | 0 | 0 | 47 | 45 | 41 | 91.1% |
| `gh-qpdf` | 81 | 0 | 0 | 0 | 70 | 70 | 42 | 60.0% |
| `gh-safedocs` | 26 | 5 | 0 | 0 | 2 | 2 | 1 | 50.0% |
| `gh-verapdf` | 134 | 0 | 0 | 0 | 0 | 0 | 0 | n/a |
| `int-wipo` | 116 | 0 | 0 | 0 | 0 | 0 | 0 | n/a |
| `uk-govuk` | 302 | 0 | 0 | 0 | 225 | 172 | 153 | 89.0% |
| `us-dol` | 140 | 0 | 0 | 0 | 47 | 26 | 20 | 76.9% |
| `us-irs` | 69 | 0 | 0 | 0 | 8 | 1 | 0 | 0.0% |
| `us-opm` | 66 | 0 | 0 | 0 | 37 | 5 | 5 | 100.0% |
| `us-ssa` | 199 | 0 | 0 | 0 | 8 | 0 | 0 | n/a |
| `us-uscis` | 88 | 0 | 0 | 0 | 87 | 1 | 0 | 0.0% |
| `us-uscourts` | 69 | 0 | 0 | 0 | 2 | 0 | 0 | n/a |

## The fleet, per filter

**`compared` is the denominator every agreement figure is computed over** —
pictures minus remapped minus inverted. It is printed beside `pictures`
because the two are far apart and only one of them is the claim: `DCTDecode`
has 477 pictures and 355 of them were compared, so "117 exact" is 117 of 355
and never 117 of 477.

| filter | pictures | **compared** | exact | agreement | inverted | remapped | unmatched | differing | worst |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `JPXDecode` | 518 | 518 | 80 | **15.4%** | 0 | 0 | 0 | 438 | 0.8988 |
| `DCTDecode` | 499 | 377 | 125 | **33.2%** | 1 | 121 | 33 | 219 | 0.8805 |
| `(samples)` | 4256 | 3810 | 3236 | **84.9%** | 173 | 273 | 13 | 561 | 0.8571 |
| `DCTDecode mask` | 12 | 12 | 11 | **91.7%** | 0 | 0 | 0 | 1 | 0.0000 |
| `(samples) mask` | 1249 | 1133 | 1088 | **96.0%** | 81 | 35 | 1 | 44 | 0.9583 |
| `JBIG2Decode` | 11 | 10 | 10 | **100.0%** | 0 | 1 | 0 | 0 | — |
| `JBIG2Decode mask` | 258 | 33 | 33 | **100.0%** | 199 | 26 | 0 | 0 | — |
| `JPXDecode mask` | 2 | 2 | 2 | **100.0%** | 0 | 0 | 0 | 0 | — |

## What this says

### 1. Nothing is refused that poppler can read

`refused` is **0** across all 2724 documents of the 20 populations that ran.
Every document ours would not open, poppler would not open either — 21 of them,
and 7 of those are the DRM'd `ia-texts` documents. For a reader that is the result you want: the
coverage gap against the field is empty.

### 2. Exact equality is the wrong threshold for a lossy codec, and it is the one used here

`JPXDecode` reads 15.4% and `DCTDecode` 33.0%, which look like the two worst
things in the corpus. They are not, and the report would mislead if it stopped
at the rate. The differences are mostly minute:

| population | filter | differing | median | worst |
|---|---|---:|---:|---:|
| `ia-medical` | JPXDecode | 426 | 0.001966 | 0.370584 |
| `fr-cerfa` | DCTDecode | 105 | 0.000135 | 0.750000 |
| `gh-pdfcpu` | DCTDecode | 43 | 0.000909 | 0.002234 |
| `uk-govuk` | DCTDecode | 18 | 0.000013 | 0.013940 |
| `ia-uscourts` | DCTDecode | 14 | 0.000086 | 0.002106 |

A median of 0.000135 is one pixel in seven thousand. JPEG and JPEG 2000 do not
require bit-exactness of any decoder — the inverse transform is specified to a
tolerance, not to a value — so two conformant implementations *may* differ and
counting that as a failure measures conformance to poppler rather than to the
format.

**This contradicted the premise stated in the repository's README**, that "two
implementations of one image format either agree bit for bit or one of them is
wrong". That is true of `CCITTFaxDecode` and `JBIG2Decode`, which are lossless
and where the corpus duly reads 100%. It was false of `DCTDecode` and
`JPXDecode`. Those two need a tolerance, and **the two rates above are a floor
and not a verdict**.

The tail is where the real question is: `ia-medical` JPX has a worst of 0.37
and `gh-pdfbox` a median of 0.899, and those are not rounding.

**The premise has since been corrected, using this table.** The README now
judges the lossless filters bit-exact and the two lossy ones by a tolerance of
1% of pixels, chosen because the eighteen lossy rows of these records fall into
a group of fourteen whose medians run from 0.000011 to 0.001966 and a group of
four running from 0.047161 to 0.898821, with nothing between — and 1% is the
round number at the middle of that empty band. **Every rate printed in this
document is still the exact-equality rate**, because that is what the run
recorded; the tolerance changes how they are read, not what they are.

One population was re-measured to check that, with every picture's share
written out instead of summarised. **It is one population — `fr-cerfa`, 450
documents, first page — taken at `render` v0.20.0**, and it is reported as one:
nothing here is a new corpus figure. Its 105 differing `DCTDecode` pictures run
from 0.00000046 to 0.0086996 and then jump to 0.094990, so the empty band is
there at picture level too and 1% falls inside it. Under the tolerance that
population's DCT agreement reads 97.4% (188 of 193 comparable) where this
document records 45.6% (88 of 193).

**Each of the 105 was then checked against what `pdfimages -list` says its page
holds**, because finding 3 above shows `match` manufactures disagreements when
a page draws many pictures of one size. Ten had an ambiguous size and 95 did
not, and the split is the whole tail: of the 95 unambiguous, 94 are at or below
0.0087 and **one** is above — `cerfa_12711.pdf`'s single 418×357 picture, at
0.094990. The four largest shares in this population's DCT column — 0.750000,
0.571429, 0.375000, 0.250000 — are all 7×1 and 8×1 slivers on page 1 of
`cerfa_12626.pdf`, where `pdfimages` extracts 2122 pictures of size 8×1 and 81
of size 7×1, so they are paired at random and are two to six differing pixels
out of eight. **The worst DCT disagreement in `fr-cerfa` is 0.095, not 0.750**,
and the 0.750 recorded above is the matcher.

That run is also the only measurement of how far the version seam moved a
population, and the answer for this one is: barely. At v0.20.0 with the
name-collapsing removed, `fr-cerfa` reproduces the table above exactly for
`DCTDecode` (206 pictures, 88 exact, 105 differing, median 0.000135, worst
0.750000), for `(samples) mask`, for `DCTDecode mask` and for `JPXDecode`. Only
`(samples)` moved, from 3414 pictures to 3404 and from 45 remapped to 35, with
its exact, inverted, differing, median and worst all unchanged — including the
173 in the `inverted` column, which finding 3 above and
[conformance#13](https://github.com/go-pdfkit/conformance/issues/13) show are
144 parts matcher and 29 parts `pdfimages`, and no part decoder. **One
population is not the corpus**, and the other nineteen are not re-measured
here.

**One caveat was checked before any of this was written.** `decodeJPEG` prefers
the codestream's dimensions over the dictionary's when the two disagree, while
`pdfimages` lists the dictionary's — so for those files the comparison would be
lining up the wrong grid, and the DCTDecode figure would be about the matcher
rather than about JPEG. Counted over 362 documents of the forms corpus, with
repeated draws collapsed:

| filter | our size agrees with the judge's | no picture of our size |
|---|---:|---:|
| `DCTDecode` | 252 | **1** |
| `(samples)` | 3620 | 1 |
| `(samples) mask` | 773 | 1 |

One DCT picture in 253. The disagreement is real — in `2735_2735_5089.pdf` ours
is 233×113 where poppler lists 173×87 — and it is rare enough that it does not
touch the rate. The finding stands, and stands better for having been checked.

### 3. Inversions are concentrated, not spread

An exact complement is a convention rather than a disagreement, but *where* it
occurs says whether it is understood. It is not thinly spread:

| filter | inverted | where |
|---|---:|---|
| `JBIG2Decode mask` | 199 | `ia-medical` 193 of 247, then 3 and 3 |
| `(samples)` | 173 | **`fr-cerfa` alone**, 173 of 3414 |
| `(samples) mask` | 81 | `ia-uscourts` 34 of 51, `us-opm` **28 of 29**, `fr-cerfa` 15 of 762 |
| `DCTDecode` | 1 | `uk-govuk` |

The two mask concentrations are the stencil polarity convention: `pdfimages`
writes every stencil with the opposite polarity to its samples, and `us-opm`'s
28 are 28 of 28 `/ImageMask true`. They are expected, and the JBIG2 line is the
same thing rather than a JBIG2 thing.

**The `(samples)` line is not an inversion at all**, and the concentration was
the clue. All 173 are in 8 documents; 168 of them are 2x2 pixels and none is
larger than four. Those documents set coloured text by stretching a solid
2x2 swatch under a large `/SMask`, and page 1 of `cerfa_10074.pdf` draws 211 of
them, every one uniform. `match` pairs a picture with the first unclaimed
picture of the same size, so a black swatch of ours is paired with a white
swatch of theirs and `difference` reads 1.0. Matched by object identity
instead, the 8 documents come to 3414 agreeing, 29 complements and 6 differing,
where this instrument reads 2906, 173 and 370. **144 of the 173 are the
matcher.** The 29 that survive are a second `pdfimages` behaviour: a one-bit
`/Indexed` picture is written black only when the palette entry is exactly
`#000000`, so a `#333333` swatch comes out as paper. Both are recorded in
[conformance#13](https://github.com/go-pdfkit/conformance/issues/13), with the
minimal documents that reproduce them.

### 4. The comparison reduces colour to one bit, which makes colour pictures fragile

`difference` asks only whether a pixel is ink — luminance below 128, alpha
above it. For a bilevel scan that is exactly right and is why the CCITT and
JBIG2 numbers can be trusted. For a colour picture it is a bisection: a small
shift in colour management moves pixels across the threshold wholesale.

`fr-cerfa` shows the effect at its strongest, and it is the largest single
disagreement in the corpus — 535 differing `(samples)` pictures at a median of
0.286, a quarter of every pixel — and poppler reports those pictures as `icc`
(3 components, 8 bits) and `index`. Two implementations doing different ICC
conversion, compared through a luminance bisection, can disagree on a quarter
of the pixels while both being right.

**So this is named as the top candidate and explicitly not as a defect.** It
has not been established either way, and the check that would settle it is a
per-channel comparison rather than an ink/paper one.

## The version seam: these records are v0.19.0, the instrument is now v0.20.0

These records were taken against **`render` v0.19.0**, which is what the
`modules` block of every JSON file says and what the conditions table above
repeats. **The instrument that took them has since moved to v0.20.0**, so a run
taken today is on the far side of a seam from every number in this document,
and the two must not be subtracted from one another without saying so.

The seam is what `Images` returns. Under v0.19.0 a page that draws the same
XObject repeatedly yields one entry per draw; under v0.20.0 it yields one per
picture. The instrument compensated for v0.19.0's behaviour by taking each
picture once, keyed on the name the page draws it by, and **that compensation
has been removed**, because against v0.20.0 it is not merely redundant but
wrong: a name is unique within a resource dictionary and not across them.

The figure that decided it was re-measured rather than inherited. An earlier
note here read *"over 368 documents, 16 of them have two pictures sharing a
name … collapsing on the name would drop 28 distinct pictures"*. Measured again
under v0.20.0 over the **whole** forms corpus — 2268 documents, first page,
825 of which draw any picture at all — the count is larger:

| population | documents whose first page shares a name | pictures the compensation would drop |
|---|---:|---:|
| `gh-qpdf` | 16 | 28 |
| `fr-impots` | 24 | 36 |
| **total** | **40 of 2268** | **64** |

The 16 and the 28 are exactly the earlier note's figure: it was counting
`gh-qpdf` alone. `gh-qpdf` is qpdf's `form-xobjects-*` and `shared-form-*`
fixtures, where two form XObjects each name their own `Im1`, and a fixture
corpus is meant to hold cases like that. **`fr-impots` is not a fixture
corpus.** Twenty-four of its fifty documents — one issuer's real tax forms —
put two different pictures under `Im0` on their first page, and it is that half
of the count which says the compensation had to go rather than merely could.

The same sweep over `/Users/Shared/pdfscans` was started and stopped before it
finished, so **there is no scanned-corpus figure here**; the number above is
the forms corpus and says nothing about the other one.

## What is not measured, and why

Three populations were attempted and did not finish. They are in the tables above
by name, marked **not run**, because a population that gave trouble is exactly
the one whose omission would flatter the result.

Two of them — `gh-openpdf` at 27.5 GB and `ia-biodiversity` at 24.9 GB — were
killed by a memory watchdog. `render` v0.19.0 walks a page's resources as a
tree when they are a graph, and a document whose form XObjects name each other
fans out combinatorially. One document reached **87 GB resident** before being
killed. It reproduces in a plain `render.Images(d, 1)` call with none of this
repository's code involved, and `render` v0.20.0 fixes it.

The third, `ia-americana`, is a different failure and is recorded as one: it
ran to the 45-minute cap with memory flat, so it was not the allocation defect.
Whether it is merely a slow population of large scans or something that does
not terminate is **not known**, and saying so is more useful than guessing.

Re-running these three is the first thing a later run should do, and under
v0.20.0 two of them should now complete.
