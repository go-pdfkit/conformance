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
| taken | 2026-09-24T17:08:38Z .. 2026-09-24T20:07:32Z (UTC) |
| judge | pdfimages version 26.04.0 |
| **measure** | **per channel, gate `D` = 2, count budget `N` = 0** ([conformance#16](https://github.com/go-pdfkit/conformance/issues/16)) |
| **pairing** | **by object number, falling back to size only when one side published none** ([#30](https://github.com/go-pdfkit/conformance/pull/30)); a MASK by the object of the picture that names it ([#32](https://github.com/go-pdfkit/conformance/pull/32), [#34](https://github.com/go-pdfkit/conformance/pull/34)) |
| **counted apart** | a `/Decode` array, and a picture the judge writes as SAMPLES rather than as colour ([#33](https://github.com/go-pdfkit/conformance/pull/33)) |
| **walk** | **the page's CONTENT STREAM, not its /Resources** ([render#43](https://github.com/go-pdfkit/render/pull/43)) |
| **bucketing** | the listing **and** the picture's own `/ColorSpace` ([conformance#20](https://github.com/go-pdfkit/conformance/issues/20)) |
| **bound on the judge** | **2m0s per document, per tool** ([conformance#21](https://github.com/go-pdfkit/conformance/issues/21)) |
| `go-pdfkit/render` | **v0.35.0** |
| `go-pdfkit/reader` | v0.6.0 |
| `go-gfx/gfx` | **v0.31.0** |
| `tannevaled/gobig2` | **v0.2.0** |
| `go-images/jpeg2000` | **v0.1.0** *(was `ajroetker/go-jpeg2000` v0.0.2; see §15)* |
| `go-images/jpeg` | **v0.1.0** *(was the standard library's `image/jpeg`; see §18)* |
| pages per document | 1 (the first page of each document) |
| corpora | `/Users/Shared/pdfscans` (MANIFEST.tsv), `/Users/Shared/pdfforms` (MANIFEST.tsv) |

**These figures still hold eight releases later.** `render` has gone from
v0.35.0 to v0.43.0 since they were taken -- a real bold face where there was a
faked one, and seven changes made for speed -- and two populations re-measured
with v0.43.0 come out **identical**: `ia-texts` and `fr-impots`, every bucket,
every term, and the population counts. The speed changes were each shown
byte-identical over both corpora, and the bold face is text, which this
instrument does not look at. What is stale here is the version numbers in the
rows above, not the numbers in the tables below.

**Every one of the 23 populations ran to completion, and every one is in the
tables below.** All 23 exited 0, and **no document in the set anywhere hit the
judge's two-minute bound** -- the `hung` field is absent from all 23 records.

That last sentence is not free, and §22 says what it cost: the first record for
`ia-americana` had **three** entries in it, and reading the counts without
reading that field turned a bounded judge into what looked like a regression.

**Two decoders changed identity between runs, and both are forks.** The JPEG
2000 one went from `ajroetker/go-jpeg2000` v0.0.2 to `go-images/jpeg2000`
v0.1.0, under the same Apache-2.0 licence (§15). The JPEG one went from the
standard library's `image/jpeg` to `go-images/jpeg` v0.1.0, under Go's own
BSD-3 (§18). Each carries one change, each states it in a `NOTICE`, and the
JPEG fork runs Go's own test suite unaltered.

**This run took two hours and nine minutes without a break.** An earlier one
was interrupted and resumed, and one before that spanned two days — the machine slept between `ia-biodiversity` and what followed. No
figure moves either way: what is compared is pixels, and pixels do not depend
on when the comparison ran. It is the reason this file carries no timings.

**And the binary matters more than the clock.** An earlier set of records for
this same change was measured with a build made over local `replace`
directives, so the module list each record carries said `render` v0.31.0 while
the file said v0.32.0. The code was identical — every one of the 23
populations is byte-identical between the two builds, bucket for bucket — but
a record is not entitled to say what produced it unless it was produced that
way. `TestTheBaselineReadmeDescribesTheRecordsBesideIt` refused the first set,
and it was right to.

**Why this run exists.** Six changes, in two halves, and the same sentence
covers both: **a picture is not a colour, and a name is not an identity.**

Three are in `render`, and none is in a codec. A JPEG carries SAMPLES: for
`DeviceRGB` and `DeviceGray` that costs nothing, but a `Separation`'s sample is
an amount of INK, so a tint of nothing is paper and read as grey it is black —
the largest disagreement in the corpus was a `DeviceN` barcode 255 levels from
poppler on every pixel (v0.23.0). `DeviceCMYK` was drawn with the naive
`(1-c)(1-k)` rather than the printing primaries (v0.24.0), and the CMYK JPEG path
did not even see that fix (v0.25.0).

Three are here. A mask is listed by `pdfimages` under the object of its PARENT,
and nothing recorded such a name, so every mask in the corpus was paired by SIZE
(#32). Publishing that number then made the object test decisive and it failed
at once, because `isMask` knew two of the four types `ImageOutputDev.cc:138-147`
prints (#34). And a one-bit picture over a palette or a tint is written by the
judge as BITS, with the colour space never consulted, so comparing it to our
colour is not a comparison (#33).

Everything else was held, and checked before the run rather than asserted
afterwards: the same judge (`pdfimages version 26.04.0`), the same other module
versions listed above, the same one page per document, the same two-minute
bound.

What that cost the run this replaces, over the same 23 populations, the same
3280 documents and the same judge:

| | v0.22.0 | v0.25.0 | v0.28.0 | v0.30.0 | **v0.32.0** |
|---|---:|---:|---:|---:|---:|
| pictures returned | 8063 | 8063 | 8063 | 8063 | 8063 |
| carrying a `/Decode` array | 545 | 545 | 545 | 545 | 545 |
| **written by the judge as bits** | **0** | **678** | **261** | **261** | 261 |
| with no counterpart at all | 5 | 5 | 5 | 5 | 5 |
| pictures compared | 7513 | **6835** | **7252** | **7252** | 7252 |
| exact | 6657 | 6156 | 6582 | 6586 | **6588** |
| reported inverted | 426 | 425 | 425 | 425 | 425 |
| **reported differing** | **430** | **254** | 245 | 241 | **239** |
| **agreement** | **93.9%** | **96.0%** | 96.4% | 96.5% | **96.5%** |

**The picture count did not move, and that is the point.** 678 pictures left the
comparison at v0.25.0 because the judge writes their SAMPLES and not their
colour, which is a different question rather than a disagreement; the 176 fewer
differing are what is left when 131 of those and 22 mis-paired masks stop being
counted as defects. Seven populations moved, sixteen are identical to the byte,
and **not one moved backwards** on any term.

**The third column is the more interesting one, because it moves the opposite
way.** 417 pictures came BACK into the comparison — the bits rule had been keyed
by name and was reaching pictures it did not mean (§15) — and the differing
count still fell, from 254 to 245. More pictures judged and fewer of them
differing is the only combination that cannot be had by moving the goalposts.

Four changes lie between the two columns, and they do different things. The
object-keyed walk is the 417 (`render` v0.26.0). Reading `CalGray` and `CalRGB`
instead of their device namesakes is the 9 fewer differing, and it is also where
the magnitudes went: `DCTDecode converted` peaks at **33** where it peaked at
110, its worst `mse` falling from 909.34 to 34.99 (v0.27.0, §16). Drawing a
`Lab` colour rather than a grey of its lightness moved no counter in this
corpus, which has 12 documents mentioning `/Lab` and none of them on a first
page (v0.28.0). A fifth follows in the last column: reading an `ICCBased`
profile (v0.30.0, §16).

**A sixth moves no figure in the table above at all.** Decoding a JPEG through
a fork of Go's `image/jpeg` that keeps a four-component picture's chroma
(v0.31.0, §18) took
`gh-openpdf DCTDecode converted` from a worst peak of **33 to 20** and its
worst `mse` from 4.38 to **0.30** — the picture itself from 33 to **3**, which
is what an ordinary JPEG reads. Not one counter moves: it differed at 33 and
differs at 3, both being outside a gate of 2. **The 20 that is now the bucket's
worst belongs to the OTHER picture in it**, the `CalRGB` one, whose residual
§17 accounts for as a decoder disagreement amplified by a calibrated space.

> **This paragraph called that 20 "the largest magnitude left". It was the
> largest PEAK.** The largest `mse` was elsewhere and was fifty times bigger:
> `fr-impots DCTDecode converted` at **34.99**, a peak of 11 spread over 71% of
> a picture's pixels. A peak and a mean squared error do not rank the same
> buckets — one is the worst pixel, the other is every pixel — so a sentence
> that says "largest" without saying largest BY WHAT has quietly picked one and
> called it the answer. §19 is what that sentence should have pointed at.

**The fifth column is the second time a colour change moves a counter, and it
is the last magnitude in this file that was larger than a rounding.** Reading a
press profile's own lookup table (v0.32.0, §19) takes that `fr-impots` picture
from 34.99 to **exact** — not to within the gate, to exact — so `differing`
falls from 241 to **239**. Agreement still reads 96.5%, because 96.47% and
96.50% round the same way; the two pictures are the honest figure and the
percentage is the one that cannot show them.

That makes three fixes in this file — the CMYK JPEG path of §11, the JPEG 2000
of §15 and this one — whose whole value is in the magnitude and none of it in
the count. It is the reason this file quotes sizes, and §19 is what happens
when the prose reads only one of the sizes it quotes.

The fourth moves no counter and is the largest single magnitude in this file's
history: a four-component JPEG 2000 is read as ink rather than as red,
green and blue taken from its first three components, so
`gh-pdfbox/JPXTestCMYK.pdf` goes from **255** levels out on every pixel to a
peak of **3** and an `mse` of 42 179.88 to 0.21 (v0.29.0, §15). It counted as
differing before and counts as differing now; the picture is simply no longer
wrong.

**Nothing moved backwards here either**, and it was checked rather than hoped:
across the first three columns, 7 of the 23 populations changed and 16 are
identical to the byte, and **not one bucket row anywhere gained a differing
picture, a bigger peak or a bigger `mse`**. Of the changed populations, four are
the calibrated step alone — `fr-cerfa`, `gh-openpdf`, `us-opm` and
`ia-uscourts` — and exactly one bucket row separates v0.28.0 from v0.29.0.

**The fourth column is the first in this file where a colour change moves a
COUNTER.** Reading an `ICCBased` profile where it is arithmetic (v0.30.0, §16)
took four pictures from differing to **exact** — not to within the gate, to
exact — so `differing` falls from 245 to 241 and agreement reads 96.5%. Two
bucket rows separate the columns and both go to nought: `ia-medical`
`(samples) converted`, 1 differing at peak 19 and `mse` 121.85, and `fr-impots`
`(samples) converted`, 3 of 16 at peak 23. They were the two largest
unexplained magnitudes the corpus still held.

**Read `unmatched` beside them.** It stays at 5, and that number is the whole
proof that #34 landed: with the mask paired by its parent's object but `isMask`
still naming two types of four, it was **280**.

**The inversion count barely moved, and its MEANING did.** 426 becomes 425.
Before, those were masks matched to the first unclaimed picture of the same
size; now each is matched by the object number both sides publish, and still
comes out an exact complement. The number is the same and the claim behind it is
not — a count that holds while the pairing under it is rebuilt is worth more than
one that improved.

Its path was not quiet. Pairing a mask by its parent's object without #34 left
**280** pictures unpaired and dropped the inversions to 150; the two changes have
to be read together, which is why they are one run and not two.

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
oversight. **Peak memory was not captured for this run** and no earlier figure is
carried over: a number measured on a different build of the thing being measured
is a number about something else.

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
- **raw bits** — the judge wrote this picture's SAMPLES rather than its colour.
  `ImageOutputDev.cc:642` takes `PNGWriter::MONOCHROME` whenever the colour map
  has one component and one bit, and then writes the bytes with the colour space
  never consulted. For a one-bit grey that loses nothing — the sample IS the
  level — so only an index or a tint is counted here. 678 across the fleet, and
  they are neither an agreement nor a disagreement.
- **unmatched** — nothing of the judge's could be paired with this one: no row
  carried its object number, and none of the right size was free to fall back
  to. Since the walk returns what a page **draws** and a mask is looked up under
  its parent's object, this is rare — **5** across the fleet — and each one is a
  question rather than a fact of life.
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
| `ia-medical` | 250 | 0 | 0 | 0 | 0 | 746 | 723 | 194 | 529 | 528 | 33 | 99.8% | 1 | 1 |
| `ia-biodiversity` | 250 | 0 | 1 | 0 | 0 | 754 | 705 | 55 | 650 | 650 | 157 | 100.0% | 0 | 0 |
| `ia-americana` | 250 | 28 | 0 | 0 | 0 | 502 | 494 | 89 | 405 | 379 | 96 | 93.6% | 8 | 8 |
| `ia-texts` | 12 | 7 | 0 | 0 | 0 | 14 | 14 | 3 | 11 | 11 | 2 | 100.0% | 0 | 0 |
| `ia-uscourts` | 250 | 0 | 0 | 0 | 0 | 133 | 113 | 40 | 73 | 65 | 53 | 89.0% | 17 | 3 |

Government and library forms — `/Users/Shared/pdfforms`:

| population | documents | unopenable | refused | declined | hung | pictures | direct | inverted | compared | exact | identical | agreement | converted | calibrated |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ca-cra` | 84 | 0 | 0 | 0 | 0 | 151 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `fr-cerfa` | 450 | 0 | 0 | 0 | 0 | 4378 | 1206 | 15 | 1191 | 1079 | 1020 | 90.6% | 2858 | 1747 |
| `fr-impots` | 50 | 0 | 0 | 0 | 0 | 109 | 20 | 0 | 20 | 19 | 6 | 95.0% | 19 | 3 |
| `gh-openpdf` | 56 | 14 | 0 | 0 | 0 | 49 | 26 | 0 | 26 | 21 | 6 | 80.8% | 2 | 1 |
| `gh-pdfbox` | 157 | 8 | 0 | 0 | 0 | 41 | 29 | 1 | 28 | 26 | 23 | 92.9% | 9 | 1 |
| `gh-pdfcpu` | 147 | 0 | 0 | 0 | 0 | 696 | 639 | 0 | 639 | 599 | 597 | 93.7% | 57 | 32 |
| `gh-pypdf` | 34 | 1 | 0 | 0 | 0 | 15 | 7 | 0 | 7 | 5 | 5 | 71.4% | 7 | 5 |
| `gh-qpdf` | 81 | 0 | 0 | 0 | 0 | 63 | 63 | 0 | 63 | 60 | 41 | 95.2% | 0 | 0 |
| `gh-safedocs` | 26 | 5 | 0 | 0 | 0 | 2 | 2 | 0 | 2 | 1 | 1 | 50.0% | 0 | 0 |
| `gh-verapdf` | 134 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `int-wipo` | 116 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 |
| `uk-govuk` | 302 | 0 | 0 | 0 | 0 | 225 | 135 | 1 | 134 | 134 | 103 | 100.0% | 39 | 18 |
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

| filter | pictures | direct | inverted | **compared** | exact | identical | agreement | converted | conv. exact | conv. differing | calibrated | remapped | raw bits | unmatched | differing | worst peak |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `(samples)` | 4280 | 904 | 0 | 904 | 904 | 904 | **100.0%** | 2950 | 2950 | 0 | 1762 | 286 | 135 | 5 | 0 | — |
| `(samples) mask` | 1336 | 1128 | 154 | 974 | 974 | 974 | **100.0%** | 15 | 15 | 0 | 15 | 67 | 126 | 0 | 0 | — |
| `JPXDecode` | 1241 | 1231 | 0 | 1231 | 1231 | 7 | **100.0%** | 10 | 9 | 1 | 9 | 0 | 0 | 0 | 0 | 3 |
| `JBIG2Decode mask` | 594 | 524 | 274 | 250 | 250 | 250 | **100.0%** | 0 | 0 | 0 | 0 | 70 | 0 | 0 | 0 | — |
| `DCTDecode` | 590 | 425 | 0 | 425 | 208 | 4 | **48.9%** | 44 | 23 | 21 | 32 | 121 | 0 | 0 | 217 | 20 |
| `DCTDecode mask` | 12 | 11 | 0 | 11 | 11 | 5 | **100.0%** | 1 | 1 | 0 | 1 | 0 | 0 | 0 | 0 | — |
| `JBIG2Decode` | 11 | 10 | 0 | 10 | 10 | 10 | **100.0%** | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | — |
| `JPXDecode mask` | 2 | 2 | 0 | 2 | 2 | 2 | **100.0%** | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | — |

Across the whole fleet: **3385 of 3957 direct comparable pictures agree, 85.5%**,
and 1990 of them are bit-identical. Neither figure moves in this run, and §21
says why: the three pictures it gained are all complements, which `compared`
excludes. At v0.20.0 it was 3347 of 3962, **84.5%**,
with the same 1990 identical.

## What this says

**Twenty-two findings, and they are about five runs.** §1 to §8 and §10 were
written about earlier ones — §1 to §5 and §7 about the v0.20.0 → v0.21.0
comparison of 2026-08-31, §8 and §10 about the two runs of 2026-09-07 — and are
kept because their reasoning still holds; where they quote a figure, that figure
is the one their own run measured. §6 was rewritten when a later run disproved
it. **§9 is rewritten again here**, because the gap it called "the work" turned
out not to be a codec at all. §11 to §20 belong to the run of
2026-09-23, and **§21 and §22 to this one**.

Three of them correct this file rather than the library. §14 corrects a claim
§14 itself made before the pictures were split; §16 corrects §11, which named
`CalRGB` among the spaces where reading a JPEG's samples as colour costs
nothing, and it cost 110 levels; and §15 records that the work its own last
paragraph proposed has been done.

Read §11 to §13 together. They are three shapes of one mistake — **a picture is
not a colour, and a name is not an identity** — and each was found by a number
that could not be reconciled rather than by re-reading code.

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
record must not be read as if it were. §22 adds the second way it fires, which
this section did not foresee: **not a document that hangs, but a machine that
was busy**. The bound is on wall-clock time, so what it separates is not
"pathological" from "ordinary" but "finished in time" from "did not", and the
second set grows when something else is running. `pdfforms/gh-qpdf/qpdf_qtest_qpdf_shared-unnamed-field.pdf`
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

That value stopped being hypothetical on 2026-09-20. A throwaway census over
both corpora — `pdfimages -list` on every document, with no bound because it
was a one-off — stalled, and it stalled on this file. Twelve minutes on one
2496-byte document, with nothing on the terminal to say so. The sweep that
carries the bound reads the same corpus in minutes.

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

### 9. `DCTDecode` was never the gap it looked like

Every earlier run of this file named `DCTDecode` as the remaining work, and each
time the number moved for a reason that was not a codec:

| | agreement | what moved it |
|---|---:|---|
| 2026-08-31, v0.20.0 | 43.2% | — |
| pairing by object | 43.7% | the pairing was never its problem |
| walking what is drawn | 48.9% | 61 unpaired pictures no page drew |
| this run | **48.9%** | three colour defects, none in a decoder |

**And this time the agreement did not move at all** — 217 pictures differ, as
before. What moved is the SIZE: the filter's worst peak falls from **255 to
110**, and its converted bucket gains two exact. The colour defects were never
in the many small disagreements; they were in the few enormous ones, and a count
of differing pictures cannot see the difference between 255 apart and 3 apart.

The drill from the aggregate down to one picture is §11. What it found was that
the largest disagreements in the corpus were a `Separation` image drawn as its
own negative, a `DeviceCMYK` drawn with algebra instead of ink, and a CMYK JPEG
converted along a path that had never been corrected — and that the residual,
once those were gone, is bounded and explained (§14).

**Calling a filter "the gap" is a statement about where to look, and it was
wrong three times running.** The filter was never the unit of the defect: the
colour space was.

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

### 11. Three colour defects, none of them in a decoder

The `DCTDecode` gap §9 named as "the work" was not a codec at all. Drilling from
the aggregate down to one picture found three defects in a row, each downstream
of the JPEG being decoded correctly.

> **This section named `CalRGB` among the spaces where the distinction costs
> nothing. That was wrong and it cost 110 levels; see §16.** The sentence is
> corrected below rather than left standing, and the error is left named.

**A JPEG carries samples, not colours.** `decodeJPEG` went from `image/jpeg`'s
output straight to pixels, while the non-JPEG path went through the colour space
the dictionary names. For `DeviceRGB`, `ICCBased` and `DeviceGray` the
distinction costs nothing. For a space that **transforms** its samples it costs
the whole picture: a `Separation`'s sample is an amount of **ink**, so a tint of
nothing is paper, and read as a level of grey it is black.

| picture | before | after |
|---|---|---|
| `uk-govuk` `DeviceN "Black"` over DeviceCMYK, 549×91 | peak **255**, mse 65 025, mean **−255.00** | within the gate |
| `fr-impots` `Separation "PANTONE 293 U"`, 166×84 | peak **255**, mse 22 819 | peak 44, mse 380.8 |

**The second row is not where that picture ended.** Its "after" is this
section's after, and three changes followed it: peak 44 → 11 at v0.27.0, and
then **exact** at v0.32.0, when the `ICCBased` alternate its tint transform
names stopped being read as American ink. §19.

The first was the largest disagreement in the corpus: every pixel 255 levels
from poppler, mean exactly −255 — solid black against solid white. This is what
poppler does and not an interpretation of it: `DCTStream` hands
`GfxImageColorMap` the component samples and the colour map turns them into
colour. `render` v0.23.0.

**`DeviceCMYK` was algebra, not ink.** `cmykToRGBA` called gfx's naive
`(1-c)(1-k)`, which that file documents as *"not for print colour management"*.
It is now the U.S. Web Coated (SWOP) primaries — cyan (0,173,239), magenta
(236,0,140), yellow (255,242,0), key (35,31,32) — interpolated over the sixteen
corners of the cube.

That is not fitting the judge. Over a 625-point grid:

| | max | mean |
|---|---:|---:|
| naive vs poppler | 115 | **27.1** |
| pdf.js vs poppler | 61 | **10.8** |
| naive vs pdf.js | 107 | **27.8** |

poppler and pdf.js approximate the same SWOP table independently and agree with
each other 2.5× more closely than either agrees with the naive formula. The
rewrite was proved against poppler's unrolled matrix verbatim: they agree to
`2.2e-16` over 20 625 points, one ULP. `gfx` v0.20.0, `render` v0.24.0.

**And the CMYK JPEG path never saw that fix.** It went through
`raster.FromImage`, which uses the standard library's naive formula — two paths,
two answers, for one set of numbers. Eight DVLA forms carry a YCCK scan of a
whole page and differed from poppler on 99% of their pixels for that reason
alone:

```
v112   peak 38 mse  76.1  ->  peak 16 mse  3.9
v888   peak 86 mse  79.4  ->  peak 50 mse  5.7
v317   peak 84 mse 192.1  ->  peak 46 mse 26.0
```

`render` v0.25.0. Note what that change did **not** do: the corpus counters did
not move. A picture at peak 64 was differing before and after, and those eight
scans carry a `/Decode` array, which this file counts apart on purpose. The
whole of it is in the SIZE of the disagreements, which is why sizes are quoted
and not a count.

### 12. Masks were never paired by object, and the fix exposed an older defect

`pdfimages` lists a mask under the object of its **parent**: `cerfa_10074.pdf`'s
object 119 is a 2×2 picture whose `/SMask` is object 120, and the smask row says
119. Nothing here recorded such a name, so **every mask in the corpus fell back
to being paired by size** — on a page drawing 211 same-size pictures under masks
carrying the glyph shapes, a lottery.

| `fr-cerfa`, `(samples) mask` | before | after |
|---|---:|---:|
| exact | 711 | **733** |
| **differing** | **22** | **0** |
| **peak worst** | **255** | **0** |

All 22 disagreements were pairs of different masks. **They do not shrink, they
disappear** — which is how a wrong pairing ends, where a decoding defect would
merely diminish.

Publishing that number then made the object test decisive, and it failed
immediately: `isMask` recognised `smask` and `stencil`, while
`ImageOutputDev.cc:138-147` prints **four** types — `image`, `stencil`, `mask`,
`smask` — three of which are masks. Its comment even said *"one of the two
kinds"*. A `/Mask` row matched nothing at all and **unmatched went 5 → 280**,
193 of them in `ia-medical`. That 280 is a state this file never shipped: it was
measured, diagnosed and closed before the run above, which is why the table at
the top of this document reads 5 both sides. The two changes are one run because
either alone is worse than neither.

The test could not have caught it: it listed three of the four types and passed.
**A truth table with a row left out cannot say the row is wrong**, and the row
was missing from the test for the same reason it was missing from the code. All
four are now written out in the order the source declares them.

### 13. The judge can be the lossy one

`ImageOutputDev.cc:642` takes `PNGWriter::MONOCHROME` whenever the colour map has
one component and one bit, and then writes `str->getChar() ^ invert_bits` — the
samples, with the colour space never consulted.

For a one-bit `DeviceGray` that loses nothing: the sample **is** the level. For
an index or a tint it loses everything. `cerfa_10074.pdf` carries an `Indexed`
palette of `808080` and `ffffff`, which we draw as the mid grey it says and the
judge writes as a bit.

| `(samples) converted` | before the rule | after it | **this run** |
|---|---:|---:|---:|
| pictures | 3072 | 2395 *(677 counted apart)* | **2950** *(135 counted apart)* |
| **differing** | **140** | **9** | **4** |
| **peak worst** | **255** | **23** | **23** |

**131 of the 140 disagreements were this**, and `render` was right in every one.
They are counted apart beside `/Decode`, under the rule this file already
carries: the two sides were not asked the same question.

The third column moved in two steps and the middle one is not in this table.
The rule was keyed by NAME and reached pictures it did not mean; keyed by object
it sets aside 135 rather than 677, so 543 pictures came back into the comparison
and the differing count read **11** — more pictures compared, more of them
differing (§15). Then `CalGray` and `CalRGB` stopped being read as their device
namesakes and **7 of those 11 became exact**, not merely within the gate, which
is how 11 became 4 (§16).

What found it was a number that belonged to neither expected answer. Ours read
**128** where the two candidate palettes offered 0 and 255 — and a value absent
from both vocabularies cannot be a rounding. The palette really was `808080`.

### 14. What remains, measured by splitting the pictures rather than by inference

The previous section of this file said "the rest is the IDCT". **That was an
inference and it is wrong.** Splitting `DCTDecode`'s pictures by the shape of
their own frame header settles it. The split was taken at `render` **v0.25.0**
and is left at those figures, because it is the reading that found the defect;
what the same pictures do now is in §16:

| shape and bucket | pictures | differ | **worst peak** |
|---|---:|---:|---:|
| 1 component, 1×1, direct | 36 | **0** | **1** |
| 3 components, 1×1, direct | 68 | 26 | **4** |
| 3 components, 2×1, direct | 1 | 1 | 3 |
| 3 components, 2×2, direct | 209 | 154 | **4** |
| 1 component, 1×1, converted | 6 | 2 | 11 |
| 3 components, 1×1, converted | 2 | 2 | 3 |
| **3 components, 2×2, converted** | 24 | 10 | **110** *(33 at v0.28.0)* |
| 4 components, 2×2, converted | 1 | 1 | **33** *(§18)* |

**How the split was taken**, since the records do not carry it: the shape comes
out of each drawn JPEG's own frame header — the component count is the tenth byte
of the `SOF` segment and the first component's sampling factors are the twelfth,
one nibble each — walked over the page's content stream so the picture is the one
the page draws. Twenty lines against `images.Judge`, and the same numbers come
back. If this becomes a standing question rather than a one-off, the shape
belongs in the records instead.

Three readings, and each is a measurement rather than a deduction.

**A one-component JPEG exercises the IDCT and nothing else** — no chroma to
upsample, no colour to convert — and **not one of the 36 exceeds the gate**, at a
worst peak of 1. Whatever the residual is, the IDCT is not it.

**Chroma upsampling adds nothing measurable.** Unsubsampled colour peaks at 4 and
2×2-subsampled colour peaks at 4 as well. That is what a faithful port looks like
from the outside, and `chroma.go`'s `h2v2Row` is faithful term for term to
`jdsample.c:405-423`: the same `near*3 + far` column sum, the same asymmetric
`+8` / `+7` rounding on the even and odd output columns, the same `*4` edge
replication.

**Every large peak is in the CONVERTED bucket.** The direct buckets, all of
them, stop at 4.

> **What this section then said about those peaks was an inference, and it was
> wrong.** It read: *"110 and 33 are colour-space arithmetic — an ICC or Lab or
> Separation space poppler had to convert — which this file counts apart by
> construction and says is 'not a decoder disagreeing'."* The 110 was neither
> ICC nor Lab nor Separation: it was a `CalRGB` this renderer was reading as
> `DeviceRGB`, and it was ours. §16 measures it and §17 says what the 20 that
> remains is. The 33 is the four-component CMYK JPEG, unchanged by any of
> this, and §18 says what IT is.

So `DCTDecode`'s 217 differing pictures are **all within 4 levels**, and what
makes them differ is the gate rather than a defect. Three IDCTs at ±1 and a
YCbCr-to-RGB conversion at ±1 compound to exactly that: the conversion was
measured over all 16 777 216 colours and differs from libjpeg by 0 (60.15%) or 1
(39.85%) and never more, the constants being identical (91881, 22554, 46802,
116130) and only the rounding differing — Go adds `y×257` where libjpeg adds a
fixed half.

**That is a question about the gate, and this file does not answer it.** `D` = 2
is derived from the ISO/IEC 10918-2 allowance for ONE IDCT, and it is applied
here to a composition of three plus a colour conversion. The derivation does not
cover the thing it is being used on. Changing it would move every figure in this
document, so it is stated and left. §17 makes it worse and measures how: through
a calibrated space the SAME input error arrives multiplied, so one number cannot
serve both buckets.

### 15. What is bounded, and what stopped being

**A four-component JPEG 2000 was drawn wrong, and this section said it was not
fixable here. That was true of the module and false as a conclusion.**

The claim was: `go-jpeg2000`'s `convertToRGBA` has branches for one, two and
"three or more" components, and the last takes the first three as red, green
and blue, so a `JPXDecode` picture in `DeviceCMYK` loses its black plate —
`gh-pdfbox/JPXTestCMYK.pdf` was 1377×443 of that, 255 from poppler at a mean of
−170. And that the decoder's public API hands back an `image.RGBA` and nothing
else, so the fourth component was gone before `render` saw it, which left only
"components out of a third-party module, or decoding JPEG 2000 here".

**There was a third option, and it is the one that was taken: fork the
module.** `github.com/go-images/jpeg2000` v0.1.0 is that fork, under the same
Apache-2.0 licence, with its one change stated in a `NOTICE` as section 4(b)
requires and also offered upstream. A picture whose own JP2 header declares the
enumerated colour space CMYK — EnumCS 12 of ITU-T T.800 Table I.1, which the
file says and the decoder was reading and then ignoring — now comes back as an
`image.CMYK`. `render` v0.29.0 then takes it through the printing primaries
rather than `raster.FromImage`'s `(1-c)(1-k)`, which is the second half of the
lesson §11 records for the CMYK JPEG path.

| `gh-pdfbox/JPXTestCMYK.pdf` | before | after |
|---|---:|---:|
| share of pixels differing | **1.0000** | **0.000033** |
| peak | **255** | **3** |
| `mse` | **42 179.88** | **0.21** |
| `mean` | **−170.02** | **−0.002** |

The largest disagreement in the corpus is now no further from poppler than an
ordinary JPEG.

**What did NOT move is worth as much as what did.** The fleet's agreement is
unchanged at 96.4% and the differing count is still 245: this picture counted
as differing before and counts as differing now, at a peak of 3 against a gate
of 2. The whole gain is in the magnitude — the same shape §11 records for the
CMYK JPEG path, and the reason this file quotes sizes rather than a count.

**And the change touched nothing else, which was checked rather than assumed.**
Exactly one bucket row in the 23 populations differs between the two runs, the
one above; the fleet's 1233 direct `JPXDecode` pictures are 1233 exact before
and after. That matches what this section already verified about the corpus:
across the scans, JPEG 2000 pictures are `DeviceRGB` (718), `DeviceGray` (513),
`ICCBased` (5) and one naming no space at all, with **no `DeviceCMYK`** — so
there was nothing there for the fix to change, and it changed nothing.

**The pairing no longer guesses, and this is what it was worth.** This section
used to end by proposing the work. `render.Images` named a picture by its
resource name and this package rebuilt the object number by walking the
document again; where a name reached two objects — forms without `/Resources`
of their own — the ambiguity rule correctly refused an identity and the size
fallback drew from a hat.
`gh-qpdf/qpdf_qtest_qpdf_form-xobjects-no-resources-out.pdf` draws four 15×15
grey pictures (objects 9–12) and `Im1` means two of them.

`render` v0.26.0 carries `Image.Object`, and `objectsByName`, the ambiguity rule
and the mask-to-parent mapping are gone with it. The trap named here was real
and is honoured: a mask's entry carries its **parent's** number, because that is
what `pdfimages` publishes.

| `gh-qpdf` | before | after |
|---|---:|---:|
| `(samples)` direct, exact / differing | 39 / **2** | **41 / 0** |
| worst peak in that bucket | **32** | **0** |
| population agreement | 92.1% | **95.2%** |

Two pictures, as forecast — and the worst of them was the one comparison in the
whole corpus whose `mean` was exactly nought beside a peak of 32, which is what
comparing two DIFFERENT pictures looks like. The repository README's bound
section was built around that outlier; it no longer exists.

**And it moved 543 pictures out of "counted apart" and into the comparison,
which IS a gain — it is 543 pictures this file had stopped scoring.** The
raw-bits rule sets aside a one-bit picture whose palette the judge cannot
write; keyed by name it was reaching pictures it did not mean. Keyed by
object it reaches the right ones, and the movement balances to the entry:

| `(samples)` | counted apart | direct | converted |
|---|---:|---:|---:|
| v0.25.0 | 678 | 904 | 2407 |
| v0.26.0 | **135** | 904 | **2950** |

126 masks went the other way, for the same reason and as correctly:
`(samples) mask` set aside 0 and now sets aside 126, its direct and converted
counts falling by exactly that. Net, 417 more pictures are judged than before,
and none of them was drawn differently — this is the instrument seeing what it
already had.

### 16. The judge converts colour two ways, and the line between them is not where this section first drew it

This file has been counting a `converted` bucket and saying a difference in it
is "colour arithmetic rather than a decoder disagreeing". That is half right,
and the half it gets wrong is the half that was actionable.

`libpoppler.159` links `liblcms2` and calls it: `cmsOpenProfileFromMem`,
`cmsCreateTransform`, `cmsDoTransform`, eleven entry points in the Homebrew
binary. **But it does not use it for every converted space.**

| space | what `pdfimages` actually runs |
|---|---|
| **ICCBased** | `GfxICCBasedColorSpace::buildTransforms` (`GfxState.cc:1823`) falls back to `GfxState::sRGBProfile` when no display profile is set, so the transform is built **unconditionally** and the DOCUMENT's profile is applied, at relative-colorimetric intent. |
| **CalRGB, CalGray, Lab** | their transform comes from `state->getXYZ2DisplayTransform()`, and `GfxState`'s constructor builds it from a null profile (`:6489`). `pdfimages` never sets one. The little-cms branch is **never armed**, and a pure arithmetic path runs: gamma, matrix, Bradford to D65, sRGB primaries, sRGB encoding. |

So the bucket holds two different kinds of disagreement:

- against **a calibrated space**, poppler does arithmetic that is written down
  and reproducible. Reproducing it exactly — including the colour map's
  quantisation of the sample to 1/65536 before the gamma — gives **0 differing
  channels out of 15 552** on the picture below. Any disagreement there is
  ours.
- against **ICCBased**, poppler consults the document's profile through
  little-cms. Whether that is answerable depends on what the profile IS, and
  this section first said it was not answerable at all. See below: the sentence
  it used is struck through, because it was wrong for the common case.

**We had been treating the second as if it were the first**, and §11 said so in
as many words: *"For `DeviceRGB`, `CalRGB`, `ICCBased` and `DeviceGray` the
distinction costs nothing."* For `CalRGB` it cost 110 levels.

**What found it.** `openpdf-core/pdf-2-0_PDF_2.0_image_with_BPC.pdf` draws the
**same 1466-byte JPEG stream twice** — once through a `CalRGB` (Adobe RGB
matrix, D50 white point, gamma 2.2) and once through `DeviceRGB` — and its own
page text says *"It should appear red and the black darker than on the right"*.
The two extracted PNGs are the document's own controlled experiment, and
subtracting them measures poppler's CalRGB transform and nothing else: peak
**110** over **47.8%** of channels.

Reading the space cost two changes, because the second hid the first.
`colourSpaceArray` mapped `CalRGB` onto `deviceRGB` and `CalGray` onto
`deviceGray`; and `jpegThroughSpace` asked the colour space only for a
**one-component** picture in one of four named families, so a three-component
JPEG never reached a space at all. The rule is now that a space which is not
one of the four device singletons has something to say about the samples,
whatever their count.

| the CalRGB picture | before | after |
|---|---:|---:|
| peak | **110** | **20** |
| share of pixels differing | 0.5764 | **0.0098** |
| mse | 909.34 | **0.30** |

`gfx` v0.22.0, `render` v0.27.0. Three other properties of the judge came out
of the same reading and are recorded here because they bound what this
instrument can say — and the third is a correction:

- **A 16-bit sample is truncated to its high byte.** `ImageStream::getLine`
  does `imgLine[i] = *p++; p++;` and `GfxImageColorMap` forces
  `maxPixel = 255`; both halves carry the comment *"this is a hack"*. A decoder
  that reduces 16 bits to 8 by **rounding** — the correct thing — will differ
  from the judge by ±1 on every such picture, and be right.
- **A `Separation` named `Black` over `DeviceGray` short-circuits its tint
  transform** to `1 − tint` — but never for an image. `GfxImageColorMap`
  pre-computes the tint into `lookup2` and calls the alternate space directly,
  so the shortcut is reachable only from a fill. Testing a fill and concluding
  about an image would be wrong.
- **`Lab` looked like a third disagreement and was not one.**
  `GfxLabColorSpace::getXYZ` returns the f-inverse values where ISO 32000-2
  8.6.5.4 says `X = Xw·g(M)`, and reading that function alone says poppler
  departs from the format. **It does not** — `::getRGB` multiplies immediately
  after calling it. What settled it was a witness rather than a closer reading:
  a hand-built four-pixel `Lab` document run through `pdfimages` matches the
  specification's formula on 4 of 4 pixels within one level and departs from
  the no-white-point formula by 13 levels on a neutral mid tone.

  So there was no obstacle, and `labSpace` — which drew a grey of the right
  lightness and said so in its own comment — was simply a defect of ours.
  `render` v0.28.0 draws the colour, reads `/Range`, and gives `Lab` the one
  default `/Decode` in the format that is not `[0 1]` per component
  (`[0 100 amin amax bmin bmax]`, without which a lightness of 50 was read as
  0.5 and the picture came out black). poppler draws `Lab(50, 20, −30)` in a
  space with no white point as (131, 109, 171); we draw (131, 108, 170). For
  scale: 12 of 2268 form documents mention `/Lab`, none of the 1013 scans.

**A correction, and the second time this section has had to make one.** It
said: *"against ICCBased, poppler consults a colour profile we do not read, and
no amount of work on a decoder closes that. It is a different pipeline, not a
defect."* The first sentence was true and the second does not follow from it.

**Most profiles are not an engine. They are a tone curve per channel and a
matrix to the connection space** — which is the same shape `CalRGB` and
`CalGray` have, and which this repository had already learnt to compute. The
picture that measures it is
`pdfscans/ia-medical/2011001RegenerativeEndodonticsPart2.pdf`: 104×125 in an
`ICCBased` space whose profile is 540 bytes — gamma 1.8008 on each channel, and
three colorants that sum to (0.96422, 1.0, 0.82489), which is D50, as the
connection space requires.

| that picture, 39 000 channels | before | after |
|---|---:|---:|
| worst channel difference | **19** | **1** |
| channels within one level | 35.75% | **100.00%** |
| its bucket here | 1 differing, peak 19, `mse` 121.85 | **exact** |

| `fr-impots` `(samples) converted` | before | after |
|---|---:|---:|
| differing | **3** of 16 | **0** of 16 |
| peak / `mse` | 23 / 13.27 | — |

Those were the two largest unexplained magnitudes left in this corpus, and they
were the same thing. `gfx` v0.26.0 reads such a profile; `render` v0.30.0
converts through it.

**What is genuinely not answerable is narrower than the sentence claimed.** A
profile whose transform is a multi-dimensional lookup table — an `A2B0` tag,
which is how CMYK and most scanner profiles are written — needs an engine, and
`ReadICC` returns `ErrICCNotArithmetic` rather than approximating it. A caller
that meets one falls back on the component count exactly as before **and knows
that it did**, which is the difference between a limit and a silent wrong
answer.

> **That paragraph is the fifth row of §16's own table, and it stood for two
> versions.** "Needs an engine" was read as "needs an engine nobody here is
> going to write". The engine is four arrays and an interpolation; it is
> `gfx/color/icclut.go`, it is read at v0.27.0, and the picture it was costing
> was the largest `mse` in the corpus. **§19.** What the paragraph got right is
> the last clause: falling back knowingly is not the same as being wrong
> silently, and it is why the picture could be found at all.

**And a part of this that NEITHER instrument can see.** A colour space is read
the same way for a FILL as for a picture, so reading the profile changes every
`ICCBased` fill on every page as well. `images` extracts pictures, so not one
of those is in any figure above.

This section first said `compare` was where that would show. **It is not**, and
the reason is worth more than the correction. `compare` reduces both pictures
to grey, blurs them, and counts the pixels differing by more than 64 — a
quarter of the range — because it is asking *"is this the same page to somebody
looking at it"*. A nineteen-level colour shift is discarded at the first step
and would not cross the third. Run before and after over 40 `fr-impots`
documents and the whole `ia-medical` witness, it reports **exactly the same
numbers**, which is what a blind instrument reports.

That is not a defect in `compare`; it is what it is for, and its own package
comment says so. It means something narrower and more useful: **nothing in this
repository measures colour on a PAGE.** `images` measures colour and never sees
a fill; `compare` sees every fill and never sees colour. **Closing that gap was tried, and the attempt is the finding.** Three
discriminators were built and measured against two witnesses whose pictures are
known to have changed — `ia-medical`'s, which went from differing at peak 19 to
exact, and `fr-impots`' `2042_2042_4756.pdf`, from peak 23 to exact. All three
were added beside `Share`, never in place of it.

| what it kept | `ia-medical` | `fr-impots` | why it failed |
|---|---|---|---|
| every pixel, blurred | 5.826 → 5.790 | — | glyph edges reach **180 levels**; the signal is in the noise |
| pixels where both pages agree on brightness | 0.027 → 0.027 | 0.113 → 0.113 | a grey profile moves TONE, so the pixels it changes are the ones this discards |
| pixels in a flat area of both | **0.027 → 0.001** | 0.113 → 0.113 | inside a picture, a tone curve shows where the tone CHANGES, which is not flat |
| worst 32-pixel square | 18.226 → 18.226 | 15.502 → 15.502 | the worst square on a page of text is text |

The third works on one witness and not the other, and that is not a
discriminator, it is a coincidence with two data points. The experiment is kept
as a patch beside this corpus rather than landed.

**What it establishes is the shape of the difficulty, which is worth more than
a number that half works.** A page-level figure has to separate a disagreement
about WHERE THE INK IS — which two rasterisers have on the edge of every glyph,
at up to 180 levels — from a disagreement about WHAT COLOUR IT IS, which is
smaller and covers an area. No local test does it: brightness agreement and
local flatness each throw away the tonal half of the signal, and tiling is
defeated because the worst tile of a document is always its densest text.

And under that sits dilution: one 320×290 picture on a 595×842 page is 18% of
it, so twenty levels inside the picture arrive as three levels of page average
— less than the glyphs contribute. `images` does not have this problem because
it compares a picture to a picture with no rasteriser in between, which is
exactly what makes it unable to see a fill.

So the gap stands, and it is now known to be a real one rather than an
unattempted one.

**The shape of the error is worth more than the fix.** FIVE times in this file
a limit turned out to be a boundary of the thing in front of me rather than of
the problem:

| where | what it said | what the limit really was | what it cost |
|---|---|---|---:|
| §11 | the distinction costs nothing for `CalRGB` | our own code read it as `DeviceRGB` | peak **110** |
| §15 | a four-component JPEG 2000 is not fixable here | a third party's module, which can be forked | peak **255 → 3** |
| §16 | poppler consults a profile we cannot read | true of a lookup-table profile, false of the common one | peak **19 → 1** |
| §18 | somebody would have to write a decode that hands back its planes | one call in Go's `image/jpeg`, and Go is BSD-3 | peak **33 → 3** |
| §19 | a lookup-table profile needs a colour-management engine | four arrays and an interpolation; the engine is one file | `mse` **34.99 → 0** |

Each was true of one module, one function or one library, and each was stated
as though it were true of the question. The fourth is the plainest: the
sentence named the fix it needed and then assigned it to nobody, when the
obstacle was a single line importing an internal package.

**The fifth was carved out by the third's own correction, and that is the part
worth keeping.** Row three is §16 narrowing "poppler consults a profile we
cannot read" to "true of a lookup-table profile, false of the common one". The
narrowing was correct and the measurement behind it was real — and the smaller
sentence left standing was wrong in exactly the same way as the larger one.
**A correction that names a smaller limit is still naming a limit**, and it
carries the whole burden of proof it has just discharged for the part it kept.
Four rows of this table were written before that one was tested.

**The test that would have caught all five**: a sentence that cannot NAME the
artefact whose limit it describes is not describing an artefact. It is
describing the problem, which is a far stronger claim than it looks. And the
second test, which row five needed: when a limit is narrowed rather than
removed, **the remainder has not been measured just because the part that went
away was**.

### 17. A calibrated space amplifies a decoder disagreement

The `CalRGB` picture above still differs by 20 after the fix, and the reason
matters more than the number.

| question | answer |
|---|---:|
| our conversion applied to **poppler's own decoded samples** | **1 level** |
| a sample error of **±3** carried through that same space | **35 levels** |

±3 is not hypothetical: it is what the `direct` bucket measures for this very
stream. So the residual 20 is the JPEG disagreement magnified, not a colour
defect — and the magnification is a property of the space, not of the picture.
A gamma of 2.2 linearises the sample, the matrix mixes the channels, and the
sRGB encoding re-compresses with a slope of 12.92 in the toe, so one level in
can be twelve levels out; an out-of-gamut colour is then clipped, which moves
it again.

**The corollary is what nearly fooled the test written for this.** A NEUTRAL
mid tone barely moves — 129 against 128 — because the gamma and the sRGB
encoding very nearly cancel and the white-point adaptation keeps a grey grey. A
check written on a mid grey passes whether the space is consulted or not. The
saturated colours are where a calibrated space and its device namesake part
company, and the test now uses one.

**What this does to the gate.** §14 already recorded that `D` = 2 is derived
from the ISO/IEC 10918-2 allowance for one IDCT and applied here to a
composition of three plus a colour conversion. This is worse than that: through
a calibrated space the SAME input error arrives at the output multiplied by a
factor that depends on the gamma, the matrix and the position in the gamut.
Comparing the `direct` and `converted` buckets against one number compares two
different quantities, and this file should stop implying otherwise. It is
stated and not acted on, for the reason §14 gives: changing `D` would move
every figure in this document.

### 18. The four-component JPEG lost its chroma before `render` could see it

§14 left two large peaks in the `converted` bucket. §16 answered the 110. This
is the 33, and it is the last one in this corpus that is ours.

The picture is `gh-openpdf/objectXref.pdf`'s 258×258 `DeviceCMYK` JPEG, and its
frame header says what it is: an Adobe APP14 transform of 2 — **YCCK** — with
the luma and the black plate at 2×2 and the two chroma planes at 1×1, so the
chroma is subsampled by two in each direction.

**Separating the decode from the conversion.** `pdfimages -tiff` writes a
`DeviceCMYK` picture as a four-sample separated TIFF, which is poppler's own
CMYK **before** any colour conversion. Against the samples `render` produces
for the same stream — Go's `image/jpeg` output with the `255 − v` that
`uninvertAdobeCMYK` applies — 91.2% agree within one level, and the worst per
plane is:

| plane | C | M | Y | **K** |
|---|---:|---:|---:|---:|
| worst difference | 36 | 18 | 21 | **1** |

**K is the control and it is the whole argument.** It is carried at full
resolution and is never upsampled, and it is right. Sorting every pixel by how
fast the chroma changes across neighbouring blocks sorts the error with it:

| chroma gradient across neighbouring blocks | C plane | K plane |
|---|---:|---:|
| flat (≤ 2) | 5 | 0 |
| gentle (3–15) | 19 | 1 |
| strong (16–40) | **36** | 1 |
| steep (> 40) | 35 | 1 |

A decode that were simply different would move K too. This one does not.

**Why.** `render`'s `chroma.go` replicates libjpeg's fancy upsampling term for
term, and §14 measured the result: three-component pictures peak at 4 whether
their chroma is subsampled or not. It never runs here. Go's `image/jpeg`
`applyBlack` (`reader.go:675`) merges the four planes itself, through
`imageutil.DrawYCbCr`, which samples chroma by **replication** — and hands back
an `*image.CMYK`. The planes are gone before `render` is given anything.

**Could `render` put it back? Partly, and the bound was measured rather than
guessed.** Inverting the conversion recovers the per-block chroma exactly where
nothing clipped: over 14 244 blocks with no clipped channel, 14 215 show a
spread of **0.0** within the block and the worst is 0.5, which is the rounding
of Go's integer conversion and nothing else. But 7.8% of blocks have every
pixel clipped on some channel, and there the chroma is genuinely gone.
Re-upsampling what can be recovered, with libjpeg's own filter, takes the worst
CMY error from **36 to 26** — and the 26 that remains sits entirely in the
blocks that cannot be recovered.

**This section then said that was the end of it, and named "a four-component
decode that hands back its planes" as the honest fix somebody else would have
to write. The fourth time this file made that shape of mistake.** The decoder
is Go's, Go's licence is BSD-3, and a decoder can be forked.

`github.com/go-images/jpeg` v0.1.0 is that fork: Go's `image/jpeg`, licence and
copyright retained, every change stated in its `NOTICE`, and **Go's own test
suite passing there unaltered**. What it changes is the single call this
section names. The obstacle turned out to be one line, not a library:
`imageutil` is imported once in the whole package, at the call site being
replaced, so nothing else had to be carried or rewritten.

| the 258×258 YCCK picture, 266 256 samples | C | M | Y | **K** | within one level |
|---|---:|---:|---:|---:|---:|
| `image/jpeg` | 36 | 18 | 21 | **1** | 91.16% |
| the fork | **2** | **2** | **3** | **1** | **99.90%** |

And end to end, through `render` v0.31.0, in the bucket above:

| `gh-openpdf` `DCTDecode converted` | before | after |
|---|---:|---:|
| share of pixels differing | 0.1106 | **0.0003** |
| peak | **33** | **3** |
| `mse` | 4.3791 | **0.3022** |

**Peak 3 is what an ordinary JPEG reads in the direct bucket**, so the last
large peak this corpus held is no longer one. Reconstruction would have
stopped at 26; the fork reaches 3 because the planes are never merged wrongly
in the first place.

**Nothing else moved, and the control is the one that matters.** `fr-cerfa`,
`uk-govuk`, `gh-pdfcpu` and `us-dol` are byte-identical before and after.
`uk-govuk` is the test: its DVLA scans are four-component too, at 1×1 chroma,
which the change cannot reach — and does not.

**One thing the fork leaves standing.** `render`'s `chroma.go` and the fork now
each carry libjpeg's fancy upsampler, because they are reached differently: a
three-component picture comes back as an `*image.YCbCr` with its planes intact
and `render` upsamples them, while a four-component one is merged inside the
decoder. The two were checked against each other at every output position,
edges included, and agree exactly; and each is measured against poppler
independently — peak 4 for three components, peak 3 for four. That is a
duplication with a reason and a cross-check, which is not the same as a
duplication.

### 19. The largest error left was ranked out of sight by the statistic that found the others

**This section exists because a sentence in §15 said "the largest magnitude
left" and meant "the largest peak".** Ranked by peak, the worst thing in the
corpus was `gh-openpdf DCTDecode converted` at 20, which §17 already accounted
for. Ranked by mean squared error, the worst thing was somewhere else
altogether and was **fifty times larger than anything below it**:

| | worst peak | worst `mse` | share of pixels differing | mean |
|---|---:|---:|---:|---:|
| `gh-openpdf DCTDecode converted` | **20** | 0.30 | 0.0098 | +0.28 |
| `fr-impots DCTDecode converted` | 11 | **34.99** | **0.7123** | **−3.92** |
| everything else, 20 buckets | ≤ 4 | ≤ 0.65 | ≤ 0.0020 | — |

Those two numbers describe **different shapes of error**. A peak of 20 over 1%
of the pixels is a few pixels a long way out. A peak of 11 over 71% of them,
with a mean of −3.92, is the whole picture shifted. The first is a decoder
disagreement; the second is a colour that was never computed the right way at
all. The file's tables carried both figures the whole time; the prose ranked by
one of them and stopped looking.

#### What it was

`2042_2042_5180.pdf` and `2042_2042_5535.pdf` draw the same 166×84 JPEG — the
one §11 pulled back from 255 levels out — through

```
[/Separation /PANTONE#20293#20U 35 0 R << /FunctionType 2 /N 1 /C0 [0 0 0 0]
                                          /C1 [1 0.57 0 0.02] >>]
```

whose alternate space, object 35, is `[/ICCBased 95 0 R]` with **`/N 4`**. That
stream is 654 352 bytes of **Coated FOGRA39 (ISO 12647-2:2004)**: device class
`prtr`, space `CMYK`, connection space `Lab`, and its transform in an `A2B0`
/`A2B1`/`A2B2` trio of `mft2` tags — four inputs, three outputs, eleven points
per axis, 14 641 measured grid points.

`gfx`'s `ReadICC` declined every profile of that shape, so `render` fell back
on `DeviceCMYK`, which is this repository's interpolation of **U.S. Web Coated
(SWOP)** (§11). European coated offset drawn as American web ink: a small,
signed, everywhere error. Exactly a mean of −3.92 over 71% of a picture.

#### The engine, and three things that had to match rather than merely be defensible

poppler calls little-cms, so "correct" here means "what little-cms answers".
Three details each cost more than the whole gate.

**The interpolation is not the obvious one.** Weighing all 2ⁿ corners of a grid
cell is the reading anyone would write first. little-cms cuts each cell into
six tetrahedra and weighs the four corners of the one the point falls in; for
four inputs it interpolates linearly between two sub-lattices read that way.
The two agree at every grid point and differ between them:

| Coated FOGRA39, 1041 colours, 600 of them random interior points | worst ΔE76 | mean |
|---|---:|---:|
| all corners weighted | 0.261 | 0.058 |
| **as little-cms reads it** | **0.005** | **0.002** |

**The CIELAB encoding changed at version 4.** In a version 2 profile's 16-bit
table, `0xff00` and not `0xffff` stands for L\* = 100. This profile is version
2.1.

**Black point compensation is on, always.** poppler builds every transform with
`cmsFLAGS_BLACKPOINTCOMPENSATION`. Its black point is a roundtrip through the
profile — ask what ink it would lay for absolute black, then ask what that ink
measures — and **the two legs use different tags**: in through the perceptual
`B2A0`, back through the media-relative `A2B1`. Reading both legs perceptually
is a plausible mistake and answers L\* = **0.5** for this press where the
answer is **10.7**, which is a compensation that does nothing.

#### The witness

Each step was measured against poppler on the picture itself, and against
little-cms configured the same way, so that "we match" is not the same claim as
"we are close".

| | worst level from poppler | mean |
|---|---:|---:|
| little-cms, no compensation | 3.15 | 1.239 |
| **ours, the table alone** | **3.15** | 1.233 |
| little-cms, with compensation | 0.97 | 0.448 |
| **ours, compensated** | **0.97** | **0.448** |

And the picture, through the instrument this file is written from:

| `fr-impots` `DCTDecode converted` | share | peak | `mse` | mean |
|---|---:|---:|---:|---:|
| before | 0.7123 | 11 | 34.99 | −3.92 |
| the table alone | 0.0085 | 4 | 1.66 | +0.88 |
| **with the compensation** | — | — | — | **exact** |

Not within the gate of 2. **Exact.**

#### The control

Every one of the 23 populations was re-measured, and the comparison was read in
both directions rather than only the flattering one:

- 83 buckets before, 83 after — none lost, none invented
- **not one bucket anywhere gains a differing picture, a bigger peak or a
  bigger `mse`**
- exactly one bucket changes: `fr-impots DCTDecode converted`, 2 differing → 0
- the corpus goes from 6586 to 6588 exact, `differing` from 241 to 239

#### How far this reaches, which is not two documents

The measurement moved two pictures, and that is not the size of the change.
Two instruments were pointed at the corpus, and they answer two different
questions.

**What is embedded.** Every ICC profile in the bytes, found by inflating every
stream — a profile is always a standalone stream, so this reaches all of them
without a PDF parser, and the encrypted or `LZWDecode`d documents were
normalised with `qpdf --decrypt --stream-data=uncompress` first:

| profile shape | documents | read since |
|---|---:|---|
| `RGB` matrix and curves | 1127 | v0.26.0 (§16) |
| `GRAY` one curve | 63 | v0.26.0 (§16) |
| **`CMYK` `mft2` lookup table** | **57** | **v0.27.0, this section** |
| **`CMYK` `mft1` lookup table** | **3** | **v0.27.0, this section** |
| `mAB ` version 4 lookup table | **0** | not read |

**What is drawn through.** `survey`, which is in this repository and opens the
documents rather than reading round them, counts the profile behind every
colour space a page's drawing reaches — through a `Separation`'s alternate, a
`DeviceN`'s, an `Indexed` base, and through a form's own resources:

| what gfx makes of it | documents |
|---|---:|
| matrix and curves | 448 |
| one curve | 57 |
| **lookup table, 4 channels** | **24** |
| malformed | 6 |

> **That table had a sixth row and the row is why this section grew twice.** At
> `gfx` v0.28.0 it read *a parametric curve, declined — 7 documents*, and one
> of those seven was **Display P3**: sRGB's own transfer function written out
> as a formula, over DCI-P3 primaries. Drawing its samples as sRGB instead was
> **up to 101 levels of 255 out, mean 23**. v0.29.0 reads the five shapes ICC
> defines, and the row is gone; only 3 of the 7 appear in the row above,
> because 4 of them also carry an ordinary profile and were already counted
> there.

**60 embed one, 24 draw through one, and 36 name an `/OutputIntents`** — which
is a declaration of the press the document is FOR, not a colour anything is
drawn in. That is the gap, and it is the right shape: a government form
declares PDF/X conformance and prints in black.

> **60 − 36 = 24 is a coincidence of totals, and the check that would have
> published it as a finding is the one that was nearly skipped.** Per
> population the two do NOT line up: `fr-cerfa` has one document with no
> output intent and TWO that are drawn through, and `gh-pypdf` has one that
> embeds a press profile nothing draws through and declares no intent either.
> The sets are the same SIZE and not the same DOCUMENTS. Two errors in
> opposite directions cancelled in the total, which is the oldest way for a
> proxy measure to look like a proof.

So the honest sentence is the third one: **24 documents draw through a profile
this change reads**, where before it read none of them, and two of those
drawings are pictures the instrument can compare.

#### What is still declined, with its size

Two shapes, and the census above is what makes them countable rather than
rhetorical:

| shape | documents drawing through one | what it would take |
|---|---:|---|
| a `para` shape outside the five ICC defines | **0** | knowing what it means |
| a version 4 `mAB ` lookup table | **0** | a second sandwich: curves, matrix, curves |

**This section first asserted the second of those without looking**, which is
precisely the habit §16's table is about, and it did not know about the
parametric curve at all. Both were named and counted once `gfx` v0.28.0 named
the shape in the refusal itself rather than saying only that a profile was not
read — one bucket for a table engine and five formulas counted nothing, and it
took a census to notice — and the parametric one was then simply read (v0.29.0).

**And it moved no figure in this file, which is the honest half.** All 23
populations were re-measured and **every bucket in the corpus is
byte-identical** — not "no counter moved", not "within the gate": the same
numbers, all 83 of them. Those seven documents colour
FILLS through their profile, and `images` extracts pictures: it is the
paragraph §16 ends on, arriving as a measurement rather than as a caveat. The
change is worth 101 levels on seven documents and nothing at all here, and
**both halves of that sentence had to be measured**: the first against lcms,
the second against the whole corpus.

### 20. What `D` = 2 permits, derived rather than inherited

**This file has carried `D` = 2 since the beginning and has never derived it
for what it is applied to.** §1 notes that a figure in circulation used a gate
of 4; §7 says no per-filter exception has a measured case; §14 observes that
every remaining direct difference sits one or two levels above the gate and
declines to act — *a gate moved because it would flatter a number is not a
gate*. That refusal is right and it leaves a question open: **is 2 the number
this pipeline's own conformance permits, or a number inherited from a
different measurement?**

#### Where the 2 comes from

From ffmpeg's IDCT conformance test, read in the source rather than in a
quotation (`libavcodec/tests/dct.c:259`):

```c
spec_err = is_idct && (err_inf > 1 || omse > 0.02 || fabs(ome) > 0.0015);
```

`err_inf` is a **per-output-sample** maximum against a **floating-point**
reference IDCT. A conformant implementation is therefore within 1 of that
reference, and **two** conformant implementations are within 2 of each other.
That is the derivation, and it is correct — **for one IDCT**.

#### What it is applied to

A three-component JPEG puts three IDCTs, a chroma upsampling and a colour
conversion between the coefficients and the pixels this file compares.

**Upsampling cannot amplify.** Fancy upsampling is `(3·near + far + rounding)/4`
— weights that are non-negative and sum to one. A convex combination of values
each within ε is within ε. The bound is unchanged, and §14 measured exactly
that: 1×1 chroma peaks at 4 and 2×2 chroma peaks at 4 as well.

**The colour matrix does amplify**, by the sum of the absolute values of its
row:

| channel | JFIF row | gain | from ±2 per component |
|---|---|---:|---:|
| R | `Y + 1.402·Cr` | 2.402 | 4.80 |
| G | `Y − 0.344136·Cb − 0.714136·Cr` | 2.058 | 4.12 |
| **B** | **`Y + 1.772·Cb`** | **2.772** | **5.54** |

Rounding to eight bits on both sides adds one more. So:

| shape | what stands between the coefficients and the pixels | permitted |
|---|---|---:|
| 1 component | one IDCT | **2** |
| 3 components | three IDCTs, upsampling, the matrix above | **6.5** |

#### The premise is measured, not assumed

The derivation rests on both decoders being conformant IDCTs, which is an
assumption about our own. **The grey pictures measure it directly**: a
one-component JPEG exercises the IDCT and nothing else, and §14's split found
**0 of 36 outside the gate at a worst peak of 1**. One, against a permitted 2.

#### What the corpus looks like against the derived bound

| | differing | inside 6.5 |
|---|---:|---:|
| `direct` | 217 | **217** |
| `converted` | 22 | 20 |
| **total** | **239** | **237** |

Every direct difference in this corpus sits at a peak of **4 or less**. The two
that do not are `gh-openpdf DCTDecode converted` at peak 20 — and 6.5 is a
*lower* bound for a converted picture, which carries a colour conversion on top
of the decode. §17 measured that space's amplification at about twelvefold, so
a peak of 20 there is what a sample error of under 2 becomes: inside the same
premise, arrived at from the other end.

**So every difference this file reports is consistent with two conformant
decoders**, and the 239 count is the gate's, not the decoders'.

#### Two things this derivation does NOT cover

- **`JPXDecode`.** One picture differs, at peak 3. JPEG 2000's conformance is
  the irreversible 9/7 wavelet, not an IDCT, and `err_inf > 1` says nothing
  about it. Its bound would have to be derived from its own standard.
- **A calibrated space, exactly.** §17's twelvefold is a measurement of one
  space at one point, not a bound. A derived gate for the converted bucket
  would need the amplification of each space, which is a Jacobian and not a
  constant.

#### And it is still not acted on

`D` is a published condition of this instrument
([conformance#16](https://github.com/go-pdfkit/conformance/issues/16)); moving
it moves every figure in this file and every comparison with an earlier run.
The arithmetic above is an argument, not a mandate, and a derivation that
happens to raise a rate deserves more suspicion than one that lowers it, not
less.

What changes here is only that the file no longer has to say *"one or two
levels above the gate"* without being able to say what the gate is for. It can
now say which of its differences are inside what conformance permits — **237 of
239** — and that the remaining two are accounted for by §17 rather than
unexplained.

### 21. Three stencils a resource budget was refusing, and an instrument that cannot see what that bought

A JBIG2 symbol dictionary is bounded twice in `gobig2`: a **per-symbol** cap
and an **aggregate** cap charged for every symbol the dictionary holds. Both
existed to stop one adversarial seed -- 198 megapixels across 538 symbols.

The per-symbol cap was the weaker idea of the two. A dictionary cannot spend
more than the aggregate however it splits it up, so capping one symbol lower
than the aggregate **forbids a shape rather than bounding an amount of work**,
and the 16 MP symbol that motivated it is stopped by the aggregate check on the
very next statement. Raising it to the aggregate widens no worst case.

The aggregate was calibrated on the seed rather than on the corpus. A scanned
page's ink layer is a **multiple** of the page, not a fraction of it: at 16 MP
three real documents had theirs dropped and were drawn from their background
alone. **64 MP takes all three, and 96 and 128 MP take nothing further** -- the
population is exhausted at 4x. What that costs is the same factor of
adversarial decode, ~540 ms against ~135 ms, and the seed is **still refused**
at 64 MP exactly as it was at 16.

#### What it moved here: three pictures, and no rate at all

| | before | after |
|---|---:|---:|
| `ia-biodiversity` `JBIG2Decode mask` pictures | 251 | **253** |
| `ia-medical` `JBIG2Decode mask` pictures | 247 | **248** |
| fleet `JBIG2Decode mask` pictures | 591 | **594** |
| fleet `JBIG2Decode mask` **compared** | 250 | **250** |
| fleet `JBIG2Decode mask` agreement | 100.0% | **100.0%** |

All three land in the **inverted** column -- they are the complement half of a
mask pair, which this instrument excludes from `compared` by construction. So
three stencils that previously did not decode now decode, and **not one figure
in the agreement tables moves**. Read here alone, the change is invisible.

That is not a small print: it is what this instrument is. It compares
**extracted pictures** against `pdfimages`. A stencil that is dropped does not
make a picture disagree -- it makes a picture **absent**, and an absent picture
is not counted as a difference by any of the four terms. **The page is where a
dropped ink layer shows**, and the page is measured by a different tool.

#### What it moved on the page, against a control that changes one thing

`compare`, over the same 250 documents, twice, from **one binary built twice**
-- current in everything but the cap, which the control pins at 16 MP through a
`replace`. Both records are in [`baseline/pages/`](pages/), unedited:

| `ia-biodiversity`, 250 pages | 16 MP | 64 MP |
|---|---:|---:|
| worst page | **59.40%** | **20.32%** |
| p90 | 1.36% | **1.10%** |
| p99 | 5.89% | 5.86% |
| pages under 1% | 218 | **220** |
| under 2% | 235 | **237** |
| under 5% | 244 | **245** |
| under 10% | 248 | **249** |
| mean \|diff\| | 11.352 | **11.308** |
| worst square, mean over pages | 22.23 | **21.78** |
| median | 0.0020 | 0.0020 |
| identical byte for byte | 32.17% | 32.17% |

`insectlivesastol00simp.pdf` leaves the worst twelve altogether: it was the
page at 59.40% and it is now below 2.18%, which is where twelfth place sits.

Two things in that table are worth as much as the headline. **The median does
not move, and neither does bit-identity**: this is three pages out of 250, and
a population statistic taken over the middle cannot see three pages however
badly they were drawn. And the control **reproduces the previous run's figures
for this population exactly** -- 0.5940, 1.36%, 218, 11.352, 22.23 -- so
nothing else that changed between the two runs touched it. The cap answers for
all of it.

`ia-medical` gained a stencil too, and its mean went the OTHER way: 6.469 to
**6.482** levels, with no page crossing any threshold. That is not a
contradiction and it is not noise. Drawing an ink layer that was being dropped
puts ink on a page that had none there; where our stencil and poppler's differ
by a pixel at an edge, the page is **structurally** right and **arithmetically**
a little further away. A mean over levels cannot tell those apart, which is the
same reason the share-of-pixels column exists beside it.

### 22. The judge was cut off by the load on the machine, and the counts read as a regression

The first record taken for `ia-americana` with this build lost pictures against
the one it replaced:

| | JBIG2 masks | `JPXDecode` | of those, exact |
|---|---:|---:|---:|
| the run this replaces | 82 | 226 | 221 |
| the first re-measure | **79** | **220** | **215** |
| the same build, again | 82 | 226 | 221 |

`documents`, `unopenable`, `refused` and `declined` were **identical** across
all three. Nothing errored. The first reading of it was that the change had
cost us six pictures, and the second was that it was unexplained.

It was neither. The record carries a `hung` field, `omitempty`, and the middle
run had **three entries in it** -- three documents `pdfimages` did not finish
inside the 2m0s bound this file has always published. **The bound is on the
judge, not on us.** A document the judge does not finish yields no reference
pictures at all, so our counts fall with no error anywhere.

Three documents, each a scanned page of the shape §21 describes -- one JBIG2
stencil and two `JPXDecode` layers -- is **-3 and -6**, which is exactly what
was lost. And `exact` fell by as much as `pictures`: not one picture had begun
to disagree, several had simply not been judged.

What put the judge over its bound was this machine, busy compiling and running
test suites while the measurement ran. The rule against measuring under load
was already written down; what was not, and is now, is **how** it fails. It
does not add noise to a figure. It **truncates the reference** and leaves a
figure that is internally consistent, plausible, and short.

The check that belongs in front of any such explanation is one line: read the
field that counts what was abandoned, before reading the field that counts what
was found. An `omitempty` field is absent from every healthy record, which is
exactly why nobody looks for it.

## Every differing bucket in the run

> **The previous revision of this table carried three rows its own records did
> not produce**, and they are removed here. `fr-cerfa DCTDecode converted` at a
> worst peak of 11, `ia-uscourts DCTDecode converted`, and `gh-qpdf DCTDecode
> direct` were the figures of an EARLIER run, left standing beside the current
> ones.
>
> The cause was the tool that installs these tables, not the one that computes
> them. It matched a generated row to the row it replaced by the table and the
> first cell — the population — and this table has **three rows for
> `gh-pdfbox`, three for `fr-impots` and two for several others**, so the key
> collided and the rows it did not match were left behind. `tables.py` was
> right throughout; the check around it was the half that was wrong, because it
> asked only whether every generated row appears in the file and never whether
> every row in the file is generated. **A check in one direction cannot see
> something extra.** It now asks both ways, and a third time for duplicates,
> and the four tables are replaced whole rather than row by row.

| population | filter | bucket | differing | share med | share worst | peak med | peak worst | mse med | mse worst | mean med | mean worst |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `gh-pdfbox` | `JPXDecode` | converted | 1 | 0.000033 | 0.000033 | 3 | 3 | 0.2057 | 0.2057 | -0.0020 | -0.0020 |
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
| `uk-govuk` | `DCTDecode` | converted | 1 | 0.000225 | 0.000225 | 3 | 3 | 0.1181 | 0.1181 | +0.0260 | +0.0260 |
| `fr-cerfa` | `DCTDecode` | converted | 11 | 0.000259 | 0.001846 | 3 | 4 | 0.1429 | 0.3225 | +0.0225 | +0.3024 |
| `gh-qpdf` | `DCTDecode` | direct | 3 | 0.000319 | 0.000319 | 3 | 3 | 0.3483 | 0.3822 | +0.2801 | +0.3099 |
| `gh-pdfcpu` | `DCTDecode` | direct | 40 | 0.000349 | 0.000923 | 3 | 4 | 0.2554 | 0.3582 | +0.0015 | -0.2789 |
| `ia-uscourts` | `DCTDecode` | converted | 2 | 0.000388 | 0.000388 | 3 | 3 | 0.0936 | 0.0936 | +0.0678 | +0.0678 |
| `gh-pdfbox` | `DCTDecode` | direct | 2 | 0.000508 | 0.000508 | 4 | 4 | 0.2605 | 0.2605 | +0.0024 | -0.1511 |
| `gh-pypdf` | `DCTDecode` | direct | 2 | 0.000517 | 0.000517 | 3 | 3 | 0.2581 | 0.2581 | -0.0561 | -0.0989 |
| `gh-safedocs` | `DCTDecode` | direct | 1 | 0.000595 | 0.000595 | 4 | 4 | 0.2810 | 0.2810 | -0.1230 | -0.1230 |
| `us-dol` | `DCTDecode` | direct | 15 | 0.001119 | 0.001339 | 3 | 4 | 0.3029 | 0.3190 | -0.2787 | -0.2882 |
| `gh-openpdf` | `DCTDecode` | converted | 2 | 0.009838 | 0.009838 | 20 | 20 | 0.3022 | 0.3022 | +0.2829 | +0.2829 |

## §23 — Where this runs, which is not where the references run

Measured 2026-09-26. Every number above compares us against poppler on one
machine, an arm64 Mac. That comparison cannot say anything about a claim we had
been making implicitly all along — that a pure-Go stack runs anywhere Go runs.
That was a property of the **language**, not a measured property of this code.

Ten repositories now run their whole test suite on six architectures per commit:
`render`, `reader`, `pdffont`, `conformance`, `opentype`, `fonts`, `jpeg`,
`jpeg2000`, `gfx`, `gobig2`. Before this they cross-compiled for four 64-bit
targets and ran on none of them; `jpeg2000` had never been built for a 32-bit
target at all.

| | |
|---|---|
| `riscv64`, `loong64`, `ppc64le` | green from the first run |
| `s390x` — **big-endian** | green from the first run |
| `386`, `arm` — **32-bit `int`** | four real defects |

**The big-endian result is the negative one worth recording.** This stack
reassembles multi-byte numbers out of byte streams everywhere — image samples,
font tables, JBIG2 bitplanes, JPEG markers — and not one of those readers carried
an endianness assumption. Nothing had to be fixed for s390x.

**The 32-bit lanes found four defects**, all of the same shape: a ceiling whose
arithmetic overflowed before the ceiling could refuse anything.

1. `render/affordDecoded` — `cw*ch > maxImagePixels` in an `int`.
   `65535 × 65535` wraps to **−131 071** in an `int32`, so a **1 KB file** made
   the decoder ask for four gigabytes and `image.NewGray` panicked.
2. `render/decodeBase` — the same product, the same fix.
3. `render/Page` — converted the extent to an `int` *before* judging it.
   Converting a float too large for an `int` is undefined in Go, so a `MediaBox`
   of `1e300` **panicked on arm64 too**. The 32-bit lane only led us to it; the
   defect was on the main platform.
4. `opentype` table directory — `int(be32(length))` is **−1** on 32-bit for
   `0xFFFFFFFF`, so `off+length` came out *smaller* than `off`, the "out of
   range" check passed a table that was not in the file, and the slice panicked
   `b[184:183]`.

All 3 215 corpus documents render byte-identical across those fixes, so nothing
above this section moved.

### Two lanes that failed on the bound rather than on the code

`riscv64`, `s390x` and `386` failed `TestAPageMayBeGivenOnlySoLong`, which
allowed a fixed five seconds. The deadline is consulted once every 256 drawing
operations, so the overshoot is a quantity in **operations**; five seconds was
that quantity converted at the speed of the machine that wrote the test. The
same correct code took 5.4 s, 8.3 s and 11.3 s. The bound is now a ratio against
an unbounded draw of the same page — 0.109 measured at 40 000 rectangles — so
every fixed cost cancels and no machine's speed is written into the assertion.

### `386` runs without an emulator, and that is a correctness requirement

`qemu-i386` loses the guest's floating-point state across the signal Go delivers
for asynchronous preemption. Eight identical lanes per condition on one commit:

| condition | failures |
|---|---|
| `qemu-i386` | **5 of 8** |
| `qemu-i386`, `GODEBUG=asyncpreemptoff=1` | 0 of 8 |
| native on the same x86-64 runner | 0 of 8 |
| `qemu-arm` | 0 of 8 |

Each failure hit a different input although the seed is fixed, and the wrong
value was a sign flip rather than a rounding difference. A 386 ELF is a native
binary on that runner, so the emulator bought nothing and cost a lane that went
red at random.

**Method note.** Two instrumented runs of the panicking test passed, and that
was written down as the instrumentation mattering — before the unmodified
subject was sampled and found to pass 3 times in 8. Each probe had been a single
draw from a 38%-pass population and had shown nothing. The rate has to be
measured on the unmodified subject before any arm can be compared.

### What the references do

poppler and pdfium test on none of the four 64-bit architectures here. 32-bit
ARM they do hold, and we did not until now. This section is the only part of
this document where the comparison is not a number against poppler's number, and
it is the only part where the answer is ground the references do not cover.

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
- ~~**The bound on the judge is unexercised by `images` on this corpus.**~~
  **It fired, three times, and §22 is what it cost to find out.** Finding 5
  said this of the run that first wrote it and it stayed true through several
  more, which is how it came to be read as a property of the corpus rather than
  of one run. It is a property of the **machine**: the bound is on wall-clock
  time, and the corpus was measured beside a compiler. What is true of the
  records beside this file is the narrower thing the Conditions now state --
  none of *these* 23 has a `hung` entry.
- **One page per document.** A first page is not a document.
