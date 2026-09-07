# What go-pdfkit comes to today

This is a baseline: the fidelity of `go-pdfkit` against poppler, per population
and per image filter, written down so that tomorrow's regression can be seen.
It is not an argument that anything is right. It is the number that a later run
is subtracted from.

The records beside this file are what `images -json` wrote, unedited. Each one
carries its own conditions — the corpus, the judge's version, the **gate** the
comparison used, **the bound the judge was held to**, every module version it
was built against, and when it was taken — because a figure that falls between
two runs means a regression **only if everything else held**, and a drop is
otherwise as likely to be a newer poppler, a corpus that grew, or a different
instrument.

## Conditions

| | |
|---|---|
| taken | 2026-09-07T19:30:25Z .. 2026-09-07T21:35:00Z (UTC) |
| judge | pdfimages version 26.04.0 |
| **measure** | **per channel, gate `D` = 2, count budget `N` = 0** ([conformance#16](https://github.com/go-pdfkit/conformance/issues/16)) |
| **pairing** | **by object number, falling back to size only when one side published none** ([conformance#30](https://github.com/go-pdfkit/conformance/pull/30)) |
| **walk** | **the page's CONTENT STREAM, not its /Resources** ([render#43](https://github.com/go-pdfkit/render/pull/43)) |
| **bucketing** | the listing **and** the picture's own `/ColorSpace` ([conformance#20](https://github.com/go-pdfkit/conformance/issues/20)) |
| **bound on the judge** | **2m0s per document, per tool** ([conformance#21](https://github.com/go-pdfkit/conformance/issues/21)) |
| `go-pdfkit/render` | **v0.22.0** |
| `go-pdfkit/reader` | v0.6.0 |
| `go-gfx/gfx` | v0.19.0 |
| `tannevaled/gobig2` | v0.1.0 |
| `ajroetker/go-jpeg2000` | v0.0.2 |
| pages per document | 1 (the first page of each document) |
| corpora | `/Users/Shared/pdfscans` (MANIFEST.tsv), `/Users/Shared/pdfforms` (MANIFEST.tsv) |

**Every one of the 23 populations ran to completion, and every one is in the
tables below.** All 23 exited 0.

**Why this run exists.** The INSTRUMENT changed, and nothing else did. Twice
over, and both changes are about the same mistake: **asking one question and
measuring the answer to another.**

`render.Images` said it returned « the pictures the i'th page **draws**, in the
order of the names it **draws** them by ». It walked the page's `/Resources`
dictionary. A resource dictionary is a **catalogue of what a page may draw**, and
PDF lets every page in a file share one — the French tax forms do.
`2044_2044_4764.pdf` gives all ten of its pages the same dictionary, holding ten
118×118 Data Matrix barcodes, one per page; page 1 draws exactly one of them and
the walk returned all ten.

The judge extracts what a page **draws**, so the nine extras had nothing to pair
with — and one of them was matched to the drawn barcode's row, because all ten
are the same size. Two unrelated barcodes were compared, came out **255 apart on
every term**, and the barcode the page really draws was left unpaired and
unmeasured. `render#43` walks the content stream instead; `conformance#30` stops
the size fallback claiming a row when both sides published an object number and
the two differ.

Everything else was held, and checked before the run rather than asserted
afterwards: the same judge (`pdfimages version 26.04.0`), the same other module
versions listed above, the same one page per document, the same two-minute
bound. **The instrument is the only thing that moved**, which is what this file
demands of any figure it prints.

What that cost the run this replaces, over the same 23 populations, the same
3280 documents and the same judge:

| | v0.21.0 (19:30 earlier today) | v0.22.0 (this run) |
|---|---:|---:|
| pictures returned | 8190 | **8063** |
| **with no counterpart at all** | **110** | **5** |
| pictures compared | 7515 | 7513 |
| exact | 6598 | **6657** |
| reported inverted | 422 | 426 |
| reported differing | 495 | **430** |
| pages refused by our own budget | 4 | **1** |
| **agreement** | **93.0%** | **93.9%** |

**Read the first two rows before the last one.** 127 fewer pictures came back,
and 105 of them were pictures that had nothing on the judge's side to be
compared with — because the page never drew them. The agreement did not rise
because anything decodes better; it rose because 65 comparisons of two different
pictures stopped being counted as disagreements.

Eight of the 23 populations moved and **not one moved backwards** on any term —
not exact, not differing, not agreement. The other fifteen are identical to the
byte, which is what says this reached only the pages the defect could reach. `gh-qpdf` moved most: 45 disagreements to 5, and
28 exact to **58**, from **fewer** pictures than before. More right answers out
of less material is what a pairing being fixed looks like.

**It is not uniformly "fewer inversions", and that matters.** `ia-uscourts`
gained two — 37 to 39 — while its differing count fell by the same two: the
right pairing found two genuine complements the wrong one had hidden inside a
disagreement. A change that only ever removed inversions would be a change that
only ever flattered.

**What `exact` asserts here.** A picture agrees when **no channel of any pixel
differs from poppler's by more than two levels of 255**. Two is the ISO/IEC
10918-2 IDCT allowance either side of the reference, read out of
`libavcodec/tests/dct.c:259`; the derivation and the survey it sits in are in
[the repository README](../README.md). Comparing at all is defensible here and
would not be for a page: `pdfimages` **extracts** rather than renders, so there
is no rasteriser between the codec and the pixels. `compare`, which draws whole
pages, cannot use this criterion and does not.

**There are deliberately no timings in this document.** Another job was running
on the machine throughout, so every duration measured here would be a
measurement of that job as much as of this one. Counts and pixel comparisons
are unaffected by load; wall-clock is not. The absence is a decision, not an
oversight. **Peak memory was not captured for this run**, and the previous run's
figures — `ia-americana` 5.9 GB, `ia-biodiversity` 3.6 GB, `ia-medical` 3.2 GB —
are not carried over: this run decodes 127 fewer pictures and several very large
ones fewer, so quoting them here would be quoting a measurement of something
else.

## How to read the columns

A picture that could not be compared is not evidence, and is counted apart
rather than folded into a disagreement:

- **unopenable** — neither implementation would open the document. `pdfinfo` was
  asked about every document ours refused. This is a fact about the corpus. **A
  population's real size is its documents minus this.**
- **refused** — ours would not open or draw it *and poppler would*. It is not
  only a defect count: `render` refuses a page whose declared pictures exceed a
  256-megapixel budget, and that refusal is our own bound working. See finding 6.
- **hung** — a poppler tool did not answer within the bound, so the document was
  **named** rather than dropped. It is 0 everywhere in this run, and finding 5
  says why that is not the same as "nothing hangs".
- **declined** — ours drew pictures and `pdfimages` took none out, so there was
  nothing to compare against.
- **remapped** — the picture carries a `/Decode` array, which a viewer applies
  and `pdfimages` does not. The two sides were not asked the same question.
- **unmatched** — nothing of the judge's could be paired with this one: no row
  carried its object number, and none of the right size was free to fall back
  to. Since the walk returns what a page **draws**, this is now rare — 5 across
  the fleet, from 110 — and each one is a question rather than a fact of life.
- **converted** — the picture's colour space had to be converted to reach RGB.
  Per channel that arithmetic is large and is **not** a decoder disagreeing, so
  those pictures are tallied in their own bucket with their own agreement figure
  and their own magnitudes. **The bucket is now decided by both sides**: by what
  `pdfimages` lists (`cmyk`, `lab`, `icc`, `index`, `sep`, `devn`, or a row that
  could not be read), *and* by the picture's own `/ColorSpace` resolving to
  `CalRGB`, `CalGray`, `ICCBased` or `Lab` — because poppler lists a `/CalRGB`
  picture as `rgb` while converting it. That is
  [conformance#20](https://github.com/go-pdfkit/conformance/issues/20), and
  finding 3 measures what it moved.
- **calibrated** — how many of a converted bucket's pictures are CIE-tagged **in
  their own dictionary**. It is how much of the bucket the document itself
  accounts for, and it is **not** the count of what the new rule moved: most
  calibrated pictures are `ICCBased` or `Indexed`, which the listing already
  called converted.
- **inverted** — ours and theirs are exact complements within the gate.
  `pdfimages` writes a **stencil** — `/ImageMask true`, or a stream named as a
  picture's `/Mask` — with the opposite polarity to its samples, whatever
  filter it arrived in; that is a convention, not a disagreement.
- **agreement** — exact ÷ **direct comparable**, where direct comparable is the
  direct bucket's pictures minus its complements. It is never computed over the
  converted bucket. A filter with nothing comparable reads `n/a`, not 0%.

Beside the counts, each bucket that had anything differ carries the **median
and the far end of four magnitudes** over its differing pictures: **`peak`**
(the largest channel difference in the picture, in levels of 255 — this is the
criterion), **`share`** (the fraction of pixels where some channel exceeded the
gate), **`mse`** (mean squared error over the compared channels, in levels
squared) and **`mean`** (the **signed** mean error, ours minus theirs, whose far
end is the value furthest from zero *with its sign*, so a bias says which way it
ran).

A population that was **not run** would say so in the table, by name. None was.

## The fleet, per population

Scanned pages — `/Users/Shared/pdfscans`:

| population | documents | unopenable | refused | declined | hung | pictures | direct | inverted | compared | exact | identical | agreement | converted | calibrated |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ia-medical` | 250 | 0 | 0 | 0 | 0 | 745 | 722 | 193 | 529 | 528 | 33 | 99.8% | 1 | 1 |
| `ia-biodiversity` | 250 | 0 | 1 | 0 | 0 | 752 | 703 | 53 | 650 | 650 | 157 | 100.0% | 0 | 0 |
| `ia-americana` | 250 | 28 | 0 | 0 | 0 | 502 | 494 | 89 | 405 | 379 | 96 | 93.6% | 8 | 8 |
| `ia-texts` | 12 | 7 | 0 | 0 | 0 | 14 | 14 | 3 | 11 | 11 | 2 | 100.0% | 0 | 0 |
| `ia-uscourts` | 250 | 0 | 0 | 0 | 0 | 133 | 114 | 40 | 74 | 66 | 54 | 89.2% | 17 | 2 |

Government and library forms — `/Users/Shared/pdfforms`:

| population | documents | unopenable | refused | declined | hung | pictures | direct | inverted | compared | exact | identical | agreement | converted | calibrated |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ca-cra` | 84 | 0 | 0 | 0 | 0 | 151 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `fr-cerfa` | 450 | 0 | 0 | 0 | 0 | 4378 | 1338 | 15 | 1323 | 1189 | 1129 | 89.9% | 2978 | 1741 |
| `fr-impots` | 50 | 0 | 0 | 0 | 0 | 109 | 20 | 0 | 20 | 19 | 6 | 95.0% | 25 | 3 |
| `gh-openpdf` | 56 | 14 | 0 | 0 | 0 | 49 | 26 | 0 | 26 | 21 | 6 | 80.8% | 2 | 1 |
| `gh-pdfbox` | 157 | 8 | 0 | 0 | 0 | 41 | 29 | 1 | 28 | 26 | 23 | 92.9% | 11 | 1 |
| `gh-pdfcpu` | 147 | 0 | 0 | 0 | 0 | 696 | 639 | 0 | 639 | 599 | 597 | 93.7% | 57 | 32 |
| `gh-pypdf` | 34 | 1 | 0 | 0 | 0 | 15 | 8 | 0 | 8 | 6 | 6 | 75.0% | 6 | 3 |
| `gh-qpdf` | 81 | 0 | 0 | 0 | 0 | 63 | 63 | 0 | 63 | 58 | 39 | 92.1% | 0 | 0 |
| `gh-safedocs` | 26 | 5 | 0 | 0 | 0 | 2 | 2 | 0 | 2 | 1 | 1 | 50.0% | 0 | 0 |
| `gh-verapdf` | 134 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `int-wipo` | 116 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `uk-govuk` | 302 | 0 | 0 | 0 | 0 | 225 | 143 | 1 | 142 | 142 | 111 | 100.0% | 31 | 10 |
| `us-dol` | 140 | 0 | 0 | 0 | 0 | 46 | 25 | 0 | 25 | 10 | 10 | 40.0% | 0 | 0 |
| `us-irs` | 69 | 0 | 0 | 0 | 0 | 8 | 1 | 0 | 1 | 0 | 0 | 0.0% | 0 | 0 |
| `us-opm` | 66 | 0 | 0 | 0 | 0 | 37 | 31 | 28 | 3 | 3 | 3 | 100.0% | 2 | 0 |
| `us-ssa` | 199 | 0 | 0 | 0 | 0 | 8 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `us-uscis` | 88 | 0 | 0 | 0 | 0 | 87 | 0 | 0 | 0 | 0 | 0 | n/a | 1 | 0 |
| `us-uscourts` | 69 | 0 | 0 | 0 | 0 | 2 | 2 | 2 | 0 | 0 | 0 | n/a | 0 | 0 |

## The fleet, per filter

**`compared` is the denominator every agreement figure is computed over** — the
direct bucket's pictures minus its complements. It is printed beside `pictures`
because the two are far apart and only one of them is the claim, and beside
`identical`, because a rate is not a claim of bit equality.

| filter | pictures | direct | inverted | **compared** | exact | identical | agreement | converted | conv. exact | conv. differing | calibrated | remapped | unmatched | differing | worst peak |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `(samples)` | 4280 | 904 | 0 | 904 | 902 | 902 | **99.8%** | 3085 | 2921 | 164 | 1761 | 286 | 5 | 2 | 255 |
| `(samples) mask` | 1336 | 1269 | 154 | 1115 | 1093 | 1093 | **98.0%** | 0 | 0 | 0 | 0 | 67 | 0 | 22 | 255 |
| `JPXDecode` | 1241 | 1231 | 0 | 1231 | 1231 | 7 | **100.0%** | 10 | 9 | 1 | 9 | 0 | 0 | 0 | 255 |
| `JBIG2Decode mask` | 591 | 521 | 271 | 250 | 250 | 250 | **100.0%** | 0 | 0 | 0 | 0 | 70 | 0 | 0 | — |
| `DCTDecode` | 590 | 425 | 0 | 425 | 208 | 4 | **48.9%** | 44 | 19 | 24 | 32 | 121 | 0 | 217 | 255 |
| `DCTDecode mask` | 12 | 12 | 0 | 12 | 12 | 5 | **100.0%** | 0 | 0 | 0 | 0 | 0 | 0 | 0 | — |
| `JBIG2Decode` | 11 | 10 | 0 | 10 | 10 | 10 | **100.0%** | 0 | 0 | 0 | 0 | 1 | 0 | 0 | — |
| `JPXDecode mask` | 2 | 2 | 0 | 2 | 2 | 2 | **100.0%** | 0 | 0 | 0 | 0 | 0 | 0 | 0 | — |

Across the whole fleet: **3385 of 3957 direct comparable pictures agree, 85.5%**,
and 1990 of them are bit-identical. At v0.20.0 it was 3347 of 3962, **84.5%**,
with the same 1990 identical.

## What this says

**Ten findings, and they are about three runs.** §1 to §5 and §7 were written
about the v0.20.0 → v0.21.0 comparison of 2026-08-31, and §8 about the pairing
change of 2026-09-07; they are kept because their reasoning still holds, and
where they quote a figure, that figure is the one their own run measured. **§6
is disproved by THIS run** and is rewritten rather than left standing — three of
the four refusals it called permanent are gone. §9 is updated, because the gap
it names moved. §10 is new and belongs to this run.

### 1. The claimed "61.2% → 99.3%" is not what this instrument measures, and the correction is not a smaller improvement — it is a different quantity

The figure in circulation for `render` v0.21.0 is **61.2% → 99.3% corpus-wide**.
It came from one sweep's own count, and it is **not reproducible from the landed
instrument**. Two things about it:

- it is **`DCTDecode` alone**, not the corpus, over the 559 pictures
  `pdfimages -list` calls `gray` or `rgb`;
- it is at a **gate of 4**, not the landed gate of 2.

And that denominator no longer exists in this repository, because
[conformance#20](https://github.com/go-pdfkit/conformance/issues/20) is exactly
the finding that *"what `pdfimages -list` calls `rgb`"* silently includes
`/CalRGB`.

**What the landed instrument says, at gate 2, over all 23 populations:**

| | v0.20.0 | v0.21.0 |
|---|---:|---:|
| `DCTDecode`, direct comparable | 430 | 426 |
| exact | 146 | 184 |
| **agreement** | **34.0%** | **43.2%** |
| whole fleet, direct comparable | 3962 | 3957 |
| **whole-fleet agreement** | **84.5%** | **85.5%** |

**Whole fleet here means direct-comparable pictures on the first page of each
document, across all 23 populations** — not documents, not pages, and not the
converted bucket, which is 3139 pictures this figure declines to score at all.
A rate quoted without that is the kind of rate this section exists to correct.

**And those two columns differ by three changes, not one**, which matters if the
figure is quoted as `render` v0.21.0's: the denominators are 3962 and 3957
because #20's bucketing rule moved five pictures out, and `go-gfx/gfx` went
v0.16.0 to v0.19.0 in between. All five moved pictures differ at v0.21.0 — the
four CalRGB ones at peaks 110, 11, 10, 10 and the `(samples)` one in
`ia-uscourts`, which differs in both runs — so **holding the bucketing at
v0.20.0's, v0.21.0 reads `DCTDecode` 184/430 = 42.8% and the fleet 3385/3962 =
85.4%.** That is the figure attributable to the library rather than to the
instrument, and it is still not v0.21.0 in isolation because of the `gfx` bump.
The cleanest single attributable number is `uk-govuk`, which moved no pictures
between buckets: **92.3% → 100.0%**, 131 of 142 to 142 of 142.

Nine points on JPEG and one on the fleet. **That is the honest number, and it
understates what happened, because the gate is a cliff and the change was a
change of magnitude.** The magnitudes are where it shows. The per-population
medians of the worst channel on the **direct** `DCTDecode` rows were:

```
v0.20.0   16  18  23  23  26  31  34  47  50  58  62  171  244  255      (14 rows)
v0.21.0    3   3   3   3   3   3   4   4   4    4      171  233  255     (13 rows)
```

**Ten of thirteen rows now sit one or two levels above a gate of two**, where
eleven of fourteen used to sit between 16 and 62. The row that vanished is
`uk-govuk`, whose eleven differing pictures became none: **92.3% → 100.0%**.
`fr-cerfa`'s went from 139 differing at a median peak of 31 to 114 at a median
peak of **3**.

Paired by population rather than sorted, which is how it should be read — the
two ordered lists above are **not** aligned, and pairing their columns would be
wrong:

| population | v0.20.0 | v0.21.0 |
|---|---:|---:|
| `uk-govuk` | 18 | **no differing picture at all** |
| `gh-safedocs` | 16 | 4 |
| `ia-americana` | 23 | 4 |
| `ia-uscourts` | 23 | 3 |
| `gh-pdfcpu` | 26 | 3 |
| `fr-cerfa` | 31 | 3 |
| `us-dol` | 34 | 3 |
| `us-irs` | 47 | 3 |
| `gh-openpdf` | 50 | 3 |
| `gh-pdfbox` | 58 | 4 |
| `ia-medical` | 62 | 4 |
| `gh-pypdf` | 244 | 233 |
| `gh-qpdf` | 171 | 171 |
| `fr-impots` | 255 | 255 |

So the improvement is real and it is large, and it is a **magnitude** result
rather than a rate result at this gate. **Three rows stayed gross** —
`gh-qpdf` 171 → 171, `fr-impots` 255 → 255, and `gh-pypdf` 244 → 233, which
moved but not to anywhere near rounding. None of the three is chroma
reconstruction. They are the remaining `DCTDecode` finding, and this run does
not diagnose them.

**Nothing here changes the gate.** That ten rows now sit at 3 and 4 is evidence
about `D` and it is recorded, not acted on: a gate moved because it would flatter
a number is not a gate.

### 2. The empty band came back, and the previous run's argument against it does not survive its own library

The previous baseline found that the empty band which once justified a 1%
tolerance **was gone**: 2 of 21 direct rows under 1%, largest gap a factor of
6.8, and the two low rows carrying peak medians of **18 and 23** — *"sparse gross
error"*, it said, which a 1% budget would forgive.

At v0.21.0 the same twenty direct rows read:

```
0.000112 0.000136 0.000140 0.000163 0.000177 0.000349 0.000386 0.000508
0.000595 0.001119 0.072224 0.222222 0.257613 0.412628 0.888889 0.894813
0.958333 0.994521 1.000000 1.000000
```

**Ten of twenty are under 1%**, and the gap between the tenth and the eleventh is
a factor of **64.5** (0.001119 to 0.072224) — an emptier band than the factor of
24 the withdrawn 1% was originally read from. And the low group's peak medians
are **3 and 4**, not 18 and 23.

**The previous run's finding was true of `render` v0.19.0/v0.20.0 and is not
true of v0.21.0.** What made the low group "sparse gross error" was the chroma
defect; with it fixed, the low group is what the band was always claimed to be —
rounding, one or two levels over the gate.

**It does not put the 1% back, and nothing here proposes to.** With `N` at 0 the
criterion is the peak and the share is a report, so a band in the *share* is not
an argument about `N` at all; what it is evidence about is `D`, and the field's
answer to a per-pixel bound is a per-pixel bound. Both readings are recorded so
that the next person does not have to re-derive them, and neither is spent.

### 3. The colour-space rule moved five pictures, and four of them are the four it was filed for

[conformance#20](https://github.com/go-pdfkit/conformance/issues/20) is fixed in
this run: a picture whose own `/ColorSpace` resolves to `CalRGB`, `CalGray`,
`ICCBased` or `Lab` is converted whatever `pdfimages -list` says.

**How many pictures that moves is measurable exactly**, because the two runs are
comparable in the one respect that matters here: the per-filter **`pictures`**
and **`unmatched`** counts are **identical, filter for filter**, across v0.20.0
and v0.21.0 — 652, 4292, 1357, 1264, 600, 12, 11, 2 and 61, 7, 1, 29, 12. Neither
the set of pictures nor the size-matching changed, so **every difference in the
direct/converted split is the rule and nothing else**. (Those counts are that
comparison's, not this run's: §6 and §10 explain why the picture counts fell.)

| population | direct | converted | moved |
|---|---|---|---:|
| `fr-cerfa` | 1341 → 1338 | 2975 → 2978 | 3 |
| `gh-openpdf` | 28 → 27 | 1 → 2 | 1 |
| `ia-uscourts` | 115 → 114 | 16 → 17 | 1 |
| **fleet** | **3962 → 3957** | | **5** |

Per filter, **4 of the 5 are `DCTDecode`** (direct 430 → 426) and one is
`(samples)`. The four are the four the issue named — `gh-openpdf`'s PDF 2.0
image with a D50 white point, and `fr-cerfa`'s `cerfa_11612`, `cerfa_11616`,
`cerfa_12625` at gamma 2.22221. Run as their own population at v0.21.0 they
carry peaks of **110, 10, 10 and 11**, against **3** for the one `DCTDecode`
picture on those pages that is not CalRGB. That is the whole shape of the issue
in one population: after the chroma fix, everything left above the ISO allowance
on those documents was a colour conversion.

**`calibrated` is a different and much larger number** — 1802 pictures across the
fleet, 1741 of them in `fr-cerfa` — because it counts every CIE-tagged picture in
a converted bucket, and almost all of those were `ICCBased` or `Indexed`, which
the listing already called converted. It says how much of the converted bucket
the document itself accounts for. **It is not what the rule moved**, and adding
it up as though it were would overstate the change by two orders of magnitude.

### 4. The only filter that moved is the only filter the change touched

This is the check on the instrument rather than on the library, and it passes
exactly. Every other filter's **`exact` count is identical, picture for
picture** — 638, 1074, 1215, 12, 10, 250, 2 — and the only denominator that
moved anywhere is the one `(samples)` picture #20 reclassified:

| filter | v0.20.0 | v0.21.0 |
|---|---:|---:|
| `DCTDecode` | 34.0% | **43.2%** |
| `(samples)` | 69.7% | 69.7% |
| `(samples) mask` | 96.2% | 96.2% |
| `JPXDecode` | 99.2% | 99.2% |
| `DCTDecode mask` | 100.0% | 100.0% |
| `JBIG2Decode` | 100.0% | 100.0% |
| `JBIG2Decode mask` | 100.0% | 100.0% |
| `JPXDecode mask` | 100.0% | 100.0% |

`render` v0.21.0 changed JPEG chroma reconstruction and nothing else, and the
measurement says so: every other filter's agreement, and the fleet's identical
count of 1990, are unchanged. **`JPXDecode` stays at 99.2% with 7 of 1225
bit-equal** — a conformant lossy decoder, agreeing within two levels
almost everywhere and bit-equal almost nowhere, exactly as ISO/IEC 15444
promises.

`DCTDecode mask` is worth a line of its own: **12 of 12, of which 5 identical**.
A JPEG that is a mask is compared in ink coverage and agrees.

### 5. Nothing hung — and that is not the same as "nothing hangs"

`hung` is **0** in all 23 populations. Every poppler invocation in this run
carried the 2m0s bound of
[conformance#21](https://github.com/go-pdfkit/conformance/issues/21), and none
of them fired.

**That is not evidence that the corpus holds nothing that hangs**, and the
record must not be read as if it were. `pdfforms/gh-qpdf/qpdf_qtest_qpdf_shared-unnamed-field.pdf`
— 2496 bytes — hangs `pdfimages`, `pdfimages -list` **and** `pdfinfo`, all three
confirmed. `images` never meets it: the document opens for us, its first page
draws no picture, so `render.Images` returns nothing and poppler is never asked
about it at all. `compare`, which draws every page, does meet it, and under the
bound it names it:

```
$ compare -dir … -only check -timeout 20s
check   0 compared     1 not
            1  hung
        hung  pdftoppm  …/qpdf_qtest_qpdf_shared-unnamed-field.pdf page 1
```

So the bound is **unexercised by `images` on this corpus**, and its value is that
the next document does not stop a sweep dead at document 900 of 2268 with nothing
to distinguish the stall from a long job.

### 6. Three of the four refusals were the walk, not the budget

`refused` is **1** across 3280 documents, down from 4, and the one that remains
is a real page: `bulletinno38tasm.pdf`, whose page 1 has already spent all but
9 513 958 of its 268 435 456 pixels when it reaches a picture of 9449 × 13701.
That page genuinely draws more than the budget allows.

The other three now decode, and the previous run's own text said why without
knowing it. It called `bulletindelasoci4243soci.pdf` « the telling one: an
ordinary 794 × 1372 picture refused because the page had already spent all but
223 089 of its 268 435 456 pixels ». The page had not spent them on what it
draws. It had spent them on every picture in the **document**, because the walk
read the shared `/Resources` dictionary. Asked what it draws, that page comes
back with **three** pictures and 25 155 674 pixels — a tenth of the budget:

| document | v0.21.0 | v0.22.0 |
|---|---|---|
| `bulletinno38tasm.pdf` | refused | **refused** (9449 × 13701, genuinely) |
| `checklistofbirds00wood.pdf` | refused | 3 pictures, 28 922 225 px |
| `informeacercade00soci.pdf` | refused | 3 pictures, 23 155 356 px |
| `bulletindelasoci4243soci.pdf` | refused | 3 pictures, 25 155 674 px |

**A budget refusing three pages out of four for work nobody asked for is not a
budget being conservative, it is an instrument answering the wrong question** —
and it presented as a resource limit, which is one of the most convincing
disguises a defect has. The previous run named the anomaly precisely and drew
the wrong conclusion from it, because the number it needed to doubt was the one
it was measuring with.

**`Missing.Ours` still folds "we could not read this" together with "we chose not
to decode this", and only the first is a defect.** The instrument should tell
them apart and does not; the one that remains is named here so nobody reads the
1 as a coverage gap.

`unopenable` is 63 and `declined` is **0**, down from 1, and neither is a defect.

### 7. The gate still buys the lossless filters nothing

| filter | compared | exact | identical |
|---|---:|---:|---:|
| `JBIG2Decode` | 10 | 10 | **10** |
| `JBIG2Decode mask` | 250 | 250 | **250** |
| `JPXDecode mask` | 2 | 2 | **2** |
| `(samples)` | 915 | 891 | **891** |
| `(samples) mask` | 1115 | 1074 | **1074** |

**Every agreeing picture of every lossless filter is bit-equal**, as at v0.20.0
and as under the pairing this run replaced.
Not one needed the gate, so the uniform `D = 2` costs the filters that could
defensibly be held to 0 nothing, and there is still no measured case for a
per-filter exception table.

### 8. Most of the inversions were the pairing, and the ones that remain are the convention

**This section said the opposite on 2026-08-31, and it was wrong.** It read:
*"268 `JBIG2Decode mask` and 151 `(samples) mask` complements, the same as at
v0.20.0 — the stencil polarity convention, where it belongs. A chroma fix has no
reason to move them, and it did not."* The reasoning was sound and the
conclusion did not follow: a chroma fix had no reason to move them, and neither
side of that sentence could see that the PAIRING was inventing some of them.

Over all 23 populations, complements fall from **488 to 422**. Where they fell
is the whole finding:

| | 2026-08-31 | 2026-09-07 | |
|---|---:|---:|---|
| `fr-cerfa`, all filters | 83 | **15** | 68 were a black swatch paired with a white one |
| `JBIG2Decode mask` | 268 | 268 | unchanged — the convention |
| `(samples) mask` | 151 | 153 | **up two** |
| `ia-uscourts` | 37 | **39** | **up two** |

The two that went UP are the ones worth reading. A complement is only reported
when the direct comparison fails the gate and the complement passes it; a wrong
pairing can hide a real complement inside a disagreement just as easily as it
can invent one. Four did, in two populations, and the right pairing found them.

So the sentence that survives is narrower than the one it replaces: **the
complements on the mask filters are the stencil polarity convention, and the
ones a form corpus reported were mostly an artefact of how the two sides were
lined up.** The first half was always true. The second was not visible until
the instrument stopped guessing.

### 9. `DCTDecode` is still the only real gap, and it is smaller than it looked

With both the pairing and the walk fixed, every lossless path is at or near the
top: `JPXDecode` **100.0%**, `JBIG2Decode` 100.0%, `(samples)` **99.8%**, the
mask filters **98.0%** and 100.0%. `DCTDecode` sits at **48.9%**.

It has now moved twice, and the two moves are different in kind:

| | agreement | differing | unmatched |
|---|---:|---:|---:|
| 2026-08-31, v0.20.0 | 43.2% | — | — |
| pairing by object | **43.7%** | 240 | 61 |
| walking what is drawn | **48.9%** | **217** | **0** |

The first move was 0.5 points, because pairing by object was never `DCTDecode`'s
problem. The second is 5.2 points and it took **all 61** of its unpaired
pictures to zero — those were pictures no page drew. Twenty-three of its
disagreements went with them.

**217 remain, and they are the work.** They are paired by object number on both
sides, they are the pictures the pages actually draw, and they disagree. That is
now a question about a codec and nothing else, which is the first time in this
file's history it has been only that.

### 10. The instrument was wrong twice in one day, and both times it read as a defect somewhere else

Two changes landed between the previous run and this one, and neither is in a
decoder:

1. `render.Images` returned the pictures a page's `/Resources` **hold** rather
   than the ones it **draws** (`render#43`).
2. The size fallback claimed a judge's row even when both sides had published an
   object number and the two differed (`conformance#30`).

Each on its own produced a coherent, plausible, entirely fabricated result. The
one that started the investigation was a Data Matrix barcode reported **255
apart on every term** with an MSE of 11 906 — a number so large it reads as a
broken codec. The correct pair, measured afterwards, differs from poppler's
extraction by **165 pixels of 13 924 at a peak of 1**, which is the codec's
rounding. Go's own `image/jpeg`, which is what `render` calls, agrees with
poppler and with macOS ImageIO to **zero pixels** on it.

**What caught it was a count that would not reconcile, never a re-reading.** The
document's page 1 draws one 118×118 barcode; the walk returned ten. Nothing in
the code looked wrong, and the comment above it stated the correct rule in
words.

**And what hid it was that everything agreed.** The fixtures in both packages
built a page with pictures in its `/Resources` and an **empty content stream**,
while their own comments said the page « draws » them. The stand-in judge in
`conformance` wrote `object 7` on every listing row, and 7 is an object none of
those fixtures holds — so every end-to-end test there was pairing by size, and
the by-object path added a week earlier was never once exercised by them. A
fixture that agrees with nothing contradicts nothing.

The three tests that now stand against this were each confirmed to **fail**
without their change before being kept.

## Every differing bucket in the run

| population | filter | bucket | differing | share med | share worst | peak med | peak worst | mse med | mse worst | mean med | mean worst |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `fr-impots` | `DCTDecode` | direct | 1 | 0.000100 | 0.000100 | 3 | 3 | 0.3344 | 0.3344 | -0.2371 | -0.2371 |
| `ia-medical` | `DCTDecode` | direct | 1 | 0.000112 | 0.000112 | 4 | 4 | 0.1112 | 0.1112 | +0.0282 | +0.0282 |
| `ia-americana` | `DCTDecode` | converted | 3 | 0.000113 | 0.000396 | 4 | 4 | 0.1974 | 0.3278 | +0.1403 | +0.2795 |
| `gh-openpdf` | `DCTDecode` | direct | 5 | 0.000134 | 0.001350 | 3 | 4 | 0.1450 | 0.2069 | +0.0591 | -0.1269 |
| `ia-americana` | `DCTDecode` | direct | 26 | 0.000136 | 0.000949 | 4 | 4 | 0.2704 | 0.4512 | +0.1723 | +0.3768 |
| `us-irs` | `DCTDecode` | direct | 1 | 0.000140 | 0.000140 | 3 | 3 | 0.1805 | 0.1805 | -0.0392 | -0.0392 |
| `gh-pdfbox` | `DCTDecode` | converted | 1 | 0.000156 | 0.000156 | 3 | 3 | 0.2781 | 0.2781 | +0.2348 | +0.2348 |
| `fr-cerfa` | `DCTDecode` | direct | 112 | 0.000160 | 0.001926 | 3 | 4 | 0.1308 | 0.6541 | +0.0355 | -0.5794 |
| `ia-uscourts` | `DCTDecode` | direct | 8 | 0.000177 | 0.000359 | 3 | 4 | 0.0654 | 0.1885 | -0.0070 | -0.0783 |
| `gh-pdfcpu` | `DCTDecode` | converted | 1 | 0.000195 | 0.000195 | 3 | 3 | 0.2439 | 0.2439 | -0.0339 | -0.0339 |
| `gh-qpdf` | `DCTDecode` | direct | 3 | 0.000319 | 0.000319 | 3 | 3 | 0.3483 | 0.3822 | +0.2801 | +0.3099 |
| `gh-pdfcpu` | `DCTDecode` | direct | 40 | 0.000349 | 0.000923 | 3 | 4 | 0.2554 | 0.3582 | +0.0015 | -0.2789 |
| `fr-cerfa` | `DCTDecode` | converted | 11 | 0.000498 | 0.742372 | 4 | 43 | 0.3225 | 118.4345 | +0.1899 | +3.1964 |
| `gh-pdfbox` | `DCTDecode` | direct | 2 | 0.000508 | 0.000508 | 4 | 4 | 0.2605 | 0.2605 | +0.0024 | -0.1511 |
| `gh-pypdf` | `DCTDecode` | direct | 2 | 0.000517 | 0.000517 | 3 | 3 | 0.2581 | 0.2581 | -0.0561 | -0.0989 |
| `gh-safedocs` | `DCTDecode` | direct | 1 | 0.000595 | 0.000595 | 4 | 4 | 0.2810 | 0.2810 | -0.1230 | -0.1230 |
| `us-dol` | `DCTDecode` | direct | 15 | 0.001119 | 0.001339 | 3 | 4 | 0.3029 | 0.3190 | -0.2787 | -0.2882 |
| `ia-uscourts` | `(samples)` | converted | 3 | 0.043906 | 1.000000 | 97 | 101 | 25.1806 | 1089.9152 | +0.2460 | -28.9192 |
| `fr-cerfa` | `(samples) mask` | direct | 22 | 0.072224 | 0.217945 | 255 | 255 | 4696.3384 | 14171.9016 | -0.0843 | -50.0687 |
| `us-opm` | `(samples)` | converted | 1 | 0.100000 | 0.100000 | 9 | 9 | 4.6706 | 4.6706 | +0.6183 | +0.6183 |
| `ia-uscourts` | `DCTDecode` | converted | 2 | 0.138858 | 0.138858 | 105 | 105 | 95.0602 | 95.0602 | +0.9965 | +0.9965 |
| `gh-qpdf` | `(samples)` | direct | 2 | 0.222222 | 0.222222 | 32 | 32 | 227.5556 | 227.5556 | +0.0000 | +0.0000 |
| `uk-govuk` | `(samples)` | converted | 1 | 0.273188 | 0.273188 | 35 | 35 | 292.3110 | 292.3110 | -8.9241 | -8.9241 |
| `us-uscis` | `(samples)` | converted | 1 | 0.466357 | 0.466357 | 35 | 35 | 365.4217 | 365.4217 | -12.2960 | -12.2960 |
| `fr-impots` | `(samples)` | converted | 10 | 0.489155 | 0.499713 | 164 | 170 | 5784.5370 | 6193.2730 | +42.5565 | +44.5049 |
| `ia-medical` | `(samples)` | converted | 1 | 0.628462 | 0.628462 | 19 | 19 | 121.8514 | 121.8514 | -8.3359 | -8.3359 |
| `gh-openpdf` | `DCTDecode` | converted | 2 | 0.939036 | 0.939036 | 110 | 110 | 909.3403 | 909.3403 | +9.2931 | +9.2931 |
| `gh-pypdf` | `(samples)` | converted | 1 | 0.985783 | 0.985783 | 27 | 27 | 121.3016 | 121.3016 | -4.2622 | -4.2622 |
| `uk-govuk` | `DCTDecode` | converted | 1 | 0.994621 | 0.994621 | 61 | 61 | 115.5974 | 115.5974 | -2.1431 | -2.1431 |
| `fr-cerfa` | `(samples)` | converted | 145 | 1.000000 | 1.000000 | 63 | 241 | 3969.0000 | 42752.3333 | -63.0000 | -205.6667 |
| `fr-impots` | `DCTDecode` | converted | 3 | 1.000000 | 1.000000 | 255 | 255 | 22819.4133 | 22891.5689 | -115.6788 | -115.6788 |
| `gh-pdfbox` | `(samples)` | converted | 1 | 1.000000 | 1.000000 | 255 | 255 | 20952.5000 | 20952.5000 | +25.5000 | +25.5000 |
| `gh-pdfbox` | `JPXDecode` | converted | 1 | 1.000000 | 1.000000 | 255 | 255 | 42179.8812 | 42179.8812 | -170.0170 | -170.0170 |

## What is not measured, and why

- **The record does not carry what the colour-space rule moved.** `calibrated`
  counts CIE-tagged pictures in a bucket, not the ones the rule put there, and
  the five in finding 3 are a **difference between two runs**. The next run has
  no run before it that lacks the rule, so it cannot repeat that derivation;
  what it will have is this document.
- **`Missing.Ours` still folds two different things together** — a document we
  cannot read and a page we decline to decode. Finding 6 shows all four of this
  run's refusals are the second kind. The counts are correct; the label is too
  coarse.
- **The three `DCTDecode` rows that did not move are not diagnosed.**
  `fr-impots` at a median peak of 255, `gh-pypdf` at 233 and `gh-qpdf` at 171 are
  not chroma reconstruction and are not rounding.
  [conformance#13](https://github.com/go-pdfkit/conformance/issues/13) shows
  `match` manufactures disagreements when a page draws many pictures of one
  size, and no per-picture pairing audit was run for this baseline.
- **No aggregate bound is applied.** `mse` and `mean` are recorded in FFmpeg's
  and pdfium's units and bounded by nothing, because no bound has been measured
  for pictures that were *extracted* rather than rendered. Choosing one from
  these records is a job for a later run, and the records carry the terms.
- **The bound on the judge is unexercised by `images` on this corpus.** Finding
  5. It has been shown to fire, through `compare`, on the one document known to
  hang.
- **One page per document.** A first page is not a document.
