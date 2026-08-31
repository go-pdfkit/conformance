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
| taken | 2026-08-31T14:04:51Z .. 2026-08-31T15:21:44Z (UTC) |
| judge | pdfimages version 26.04.0 |
| **measure** | **per channel, gate `D` = 2, count budget `N` = 0** ([conformance#16](https://github.com/go-pdfkit/conformance/issues/16)) |
| **bucketing** | the listing **and** the picture's own `/ColorSpace` ([conformance#20](https://github.com/go-pdfkit/conformance/issues/20)) |
| **bound on the judge** | **2m0s per document, per tool** ([conformance#21](https://github.com/go-pdfkit/conformance/issues/21)) |
| `go-pdfkit/render` | **v0.21.0** |
| `go-pdfkit/reader` | v0.6.0 |
| `go-gfx/gfx` | v0.19.0 |
| `tannevaled/gobig2` | v0.1.0 |
| `ajroetker/go-jpeg2000` | v0.0.2 |
| pages per document | 1 (the first page of each document) |
| corpora | `/Users/Shared/pdfscans` (MANIFEST.tsv), `/Users/Shared/pdfforms` (MANIFEST.tsv) |

**Every one of the 23 populations ran to completion, and every one is in the
tables below.** All 23 exited 0.

**Why this run exists.** `render` v0.21.0 changes the decoded output of *every
subsampled JPEG in the fleet*: Go's `image/jpeg` replicates a subsampled chroma
sample where libjpeg interpolates, and v0.21.0 reproduces libjpeg's filter. The
records this file described before were taken at v0.20.0, so they described a
library that no longer exists. `go-gfx/gfx` also moved from v0.16.0 to v0.19.0
in between. **Every figure here was taken at one version of everything**, which
is what the `modules` block in each record is for; the previous run's figures
are quoted below only where they are named as the previous run's.

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
oversight. Peak memory is reported, because it is not a timing: `ia-americana`
5.9 GB, `ia-biodiversity` 3.6 GB, `ia-medical` 3.2 GB, everything else far
below.

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
- **unmatched** — theirs had no picture of that size to pair with.
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
| `ia-biodiversity` | 250 | 0 | 4 | 0 | 0 | 781 | 694 | 50 | 644 | 634 | 157 | 98.4% | 0 | 0 |
| `ia-americana` | 250 | 28 | 0 | 0 | 0 | 505 | 494 | 89 | 405 | 379 | 96 | 93.6% | 8 | 8 |
| `ia-texts` | 12 | 7 | 0 | 0 | 0 | 14 | 14 | 3 | 11 | 11 | 2 | 100.0% | 0 | 0 |
| `ia-uscourts` | 250 | 0 | 0 | 0 | 0 | 134 | 114 | 37 | 77 | 66 | 54 | 85.7% | 17 | 2 |

Government and library forms — `/Users/Shared/pdfforms`:

| population | documents | unopenable | refused | declined | hung | pictures | direct | inverted | compared | exact | identical | agreement | converted | calibrated |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ca-cra` | 84 | 0 | 0 | 0 | 0 | 151 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `fr-cerfa` | 450 | 0 | 0 | 0 | 0 | 4380 | 1338 | 15 | 1323 | 936 | 878 | 70.7% | 2978 | 1741 |
| `fr-impots` | 50 | 0 | 0 | 0 | 0 | 138 | 20 | 0 | 20 | 16 | 6 | 80.0% | 25 | 3 |
| `gh-openpdf` | 56 | 14 | 0 | 0 | 0 | 88 | 27 | 0 | 27 | 21 | 6 | 77.8% | 2 | 1 |
| `gh-pdfbox` | 157 | 8 | 0 | 1 | 0 | 41 | 29 | 1 | 28 | 17 | 14 | 60.7% | 11 | 1 |
| `gh-pdfcpu` | 147 | 0 | 0 | 0 | 0 | 696 | 639 | 0 | 639 | 599 | 597 | 93.7% | 57 | 32 |
| `gh-pypdf` | 34 | 1 | 0 | 0 | 0 | 17 | 8 | 0 | 8 | 6 | 6 | 75.0% | 6 | 3 |
| `gh-qpdf` | 81 | 0 | 0 | 0 | 0 | 85 | 73 | 0 | 73 | 16 | 16 | 21.9% | 0 | 0 |
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
| `DCTDecode` | 652 | 426 | 0 | 426 | 184 | 4 | **43.2%** | 44 | 14 | 29 | 32 | 121 | 61 | 242 | 255 |
| `(samples)` | 4292 | 915 | 0 | 915 | 638 | 638 | **69.7%** | 3084 | 1203 | 1813 | 1761 | 286 | 7 | 277 | 255 |
| `(samples) mask` | 1357 | 1268 | 151 | 1117 | 1074 | 1074 | **96.2%** | 1 | 0 | 1 | 0 | 87 | 1 | 43 | 255 |
| `JPXDecode` | 1264 | 1225 | 0 | 1225 | 1215 | 7 | **99.2%** | 10 | 9 | 1 | 9 | 0 | 29 | 10 | 255 |
| `DCTDecode mask` | 12 | 12 | 0 | 12 | 12 | 5 | **100.0%** | 0 | 0 | 0 | 0 | 0 | 0 | 0 | — |
| `JBIG2Decode` | 11 | 10 | 0 | 10 | 10 | 10 | **100.0%** | 0 | 0 | 0 | 0 | 1 | 0 | 0 | — |
| `JBIG2Decode mask` | 600 | 518 | 268 | 250 | 250 | 250 | **100.0%** | 0 | 0 | 0 | 0 | 70 | 12 | 0 | — |
| `JPXDecode mask` | 2 | 2 | 0 | 2 | 2 | 2 | **100.0%** | 0 | 0 | 0 | 0 | 0 | 0 | 0 | — |

Across the whole fleet: **3385 of 3957 direct comparable pictures agree, 85.5%**,
and 1990 of them are bit-identical. At v0.20.0 it was 3347 of 3962, **84.5%**,
with the same 1990 identical.

## What this says

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

So the improvement is real and it is large, and it is a **magnitude** result
rather than a rate result at this gate. Three rows did not move at all —
`fr-impots` at 255, `gh-pypdf` at 233, `gh-qpdf` at 171 — and those are not
chroma reconstruction. They are the remaining `DCTDecode` finding, and this run
does not diagnose them.

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
direct/converted split is the rule and nothing else**:

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
exactly:

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
count of 1990, are unchanged to the picture. **`JPXDecode` stays at 99.2% with 7
of 1225 bit-equal** — a conformant lossy decoder, agreeing within two levels
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

### 6. The four refusals are the same four, and still our own budget

`refused` is **4** across 3280 documents, all in `ia-biodiversity`, and they are
the same four documents as at v0.20.0: `bulletinno38tasm.pdf` (9449 × 13701),
`checklistofbirds00wood.pdf` (2868 × 4780), `informeacercade00soci.pdf`
(2790 × 3737) and `bulletindelasoci4243soci.pdf` (794 × 1372). Each opens and
each page 1 resolves; what fails is `render.Images` with
`render.ErrTooMuchToDecode`, the 256-megapixel budget declining a page. The last
is the telling one: an ordinary 794 × 1372 picture refused because the page had
already spent all but 223 089 of its 268 435 456 pixels.

**`Missing.Ours` still folds "we could not read this" together with "we chose not
to decode this", and only the first is a defect.** The instrument should tell
them apart and does not; the four are named here so nobody reads the 4 as a
coverage gap.

`unopenable` is 63 and `declined` is 1 across the fleet, and neither is a defect.

### 7. The gate still buys the lossless filters nothing

| filter | compared | exact | identical |
|---|---:|---:|---:|
| `JBIG2Decode` | 10 | 10 | **10** |
| `JBIG2Decode mask` | 250 | 250 | **250** |
| `JPXDecode mask` | 2 | 2 | **2** |
| `(samples)` | 915 | 638 | **638** |
| `(samples) mask` | 1117 | 1074 | **1074** |

**Every agreeing picture of every lossless filter is bit-equal**, as at v0.20.0.
Not one needed the gate, so the uniform `D = 2` costs the filters that could
defensibly be held to 0 nothing, and there is still no measured case for a
per-filter exception table.

### 8. Inversions are unchanged, and still conventions

268 `JBIG2Decode mask` and 151 `(samples) mask` complements, the same as at
v0.20.0 — the stencil polarity convention, where it belongs. A chroma fix has no
reason to move them, and it did not.

## Every differing bucket in the run

| population | filter | bucket | differing | share med | share worst | peak med | peak worst | mse med | mse worst | mean med | mean worst |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ia-medical` | `DCTDecode` | direct | 1 | 0.000112 | 0.000112 | 4 | 4 | 0.1112 | 0.1112 | +0.0282 | +0.0282 |
| `ia-americana` | `DCTDecode` | converted | 3 | 0.000113 | 0.000396 | 4 | 4 | 0.1974 | 0.3278 | +0.1403 | +0.2795 |
| `ia-americana` | `DCTDecode` | direct | 26 | 0.000136 | 0.000949 | 4 | 4 | 0.2704 | 0.4512 | +0.1723 | +0.3768 |
| `us-irs` | `DCTDecode` | direct | 1 | 0.000140 | 0.000140 | 3 | 3 | 0.1805 | 0.1805 | -0.0392 | -0.0392 |
| `gh-pdfbox` | `DCTDecode` | converted | 1 | 0.000156 | 0.000156 | 3 | 3 | 0.2781 | 0.2781 | +0.2348 | +0.2348 |
| `fr-cerfa` | `DCTDecode` | direct | 114 | 0.000163 | 0.062341 | 3 | 15 | 0.1312 | 2.4179 | +0.0355 | -0.5794 |
| `ia-uscourts` | `DCTDecode` | direct | 8 | 0.000177 | 0.000359 | 3 | 4 | 0.0654 | 0.1885 | -0.0070 | -0.0783 |
| `gh-pdfcpu` | `DCTDecode` | converted | 1 | 0.000195 | 0.000195 | 3 | 3 | 0.2439 | 0.2439 | -0.0339 | -0.0339 |
| `gh-pdfcpu` | `DCTDecode` | direct | 40 | 0.000349 | 0.000923 | 3 | 4 | 0.2554 | 0.3582 | +0.0015 | -0.2789 |
| `gh-openpdf` | `DCTDecode` | direct | 6 | 0.000386 | 0.997683 | 3 | 255 | 0.1645 | 9934.6949 | +0.0887 | +53.3166 |
| `gh-pdfbox` | `DCTDecode` | direct | 2 | 0.000508 | 0.000508 | 4 | 4 | 0.2605 | 0.2605 | +0.0024 | -0.1511 |
| `gh-safedocs` | `DCTDecode` | direct | 1 | 0.000595 | 0.000595 | 4 | 4 | 0.2810 | 0.2810 | -0.1230 | -0.1230 |
| `us-dol` | `DCTDecode` | direct | 15 | 0.001119 | 0.001339 | 3 | 4 | 0.3029 | 0.3190 | -0.2787 | -0.2882 |
| `ia-uscourts` | `(samples)` | converted | 3 | 0.043906 | 1.000000 | 97 | 101 | 25.1806 | 1089.9152 | +0.2460 | -28.9192 |
| `fr-cerfa` | `(samples) mask` | direct | 41 | 0.072224 | 0.248274 | 255 | 255 | 4696.3384 | 16144.0274 | -0.0843 | +50.0687 |
| `us-opm` | `(samples)` | converted | 1 | 0.100000 | 0.100000 | 9 | 9 | 4.6706 | 4.6706 | +0.6183 | +0.6183 |
| `ia-uscourts` | `DCTDecode` | converted | 2 | 0.138858 | 0.138858 | 105 | 105 | 95.0602 | 95.0602 | +0.9965 | +0.9965 |
| `gh-qpdf` | `(samples)` | direct | 35 | 0.222222 | 0.222222 | 32 | 32 | 227.5556 | 227.5556 | +0.0000 | +0.0000 |
| `fr-impots` | `DCTDecode` | direct | 4 | 0.257613 | 0.274275 | 255 | 255 | 12004.3773 | 12011.1328 | -0.2371 | -15.1967 |
| `uk-govuk` | `(samples)` | converted | 1 | 0.273188 | 0.273188 | 35 | 35 | 292.3110 | 292.3110 | -8.9241 | -8.9241 |
| `fr-cerfa` | `DCTDecode` | converted | 16 | 0.304263 | 1.000000 | 11 | 255 | 5.7358 | 31435.6250 | +0.2624 | +57.2083 |
| `gh-qpdf` | `DCTDecode` | direct | 22 | 0.412628 | 0.466199 | 171 | 171 | 2595.6229 | 3100.8935 | +1.6654 | +8.2570 |
| `fr-impots` | `(samples)` | converted | 13 | 0.455042 | 0.499713 | 170 | 255 | 5784.5370 | 12765.8108 | +3.4954 | +44.5049 |
| `us-uscis` | `(samples)` | converted | 1 | 0.466357 | 0.466357 | 35 | 35 | 365.4217 | 365.4217 | -12.2960 | -12.2960 |
| `fr-cerfa` | `(samples)` | converted | 1788 | 0.500000 | 1.000000 | 19 | 241 | 73.0000 | 36864.0000 | +0.3333 | +192.0000 |
| `ia-medical` | `(samples)` | converted | 1 | 0.628462 | 0.628462 | 19 | 19 | 121.8514 | 121.8514 | -8.3359 | -8.3359 |
| `fr-cerfa` | `(samples)` | direct | 232 | 0.888889 | 1.000000 | 115 | 255 | 1504.8095 | 8323.0833 | +0.4167 | +53.5556 |
| `gh-pypdf` | `DCTDecode` | direct | 2 | 0.894813 | 0.894813 | 233 | 233 | 1373.1803 | 1373.1803 | +0.3972 | +0.3972 |
| `gh-openpdf` | `DCTDecode` | converted | 2 | 0.939036 | 0.939036 | 110 | 110 | 909.3403 | 909.3403 | +9.2931 | +9.2931 |
| `ia-uscourts` | `(samples) mask` | direct | 2 | 0.958333 | 0.958333 | 255 | 255 | 62315.6250 | 62315.6250 | -240.3906 | -240.3906 |
| `gh-pypdf` | `(samples)` | converted | 1 | 0.985783 | 0.985783 | 27 | 27 | 121.3016 | 121.3016 | -4.2622 | -4.2622 |
| `ia-biodiversity` | `JPXDecode` | direct | 10 | 0.994521 | 0.999976 | 255 | 255 | 17813.8994 | 32873.8577 | +34.2609 | -172.2092 |
| `uk-govuk` | `DCTDecode` | converted | 1 | 0.994621 | 0.994621 | 61 | 61 | 115.5974 | 115.5974 | -2.1431 | -2.1431 |
| `fr-impots` | `DCTDecode` | converted | 3 | 1.000000 | 1.000000 | 255 | 255 | 22819.4133 | 22891.5689 | -115.6788 | -115.6788 |
| `gh-pdfbox` | `(samples)` | direct | 9 | 1.000000 | 1.000000 | 255 | 255 | 14734.9375 | 27229.9219 | -0.1458 | -62.3750 |
| `gh-pdfbox` | `(samples)` | converted | 4 | 1.000000 | 1.000000 | 255 | 255 | 27389.5091 | 36125.0000 | +12.7500 | +63.8958 |
| `gh-pdfbox` | `JPXDecode` | converted | 1 | 1.000000 | 1.000000 | 255 | 255 | 42179.8812 | 42179.8812 | -170.0170 | -170.0170 |
| `ia-uscourts` | `(samples) mask` | converted | 1 | 1.000000 | 1.000000 | 207 | 207 | 16375.4434 | 16375.4434 | -98.1878 | -98.1878 |
| `ia-uscourts` | `(samples)` | direct | 1 | 1.000000 | 1.000000 | 207 | 207 | 16181.4161 | 16181.4161 | +96.8730 | +96.8730 |

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
