# What go-pdfkit comes to today

This is a baseline: the fidelity of `go-pdfkit` against poppler, per population
and per image filter, written down so that tomorrow's regression can be seen.
It is not an argument that anything is right. It is the number that a later run
is subtracted from.

The records beside this file are what `images -json` wrote, unedited. Each one
carries its own conditions — the corpus, the judge's version, the **gate** the
comparison used, every module version it was built against, and when it was
taken — because a figure that falls between two runs means a regression **only
if everything else held**, and a drop is otherwise as likely to be a newer
poppler, a corpus that grew, or a different instrument.

## Conditions

| | |
|---|---|
| taken | 2026-08-31T09:43:00Z .. 2026-08-31T12:13:14Z (UTC) |
| judge | pdfimages version 26.04.0 |
| **measure** | **per channel, gate `D` = 2, count budget `N` = 0** ([conformance#16](https://github.com/go-pdfkit/conformance/issues/16)) |
| `go-pdfkit/render` | **v0.20.0** |
| `go-pdfkit/reader` | v0.6.0 |
| `go-gfx/gfx` | v0.16.0 |
| `tannevaled/gobig2` | v0.1.0 |
| `ajroetker/go-jpeg2000` | v0.0.2 |
| pages per document | 1 (the first page of each document) |
| corpora | `/Users/Shared/pdfscans` (MANIFEST.tsv), `/Users/Shared/pdfforms` (MANIFEST.tsv) |

**What `exact` asserts here.** A picture agrees when **no channel of any pixel
differs from poppler's by more than two levels of 255**. Two is the ISO/IEC
10918-2 IDCT allowance either side of the reference, read out of
`libavcodec/tests/dct.c:259`; the derivation and the survey it sits in are in
[the repository README](../README.md). Comparing at all is defensible here and
would not be for a page: `pdfimages` **extracts** rather than renders, so there
is no rasteriser between the codec and the pixels. `compare`, which draws whole
pages, cannot use this criterion and does not.

**Every figure in this document was taken with that instrument.** The records
this file described before were taken with a different one — a bisection at
luminance 128, where a picture agreed when no pixel's *ink classification*
differed, which was exact for a bilevel picture and blind to a level or chroma
shift everywhere else. **The two cannot be subtracted from one another**, and
no table here mixes them. Every population was re-run, so **no record under `baseline/` is the old instrument any more** and nothing in this document is inherited.

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
- **remapped** — the picture carries a `/Decode` array, which a viewer applies
  and `pdfimages` does not. The two sides were not asked the same question.
- **unmatched** — theirs had no picture of that size to pair with.
- **converted** — poppler had to convert the picture's colour space to reach
  RGB (`cmyk`, `lab`, `icc`, `index`, `sep`, `devn`, or a row that could not be
  read). Per channel that arithmetic is large and is **not** a decoder
  disagreeing, so those pictures are tallied in their own bucket with their own
  agreement figure and their own magnitudes. `index` counts as converted
  because `pdfimages` does not report its base space.
- **inverted** — ours and theirs are exact complements within the gate.
  `pdfimages` writes a **stencil** — `/ImageMask true`, or a stream named as a
  picture's `/Mask` — with the opposite polarity to its samples, whatever
  filter it arrived in; that is a convention, not a disagreement.
- **agreement** — exact ÷ **direct comparable**, where direct comparable is the
  direct bucket's pictures minus its complements. It is never computed over the
  converted bucket. A filter with nothing comparable reads `n/a`, not 0%.

Beside the counts, each bucket that had anything differ carries the **median
and the far end of four magnitudes** over its differing pictures:

- **`peak`** — the largest channel difference in the picture, in levels of 255.
  This is the criterion; everything else is a report.
- **`share`** — the fraction of pixels where some channel exceeded the gate.
- **`mse`** — mean squared error over the compared channels, in levels squared.
- **`mean`** — the **signed** mean error, ours minus theirs, in levels. The
  far end of this one is the value furthest from zero *with its sign*, so a
  bias says which way it ran.

A population that was **not run** says so in the table, by name. It is never
omitted, because a reader who cannot tell a population that scored badly from
one that never ran has a report that misleads in the direction of comfort.

## The fleet, per population

Scanned pages — `/Users/Shared/pdfscans`:

| population | documents | unopenable | refused | declined | pictures | direct | inverted | compared | exact | identical | agreement | converted |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ia-medical` | 250 | 0 | 0 | 0 | 745 | 722 | 193 | 529 | 528 | 33 | 99.8% | 1 |
| `ia-biodiversity` | 250 | 0 | 4 | 0 | 781 | 694 | 50 | 644 | 634 | 157 | 98.4% | 0 |
| `ia-americana` | 250 | 28 | 0 | 0 | 505 | 494 | 89 | 405 | 379 | 96 | 93.6% | 8 |
| `ia-texts` | 12 | 7 | 0 | 0 | 14 | 14 | 3 | 11 | 11 | 2 | 100.0% | 0 |
| `ia-uscourts` | 250 | 0 | 0 | 0 | 134 | 115 | 37 | 78 | 63 | 54 | 80.8% | 16 |

Government and library forms — `/Users/Shared/pdfforms`:

| population | documents | unopenable | refused | declined | pictures | direct | inverted | compared | exact | identical | agreement | converted |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ca-cra` | 84 | 0 | 0 | 0 | 151 | 0 | 0 | 0 | 0 | 0 | n/a | 0 |
| `fr-cerfa` | 450 | 0 | 0 | 0 | 4380 | 1341 | 15 | 1326 | 914 | 878 | 68.9% | 2975 |
| `fr-impots` | 50 | 0 | 0 | 0 | 138 | 20 | 0 | 20 | 14 | 6 | 70.0% | 25 |
| `gh-openpdf` | 56 | 14 | 0 | 0 | 88 | 28 | 0 | 28 | 21 | 6 | 75.0% | 1 |
| `gh-pdfbox` | 157 | 8 | 0 | 1 | 41 | 29 | 1 | 28 | 17 | 14 | 60.7% | 11 |
| `gh-pdfcpu` | 147 | 0 | 0 | 0 | 696 | 639 | 0 | 639 | 599 | 597 | 93.7% | 57 |
| `gh-pypdf` | 34 | 1 | 0 | 0 | 17 | 8 | 0 | 8 | 6 | 6 | 75.0% | 6 |
| `gh-qpdf` | 81 | 0 | 0 | 0 | 85 | 73 | 0 | 73 | 16 | 16 | 21.9% | 0 |
| `gh-safedocs` | 26 | 5 | 0 | 0 | 2 | 2 | 0 | 2 | 1 | 1 | 50.0% | 0 |
| `gh-verapdf` | 134 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 |
| `int-wipo` | 116 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 |
| `uk-govuk` | 302 | 0 | 0 | 0 | 225 | 143 | 1 | 142 | 131 | 111 | 92.3% | 31 |
| `us-dol` | 140 | 0 | 0 | 0 | 46 | 25 | 0 | 25 | 10 | 10 | 40.0% | 0 |
| `us-irs` | 69 | 0 | 0 | 0 | 8 | 1 | 0 | 1 | 0 | 0 | 0.0% | 0 |
| `us-opm` | 66 | 0 | 0 | 0 | 37 | 31 | 28 | 3 | 3 | 3 | 100.0% | 2 |
| `us-ssa` | 199 | 0 | 0 | 0 | 8 | 0 | 0 | 0 | 0 | 0 | n/a | 0 |
| `us-uscis` | 88 | 0 | 0 | 0 | 87 | 0 | 0 | 0 | 0 | 0 | n/a | 1 |
| `us-uscourts` | 69 | 0 | 0 | 0 | 2 | 2 | 2 | 0 | 0 | 0 | n/a | 0 |

**Every one of the 23 populations ran to completion.** Three of them never had
a record before, and what happened to each is the subject of finding 3.

## The fleet, per filter

**`compared` is the denominator every agreement figure is computed over** — the
direct bucket's pictures minus its complements. It is printed beside `pictures`
because the two are far apart and only one of them is the claim, and beside
`identical`, because a rate is not a claim of bit equality.

| filter | pictures | direct | inverted | **compared** | exact | identical | agreement | converted | conv. exact | conv. differing | remapped | unmatched | differing | worst peak |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `DCTDecode` | 652 | 430 | 0 | 430 | 146 | 4 | **34.0%** | 40 | 6 | 33 | 121 | 61 | 284 | 255 |
| `(samples)` | 4292 | 916 | 0 | 916 | 638 | 638 | **69.7%** | 3083 | 1203 | 1812 | 286 | 7 | 278 | 255 |
| `(samples) mask` | 1357 | 1268 | 151 | 1117 | 1074 | 1074 | **96.2%** | 1 | 0 | 1 | 87 | 1 | 43 | 255 |
| `JPXDecode` | 1264 | 1225 | 0 | 1225 | 1215 | 7 | **99.2%** | 10 | 9 | 1 | 0 | 29 | 10 | 255 |
| `DCTDecode mask` | 12 | 12 | 0 | 12 | 12 | 5 | **100.0%** | 0 | 0 | 0 | 0 | 0 | 0 | — |
| `JBIG2Decode` | 11 | 10 | 0 | 10 | 10 | 10 | **100.0%** | 0 | 0 | 0 | 1 | 0 | 0 | — |
| `JPXDecode mask` | 2 | 2 | 0 | 2 | 2 | 2 | **100.0%** | 0 | 0 | 0 | 0 | 0 | 0 | — |
| `JBIG2Decode mask` | 600 | 518 | 268 | 250 | 250 | 250 | **100.0%** | 0 | 0 | 0 | 70 | 12 | 0 | — |

## What this says

### 1. JPEG 2000 is conformant and almost never bit-equal; JPEG is neither

This is the finding the instrument was built to make, and the old one could not
have made it. `JPXDecode` and `DCTDecode` used to read **15.4%** and **33.2%**
and looked like two versions of the same problem. Per channel they are not the
same problem at all:

| filter | compared | exact | **identical** | agreement |
|---|---:|---:|---:|---:|
| `JPXDecode` | 1225 | 1215 | **7** | **99.2%** |
| `DCTDecode` | 430 | 146 | **4** | **34.0%** |

**JPEG 2000 agrees with poppler within two levels on 99.2% of the pictures
compared and is bit-equal on 7 of 1225.** That is precisely what a conformant
lossy decoder looks like, and it is what ISO/IEC 15444 promises: the transform
is specified, the rounding is not. `ia-medical` alone is 496 pictures, 496
exact, **1 identical**. The bisection's 15.4% was measuring how often our
rounding and poppler's fell on opposite sides of luminance 128.

**`DCTDecode` did not move**: 33.2% to 34.0%. The gate rescued JPEG 2000 and
did not rescue JPEG, which means our JPEG decoder differs from poppler's by
**more than the ISO/IEC 10918-2 IDCT allowance** on two thirds of what it was
compared on. And the peaks say by how much — the per-population medians of the
worst channel run 16, 18, 23, 23, 26, 31, 34, 47, 50, 58, 62, 171, 244, 255.
Sixteen to sixty-two levels is not rounding and it is not a wholly wrong
picture either; it is a systematic moderate error, which is the exact shape the
bisection could not see. **This is the first specific, actionable decode
finding this repository has produced**, and it is not diagnosed here.

### 2. The empty band that produced the 1% does not survive, and what sits under 1% now is worse than what used to

The withdrawn tolerance was read off eighteen population × lossy-filter rows
whose medians fell into fourteen from 0.000011 to 0.001966 and four from
0.047161 to 0.898821 — **a factor of 24 of empty band**, with 1% inside it.
The 21 direct-bucket rows of this run, ordered by median share, are:

```
0.005553 0.007024 0.047733 0.069908 0.070221 0.072224 0.080903 0.083304
0.173316 0.222222 0.256679 0.315150 0.369112 0.369672 0.412628 0.888889
0.895425 0.958333 0.994521 1.000000 1.000000
```

**The band is gone as a band.** The largest gap anywhere in that list is a
factor of **6.8** (0.007024 to 0.047733), then 2.15 and 2.08; the rest is a
continuum. Where fourteen rows of eighteen used to sit below 1%, **two of
twenty-one do.**

And the two that do are the reason a share threshold must not come back. Their
**peak medians are 18 and 23 levels** — nine and eleven times the gate. A 1%
budget would pass `uk-govuk`'s `DCTDecode` row, where half a percent of the
pixels are wrong by up to 35 levels, and `ia-uscourts`'s, where 0.7% are wrong
by up to 81. Under the bisection the low group was rounding *by construction*,
because a pixel one level from the threshold could flip it. Under a magnitude
measure the low group is **sparse gross error**, which is Cairo's objection in
our own data: *"otherwise some problems could be masked"*.

So the reasoning that produced the 1% was measuring the bisection's
interaction with this corpus's tone distribution, exactly as
[the README](../README.md) suspected, and a number read off it would now buy
the opposite of what it was meant to buy. **`N` stays at 0.**

Every differing bucket in the run, for anyone who wants to check that:

| population | filter | bucket | differing | share med | share worst | peak med | peak worst | mse med | mse worst | mean med | mean worst |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ia-americana` | `DCTDecode` | converted | 3 | 0.000113 | 0.241657 | 4 | 37 | 0.3278 | 11.3543 | +0.1403 | +0.2795 |
| `uk-govuk` | `DCTDecode` | direct | 11 | 0.005553 | 0.014278 | 18 | 35 | 0.1088 | 0.5260 | -0.0023 | -0.0031 |
| `ia-uscourts` | `DCTDecode` | direct | 11 | 0.007024 | 0.227292 | 23 | 81 | 0.2898 | 4.0804 | -0.0001 | -0.0783 |
| `ia-americana` | `DCTDecode` | direct | 26 | 0.047733 | 0.334129 | 23 | 50 | 0.8419 | 11.0975 | +0.1718 | +0.3763 |
| `gh-pdfcpu` | `DCTDecode` | direct | 40 | 0.069908 | 0.560845 | 26 | 127 | 1.2818 | 35.6550 | +0.0018 | -0.2772 |
| `ia-medical` | `DCTDecode` | direct | 1 | 0.070221 | 0.070221 | 62 | 62 | 1.3258 | 1.3258 | +0.0215 | +0.0215 |
| `fr-cerfa` | `(samples) mask` | direct | 41 | 0.072224 | 0.248274 | 255 | 255 | 4696.3384 | 16144.0274 | -0.0843 | +50.0687 |
| `fr-cerfa` | `DCTDecode` | direct | 139 | 0.080903 | 0.586986 | 31 | 76 | 2.1730 | 31.7507 | +0.0124 | +0.8845 |
| `gh-safedocs` | `DCTDecode` | direct | 1 | 0.083304 | 0.083304 | 16 | 16 | 1.2593 | 1.2593 | -0.1195 | -0.1195 |
| `us-opm` | `(samples)` | converted | 1 | 0.100000 | 0.100000 | 9 | 9 | 4.6706 | 4.6706 | +0.6183 | +0.6183 |
| `ia-uscourts` | `DCTDecode` | converted | 2 | 0.138858 | 0.138858 | 105 | 105 | 95.0602 | 95.0602 | +0.9965 | +0.9965 |
| `gh-pdfbox` | `DCTDecode` | converted | 1 | 0.172295 | 0.172295 | 90 | 90 | 6.6146 | 6.6146 | +0.2183 | +0.2183 |
| `us-irs` | `DCTDecode` | direct | 1 | 0.173316 | 0.173316 | 47 | 47 | 4.0918 | 4.0918 | -0.0621 | -0.0621 |
| `gh-qpdf` | `(samples)` | direct | 35 | 0.222222 | 0.222222 | 32 | 32 | 227.5556 | 227.5556 | +0.0000 | +0.0000 |
| `fr-impots` | `DCTDecode` | direct | 6 | 0.256679 | 0.274275 | 255 | 255 | 11906.8161 | 12011.1328 | -0.2394 | -15.1967 |
| `gh-pdfcpu` | `DCTDecode` | converted | 1 | 0.266979 | 0.266979 | 27 | 27 | 6.1398 | 6.1398 | +0.0059 | +0.0059 |
| `uk-govuk` | `(samples)` | converted | 1 | 0.273188 | 0.273188 | 35 | 35 | 292.3110 | 292.3110 | -8.9241 | -8.9241 |
| `us-dol` | `DCTDecode` | direct | 15 | 0.315150 | 0.482667 | 34 | 56 | 11.9734 | 33.8827 | -0.2318 | -0.2882 |
| `fr-cerfa` | `DCTDecode` | converted | 21 | 0.351111 | 1.000000 | 30 | 255 | 17.8306 | 31435.6250 | +0.0325 | +57.2083 |
| `gh-openpdf` | `DCTDecode` | direct | 7 | 0.369112 | 0.997876 | 50 | 255 | 16.7114 | 9956.0341 | +0.0642 | +53.3266 |
| `gh-pdfbox` | `DCTDecode` | direct | 2 | 0.369672 | 0.369672 | 58 | 58 | 19.2649 | 19.2649 | +0.0046 | -0.1506 |
| `gh-qpdf` | `DCTDecode` | direct | 22 | 0.412628 | 0.466199 | 171 | 171 | 2595.6229 | 3100.8935 | +1.6654 | +8.2570 |
| `fr-impots` | `(samples)` | converted | 13 | 0.455042 | 0.499713 | 170 | 255 | 5784.5370 | 12765.8108 | +3.4954 | +44.5049 |
| `us-uscis` | `(samples)` | converted | 1 | 0.466357 | 0.466357 | 35 | 35 | 365.4217 | 365.4217 | -12.2960 | -12.2960 |
| `fr-cerfa` | `(samples)` | converted | 1788 | 0.500000 | 1.000000 | 19 | 241 | 73.0000 | 36864.0000 | +0.3333 | +192.0000 |
| `ia-medical` | `(samples)` | converted | 1 | 0.628462 | 0.628462 | 19 | 19 | 121.8514 | 121.8514 | -8.3359 | -8.3359 |
| `fr-cerfa` | `(samples)` | direct | 232 | 0.888889 | 1.000000 | 115 | 255 | 1504.8095 | 8323.0833 | +0.4167 | +53.5556 |
| `gh-pypdf` | `DCTDecode` | direct | 2 | 0.895425 | 0.895425 | 244 | 244 | 1375.6107 | 1375.6107 | +0.4075 | +0.4075 |
| `gh-openpdf` | `DCTDecode` | converted | 1 | 0.939036 | 0.939036 | 38 | 38 | 98.4109 | 98.4109 | +5.0846 | +5.0846 |
| `ia-uscourts` | `(samples) mask` | direct | 2 | 0.958333 | 0.958333 | 255 | 255 | 62315.6250 | 62315.6250 | -240.3906 | -240.3906 |
| `gh-pypdf` | `(samples)` | converted | 1 | 0.985783 | 0.985783 | 27 | 27 | 121.3016 | 121.3016 | -4.2622 | -4.2622 |
| `ia-biodiversity` | `JPXDecode` | direct | 10 | 0.994521 | 0.999976 | 255 | 255 | 17813.8994 | 32873.8577 | +34.2609 | -172.2092 |
| `uk-govuk` | `DCTDecode` | converted | 1 | 0.994621 | 0.994621 | 61 | 61 | 115.5974 | 115.5974 | -2.1431 | -2.1431 |
| `fr-impots` | `DCTDecode` | converted | 3 | 1.000000 | 1.000000 | 255 | 255 | 22819.4133 | 22891.5689 | -115.6788 | -115.6788 |
| `gh-pdfbox` | `(samples)` | direct | 9 | 1.000000 | 1.000000 | 255 | 255 | 14734.9375 | 27229.9219 | -0.1458 | -62.3750 |
| `gh-pdfbox` | `(samples)` | converted | 4 | 1.000000 | 1.000000 | 255 | 255 | 27389.5091 | 36125.0000 | +12.7500 | +63.8958 |
| `gh-pdfbox` | `JPXDecode` | converted | 1 | 1.000000 | 1.000000 | 255 | 255 | 42179.8812 | 42179.8812 | -170.0170 | -170.0170 |
| `ia-uscourts` | `(samples) mask` | converted | 1 | 1.000000 | 1.000000 | 207 | 207 | 16375.4434 | 16375.4434 | -98.1878 | -98.1878 |
| `ia-uscourts` | `(samples)` | direct | 2 | 1.000000 | 1.000000 | 207 | 207 | 16181.4161 | 16181.4161 | +96.8730 | +96.8730 |
| `ia-uscourts` | `(samples)` | converted | 2 | 1.000000 | 1.000000 | 101 | 101 | 1089.9152 | 1089.9152 | +0.2467 | -28.9192 |

### 3. All three populations that never finished, finished

| population | before | now |
|---|---|---|
| `pdfscans-ia-biodiversity` | killed by the memory watchdog at **24.9 GB**, `rc=137` | **completed**, peak **3.9 GB** |
| `pdfforms-gh-openpdf` | killed at **27.5 GB**, `rc=137` | **completed**, and quickly |
| `pdfscans-ia-americana` | ran to the 45-minute cap, `rc=124`, memory flat; *"whether it is merely a slow population of large scans or something that does not terminate is not known"* | **completed**, peak **5.2 GB**, in **53 minutes** under a 90-minute cap |

The two allocation failures were `render` v0.19.0 walking a page's resources as
a tree when they are a graph, and **v0.20.0's fix is confirmed on both**. The
third was never a defect: `ia-americana` is a slow population of large scans
and the old cap was simply too short. That was the open question and it is
closed.

### 4. Colour conversion was the right diagnosis, and it was most of `(samples)`

`(samples)` used to read **84.9%** over 3810 compared pictures. Splitting the
colour-converted ones out moves **3083 of its 4292 pictures** into their own
bucket, leaving 916 compared and **69.7%**. `fr-cerfa` alone contributes 2949
of them.

The two numbers are not comparable and neither is "the" answer: the old one
averaged two implementations' ICC arithmetic into a decoder's score, and the
new one declines to score that at all. What the split buys is that the 69.7%
is *about a decoder*. And the converted bucket is not thrown away — it carries
its own magnitudes, so the claim can be checked: `fr-cerfa`'s 1788 differing
converted `(samples)` pictures have a **median peak of 19 levels**, which is
colour arithmetic, against its 232 differing **direct** ones at a median peak
of **115**, which is not.

Of the direct `(samples)` pictures that agree, **all 638 are bit-identical**.
Uncompressed samples in `gray` or `rgb` have nothing to round, so that is the
answer that had to come out, and it is a check on the instrument as much as on
the decoder.

### 5. The gate bought the lossless filters nothing, which is how it should be

One gate applies to every filter, and that is a loosening for the lossless
ones. `identical` measures the loosening, and for them it is zero:

| filter | compared | exact | identical |
|---|---:|---:|---:|
| `JBIG2Decode` | 10 | 10 | **10** |
| `JBIG2Decode mask` | 250 | 250 | **250** |
| `JPXDecode mask` | 2 | 2 | **2** |
| `(samples)` | 916 | 638 | **638** |
| `(samples) mask` | 1117 | 1074 | **1074** |

**Every agreeing picture of every lossless filter is bit-equal.** Not one of
them needed the gate. So the uniform `D = 2` costs nothing on the filters that
could defensibly be held to 0, and there is still no measured case for a
per-filter exception table.

### 6. Four documents poppler opens and we do not

`refused` is **4** across 3280 documents, and all four are in
`ia-biodiversity` — a population that had never completed, so the previous
baseline's *"`refused` is 0"* was true of the 20 populations it covered and
untested here. Every other population reports 0.

**Which four documents, and why, is not established.** Identifying them is a
second pass over that corpus and it has not been run; guessing would be worse
than the gap. `unopenable` is 63 and `declined` is 1 across the fleet, and
neither of those is a defect.

### 7. Inversions are still concentrated, and still conventions

268 `JBIG2Decode mask`, 151 `(samples) mask` complements. They are the stencil
polarity convention — `pdfimages` writes a stencil with the opposite polarity
to the samples it holds — and `render` v0.20.0 plus the object-identity finding
in [conformance#13](https://github.com/go-pdfkit/conformance/issues/13) have
already taken most of the spurious `(samples)` complements out: `fr-cerfa`'s
173 are now **15**. The remaining `(samples)` complements are masks, where the
convention belongs.

## What is not measured, and why

- **The four `ia-biodiversity` refusals are counted, not diagnosed.**
- **`DCTDecode`'s 284 differing pictures are not diagnosed either.** Finding 1
  says the disagreement is real and moderate; it does not say whose it is.
  [conformance#13](https://github.com/go-pdfkit/conformance/issues/13) shows
  `match` manufactures disagreements when a page draws many pictures of one
  size, and no per-picture pairing audit was run for this baseline.
- **No aggregate bound is applied.** `mse` and `mean` are recorded in FFmpeg's
  and pdfium's units and bounded by nothing, because no bound has been measured
  for pictures that were *extracted* rather than rendered. Choosing one from
  these records is a job for a later run, and it is now possible because the
  records carry the terms.
- **One page per document.** A first page is not a document.
