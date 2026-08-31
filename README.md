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

## The judge can hang, and a hang looks like a slow run

`pdfimages -list` **does not return** on
`pdfforms/gh-qpdf/qpdf_qtest_qpdf_shared-unnamed-field.pdf`, a document of
**2496 bytes**. Neither does `pdfimages`, and neither does `pdfinfo`. Two
sweeps of this repository stalled on it and had to be killed by hand
([conformance#21](https://github.com/go-pdfkit/conformance/issues/21)).

That is a property of the judge rather than a defect of ours, and the danger is
not the hang: it is that **a hang and a long job are indistinguishable from
outside**. A sweep that stops dead at document 900 of 2268 is waited on rather
than investigated, and the wait has no end.

So every invocation of a poppler tool here goes through
[`internal/poppler`](internal/poppler/poppler.go), under one bound, and a
document that exceeds it is **recorded by name with the tool that hung** —
never dropped, never retried silently. `images` reports it as `hung`, the
baseline record carries the list in `hung`, and `compare` names the page. A
named timeout is data about the corpus; a silent one is a gap a reader cannot
tell from a bad score, which is the distinction the record already makes
between `refused` and `unopenable`.

Three things follow from taking the hang seriously rather than merely surviving
it.

**The deadline is read off the context and not off the error.** A killed
process reports a signal and a tool that merely failed reports a status, so the
error alone cannot tell a hang from a refusal — and telling them apart is the
whole point.

**A `pdfinfo` that hangs is not an unopenable document.** `blame` asks poppler
whether it would open what ours refused, and counting no answer as *"neither
would open it"* would credit the document as `unopenable` — which is subtracted
from a population's real size, so it would quietly shrink the denominator every
rate is quoted over.

**A listing that hangs does not make a page's pictures colour-converted.** The
listing is half the instrument: it is what puts every picture into the `direct`
bucket or the `converted` one. A page whose listing never came back would be
tallied as wholly converted, which is a real number in a real column and
indistinguishable from a page of CMYK. It is reported as a hang instead.

**The version probe is asked on both streams.** `pdfimages -v` prints its
version on **stderr**, so the one invocation whose whole purpose is to record
*which* poppler judged a run comes back empty from a plain read of stdout. The
judge is half the measurement — a filter whose agreement falls because poppler
changed has not regressed — so `poppler.Combined` exists beside `poppler.Run`
for it, and only for it: a listing that is read column by column must not have
the tool's warnings folded into the table.

**The bound is not calibrated from timings, and says so.** The machine these
runs are made on is shared, so a duration measured on it measures the other job
as much as this one, and a bound read off a loaded machine would fire on
documents that are merely large. It is set far above any plausible handling of
one page — two minutes, `-timeout` on both commands — so that a firing means a
hang. The value the run used is recorded beside the gate, because a run under a
shorter bound names documents as hung that a longer one measures.

## What is compared, and what `exact` asserts

**The comparison is per channel**, and it landed with the re-measurement
below. `difference` in [`images/images.go`](images/images.go) subtracts the two
pictures channel by channel and carries four terms out of one walk:

| term | what it is |
|---|---|
| **`peak`** | the largest absolute difference any channel reached, in levels of 255. **This is the criterion**: a picture agrees when its peak is at most `D`. |
| **`share`** | the fraction of pixels where some channel differs by more than `D`. It is a **report and never a budget**: with the count budget `N` at zero, *"share is nought"* and *"peak is within `D`"* are the same statement, which is why pdfium computes a percentage and then requires it to be zero (`testing/image_diff.cpp:287-288`). |
| **`mse`** | the mean squared error over every compared channel, in levels squared — FFmpeg's `omse` (`dct.c:256`) and pdfium's `mse` (`image_diff.cpp:181`) in their own units, so a published limit can be read against it directly. |
| **`mean`** | the **signed** mean error, ours minus theirs, in levels — FFmpeg's `ome`. This is the term that catches **bias**. |

`D` is **2** and `N` is **0**. Neither aggregate term is a pass criterion, and
that is deliberate: **no bound on either has been measured for pictures that
were extracted rather than rendered**, and adopting pdfium's 0.05 for a
different operation would repeat exactly the mistake the withdrawn 1% was — a
number carried onto an instrument that did not produce it. They are recorded so
that a bound can be chosen from evidence later.

Beside `exact`, every bucket also counts **`identical`**: how many of the
agreeing pictures differed by *nothing at all*. A gate is a loosening, and a
reader who cannot tell bit equality from agreement the gate carried has an
agreement rate that means less than it looks like.

### What it replaced, and what that could not see

Until this landed, the per-pixel predicate was a **bisection**: a pixel was
*ink* when its alpha was at least 128 and its luminance below 128, and `Share`
was the fraction of pixels whose ink **classification** differed. For a bilevel
picture that is exact — a stencil, a CCITT page, a JBIG2 page hold nothing the
bisection can lose. For anything else it was strictly weaker, in one
particular direction: **a decoder rendering every pixel of a scan at luminance
120 where poppler renders 20 scored 0.000**, perfect agreement, on an error of
100 levels at every pixel. A systematic level or chroma shift is the
characteristic failure of a lossy decoder, so the instrument was blind in
exactly the direction the codecs fail.

**Every figure taken with that instrument is that narrower thing, and cannot be
subtracted from a figure taken with this one.** `baseline/README.md` says
which populations carry which, and no table mixes them.

### The three things the rule did not settle

**Colour conversion is not codec error, and per channel it is large.** Our side
is RGBA from `render`; theirs is a PNG `pdfimages` wrote. A CMYK, ICC or Lab
picture reaches those two forms through two different sets of colour
arithmetic, and a `D` of 2 fires on all of it. Widening `D` to absorb that
would destroy the gate, so instead **each picture carries the colour space
`pdfimages` reports for it** — `ImageOutputDev.cc:152-190` prints `gray`,
`rgb`, `cmyk`, `lab`, `icc`, `index`, `sep`, `devn`, and `-` for a mask — and
the pictures poppler had to *convert* to reach RGB are tallied in **their own
bucket**, with their own agreement figure and their own magnitudes, the way
`remapped` and `inverted` are already counted apart.

**The listing alone was not enough, and that was
[conformance#20](https://github.com/go-pdfkit/conformance/issues/20).** poppler
folds `csCalGray` onto `gray` and `csCalRGB` onto `rgb`
(`utils/ImageOutputDev.cc:159-164`) while the pixels it writes go through
`colorMap->getRGB` (`:451`), which for a `GfxCalRGBColorSpace` applies the
gamma, the matrix and the chromatic adaptation. So the `direct` bucket admitted
pictures poppler **had** converted, and measured every pixel of them against a
colour conversion we did not make. All four pictures that still differed by
more than four levels after `render` v0.21.0's chroma fix were `/CalRGB`.

**So the bucket is decided by both sides.** A picture is `converted` when
`pdfimages` says so, **and also when its own `/ColorSpace` resolves to a
CIE-based space — `CalRGB`, `CalGray`, `ICCBased` or `Lab` — whatever the
listing says.** Only the first two can move anything, since poppler lists
`ICCBased` as `icc` and `Lab` as `lab` and both were converted already; naming
all four makes the rule a statement about what the **document** says rather
than a patch over one judge's table. The space is read from the picture's own
dictionary, through the page's forms as `render.Images` walks them, and through
the resource dictionary when the picture names its space rather than writing it
out. **`calibrated` counts how much of a bucket the document itself accounts
for** — the pictures whose own `/ColorSpace` is CIE-based — per filter and per
bucket. It is **not** the count of what the rule *moved*, and reading it as one
overstates the change by two orders of magnitude: most calibrated pictures are
`ICCBased` or `Indexed`, which the listing already called converted. What moved
is what the listing called `gray` or `rgb`, and that is a difference between two
runs rather than a column in either — `baseline/README.md` measures it.

Where it cannot decide, it does not: a name unique within one resource
dictionary is not unique across the several a page reaches, so a page where two
forms each name their own `Im1` and only one is `/CalRGB` marks both. That
over-counts `converted` by at most those pictures, and the other direction is
the one that credits a colour conversion to a codec.

Two honesties remain. `index` is counted **converted** although its base space
is often `DeviceRGB`, because `pdfimages` does not report the base and a picture
that cannot be classified must not be credited as agreement. And a picture whose
listing row could not be read at all also lands in the converted bucket, which
makes a failure of `pdfimages -list` **loud** — every filter would read as
wholly converted — rather than silently generous.

**`Inverted` survives as its own signal.** `pdfimages` writes a stencil with
the opposite polarity to the samples it holds, which a magnitude measure would
report as maximal error at every pixel. So the complement is tested in the same
pass, and `inverted` means *the direct comparison failed the gate and the
complemented one passed it*. The direct comparison is tried first, so a uniform
mid-grey — which is within the gate of its own complement — is reported as
agreeing rather than filed away as a convention.

**Alpha, and what a mask is compared in.** For an ordinary picture the compared
channels are `R`, `G` and `B`; alpha is left out, because `pdfimages` writes an
opaque picture for anything that is not a mask and writes a soft mask out as
its own file, so a difference in alpha would be a difference in what the two
tools chose to *emit*. A mask is not comparable channel for channel, and
`render` does not put one in a single place either. Both layouts were read out
of the buffers rather than assumed:

- a `/ImageMask true` **stencil** carries no colour of its own, so `render`
  returns it black with the shape in the **alpha** channel — `us-opm`'s
  `SF2801PR.pdf` `Im0`, 325×240, every RGB nought and exactly two alpha values;
- an `/SMask` is eight-bit greyscale, so `render` returns it **opaque with its
  levels in RGB** — `us-opm`'s `sf2822.pdf` `Im0/SMask`, 116×73, alpha 255 at
  all 8468 of its pixels.

poppler writes both as opaque grey, black where the mask paints. So a mask is
compared in **one derived channel, ink coverage**, by the same formula on both
sides: `alpha × (255 − luminance) ÷ 255`. On a stencil the luminance is nought
and it is the alpha; on a soft mask the alpha is 255 and it is the inverted
luminance; on poppler's side it is always the inverted luminance. One formula,
correct for all three layouts, and it is what the bisection was doing per bit.

Taking the alpha of a soft mask instead — which the first draft of this measure
did — made `us-opm`'s one agreeing `(samples) mask` read as a 48% disagreement
with a peak of 255 and a mean of +88.6. That was an artefact of the reduction
and not a decoder, and it is recorded here because it is the kind of thing a
new instrument produces before anyone checks it against the buffers.

## The 1% tolerance is withdrawn

A previous revision of this file said: *"A `DCTDecode` or `JPXDecode` picture
agrees when at most 1% of its pixels differ. Every other filter must be
exact."* **That rule is withdrawn.** It was never implemented — `images`
records `Share` and counts `Share == 0` as exact, and nothing in this
repository has ever applied a 1% — so withdrawing it changes no number. What
it changes is what this document claims.

**It is withdrawn because of what it was measured on.** The empty band it was
read from — eighteen population medians below 0.0087, then nothing until
0.0950 — is a property of the bisection, not of decode fidelity. Binarisation
flips cluster where a picture has content near luminance 128, so that band
describes how this corpus's tone distribution meets one threshold. Under a
measure with magnitude in it the distribution is a different distribution and
the band may not survive. The reasoning from the data was sound; the data was
narrower than it was taken to be.

**And it is a shape the field avoids.** Read from the code of the projects
that do this job (clones under `/Users/Shared/biblio/`, line numbers from
those checkouts):

| | worst-pixel bound | aggregate bound | count of differing pixels |
|---|---|---|---|
| pdf.js | exact | — | no |
| poppler (our judge) | exact, MD5 | — | no |
| pdfium `--fuzzy` | ≤ 3 per channel | MSE ≤ 0.05 | computed, **required to be 0** |
| Ghostscript `bmpcmp` | ≤ `-t` per channel, default 0 | — | no |
| Cairo | < 25 per channel, hard cap | perceptual model | no |
| OpenJPEG / ISO 15444-4 | PEAK, per component | MSE, per component | counted, not a criterion |
| FFmpeg / ISO 10918-2 | peak ≤ 1 per sample | MSE ≤ 0.02, mean ≤ 0.0015 | no |
| pixelmatch | OKLab HyAB ≤ 0.1 | — | returned, budget left to the caller |
| Playwright | pixelmatch's, default 0.2 | — | budget, **default 0** |
| reg-cli | YIQ, default 0 | — | budget, **default 0**, applied second |
| Resemble.js | 16 per channel | — | ratio reported |
| blink-diff | 20 in colour space | — | budget, default 500 |

**Nobody counts bare inequality.** Every row bounds *how much* a pixel may
differ before it counts at all; several compute a differing-pixel count and
deliberately decline to make it the criterion. pdfium computes a percentage
and then requires it to be zero (`testing/image_diff.cpp:287-288`) — the
percentage is a report, never a budget. Where a count budget does exist it is
the *second* condition on top of a per-pixel bound, and it defaults to **0**,
not to 1%: Playwright's `maxDiffPixels = maxDiffPixels1 ?? maxDiffPixels2 ??
0` (`packages/utils/comparators.ts:101`), reg-cli's `--thresholdRate` and
`--thresholdPixel` both "0 by default" and both "Applied after
`matchingThreshold`" (`README.md:50-51`). Even blink-diff, the one default
budget here that is not zero, has a per-pixel `delta` of 20 in front of it
(`index.js:153`). `Share` **was** a count over a predicate with no severity in
it at all, which is the one construction all of them independently avoid; it
is now a count over a magnitude gate, and it is reported rather than spent.

Cairo states the objection outright, and it is the one that applies to us.
`test/buffer-diff.c:43-44`:

```c
/* Don't allow any differences greater than this value, even if pdiff
 * claims that the images are identical */
#define PERCEPTUAL_DIFF_THRESHOLD 25
```

used at `test/buffer-diff.c:177-182` under the comment *"Only let pdiff have a
crack at the comparison if the max difference is lower than a threshold,
otherwise some problems could be masked."* Cairo runs the most permissive
comparison of anyone here, a full perceptual model, and still refuses to let
it speak unless the worst single channel is within 25.

**A whole-image percentage is also scale-dependent, which a threshold should
not be.** 1% of a 4000×4000 render is 160 000 pixels — a contiguous block of
400×400, a redaction box or a signature. pixelmatch's windowed mode
(`index.js`, the `windowSize` scan) exists to bound *density* instead, and a
whole-image count is its degenerate case.

## Where `D` and `N` come from

The rule, and the reasons are ours rather than borrowed:

> **Compare per channel. A pixel counts as differing only when some channel
> differs by more than `D`. A picture agrees when at most `N` such pixels
> differ, and when the aggregate error is within bound. `D` and `N` default to
> 0 and are raised per case with a recorded reason.**

**`D` is 2, not pdfium's 3.** pdfium compares *rendered pages*, so its
3 buys slack for rasteriser and anti-aliasing differences. We have none to
buy: `pdfimages` extracts rather than renders, which is why this tool asks it,
so there is nothing between the codec and the pixels. What is left is codec
rounding, and the standards bound that. ISO/IEC 10918-2 requires a conformant
JPEG IDCT to be within **one level per sample** of the reference — read out of
FFmpeg's implementation of the test, `libavcodec/tests/dct.c:259`:

```c
spec_err = is_idct && (err_inf > 1 || omse > 0.02 || fabs(ome) > 0.0015);
```

Two conformant decoders sit on either side of that reference, so **2** is what
they may legitimately differ by, and it is a bound to be *derived* rather than
guessed. JPEG 2000 is tighter still: most of the ISO/IEC 15444-4 Table C.6
PEAK limits are **0**, per OpenJPEG's `tests/conformance/CMakeLists.txt:310`,
so JPX is required exact on most conformance files and permitted a small
bounded error on a few.

**Whatever a per-case exception is raised to, 25 is a ceiling it must not
cross**, for Cairo's stated reason. There is **one gate and no per-case table**
in the code, because there is no case yet: nothing measured has asked for an
exception, and an exception mechanism with nothing in it is a promise rather
than a measurement.

**An aggregate term is warranted, and it is two terms.** pdfium found
the mirror of our defect — a per-pixel gate with an unbounded count, where
many small forgiven differences accumulate — and answered it with a
mean-squared error (`testing/utils/pixel_diff_util.h:11-12`,
`testing/image_diff.cpp:171-183`). Both are taken, as FFmpeg carries them at `dct.c:259` where
`fabs(ome) > 0.0015` sits beside `omse > 0.02`. The two catch different things:
MSE catches accumulated noise, the signed mean catches *bias*. Bias is the
failure the bisection was blind to, so it is the term most specifically needed
here, and it is cheap.

**`N` is 0.** That is the field's default wherever a count
budget exists at all, and pdfium — doing our exact job — requires its
percentage to be zero. A nonzero `N` should be per population and per filter,
each one carrying a written reason, as pdfium's 142 scoped entries in
`testing/SUPPRESSIONS_EXACT_MATCHING` do and as pdf.js's nine
`knownPartialMismatch` tests do. **If `N` is ever raised, it should be a count
within a window rather than a share of the whole picture**, so that the
threshold does not loosen as the picture grows.

The design, and the re-measurement it required, are
[conformance#16](https://github.com/go-pdfkit/conformance/issues/16).

## One rule for every filter, not one per codec

The withdrawn rule split on the source data being lossy. **That split goes
with it, and nothing in the survey supports it.** No project read for this
uses one comparison for lossy inputs and another for lossless.

pdfium is the direct evidence, because it is doing our job and does draw a
line — just not that one. Its fuzzy matching is per-test opt-in through
`testing/SUPPRESSIONS_EXACT_MATCHING`, and the block that admits
`image_8bit_devicergb_dctdecode.pdf` (line 160) admits `image_bmp.pdf` (161),
`image_gif.pdf` (163), `image_png.pdf` (165) and `image_tif.pdf` (166) in the
same contiguous run, every one of them lossless, every entry scoped `mac`, all
under the comment at lines 53-54:

```
# TODO(crbug.com/459586268): Remove these entries once macOS support for Intel
# hardware goes away. Then rebase the test expectations as needed.
```

Platform and hardware, never codec loss. And JPEG 2000 — which our withdrawn
rule treated exactly as it treated JPEG — is not loosened at all there but
disabled outright, `testing/SUPPRESSIONS:735-737`.

**The seam the field cuts is "does this case have a documented reason to be
fuzzy", not "is the source data lossy".** So what landed is one rule and one
number: **`D` is 2 for every filter**, carrying the ISO citation as its
recorded reason, and there is no per-case table because there is no case.
`Gate` is a constant in [`images/images.go`](images/images.go) and an
exception mechanism with nothing in it would be a promise rather than a
measurement.

Two consequences, and both are stated rather than buried.

**It is a loosening for the lossless filters**, which could defensibly be held
to 0 — JPEG 2000 conformance is required exact on most of ISO/IEC 15444-4's
files, and CCITT and JBIG2 have no rounding at all. Whether that loosening
bought anything was measured rather than assumed, and the answer is **nothing**:
across both corpora, every agreeing picture of `JBIG2Decode`, `JBIG2Decode
mask`, `JPXDecode mask`, `(samples)` and `(samples) mask` is **bit-identical**
— 250 of 250, 10 of 10, 2 of 2, 638 of 638, 1074 of 1074. Not one needed the
gate, so the uniform `D` costs the lossless filters nothing and there is still
no measured case for an exception table.

**And it stops the split doing harm in the other direction.** Under the split a
lossless filter could never be granted anything however good the reason, and
`(samples)` was the live candidate: its 84.9% was two implementations doing
different ICC conversion read through a bisection. What that needed was not a
wider count budget but comparison per channel with the colour-converted
pictures counted apart, and it now has both: 3083 of `(samples)`'s 4292
pictures are colour-converted and are tallied on their own, leaving a
decoder figure of 69.7% over 916.

## What the landed record says under all this

**The band was an artefact of the bisection, and it did not survive.** The
withdrawn 1% was read off eighteen population × lossy-filter rows whose medians
fell into fourteen from 0.000011 to 0.001966 and four from 0.047161 to 0.898821
— a factor of **24** of empty band with 1% inside it. Measured per channel over
the same two corpora, the 21 direct-bucket rows that had anything differ are a
continuum: the largest gap anywhere in them is a factor of **6.8**, and where
fourteen of eighteen used to sit below 1%, **two of twenty-one do**.

**And those two are the argument against ever putting the threshold back.**
Their peak medians are **18 and 23 levels**, nine and eleven times the gate.
Under the bisection the low group was rounding *by construction* — a pixel one
level from luminance 128 could flip it. Under a magnitude measure the low group
is **sparse gross error**, so a 1% budget would now forgive pictures wrong by up
to 81 levels on a few hundred pixels. That is Cairo's *"otherwise some problems
could be masked"* in this corpus's own numbers, and it is why `N` stays at 0.

**The measure separates the two lossy filters, which the bisection could not.**
`JPXDecode` and `DCTDecode` used to read 15.4% and 33.2% and looked like two
versions of one problem:

| filter | compared | exact | **identical** | agreement | was |
|---|---:|---:|---:|---:|---:|
| `JPXDecode` | 1225 | 1215 | **7** | **99.2%** | 15.4% |
| `DCTDecode` | 430 | 146 | **4** | **34.0%** | 33.2% |

JPEG 2000 agrees within two levels almost everywhere and is bit-equal almost
nowhere, which is what a conformant lossy decoder looks like. **JPEG did not
move**, so it differs from poppler by more than the ISO/IEC 10918-2 IDCT
allowance on two thirds of what it was compared on — by 16 to 62 levels in most
populations. That is a specific finding about a decoder, and it is the first
one this repository has produced; the two rates are not comparable to one
another and neither is subtracted from the old one.

**And one thing the record says about itself.** `refused` is 4 across 3280
documents, all in `ia-biodiversity`, and all four are `render` v0.20.0's own
256-megapixel decode budget declining a page rather than a document we cannot
read. That is the other half of the fix that let the population run at all: it
bounded the decode, and a bound that fires reads here as a refusal. The
instrument folds "cannot read" and "declined to decode" into one count and
should not; the four are named in the baseline so nobody reads them as a
coverage gap.

The rest — every population, every filter, the ordered medians — is in
[`baseline/README.md`](baseline/README.md).

## What it comes to today

A number that is not written down cannot be regressed against. `baseline/`
holds a whole run of `images` over both corpora — the counts per population per
filter, and beside them the corpus, the poppler that judged it, **the gate the
comparison used**, every module version it was built against and when it was
taken, because a figure that drops between two runs means a regression only if
everything else held.

```
images -dir /Users/Shared/pdfscans -only ia-medical -json
```

[`baseline/README.md`](baseline/README.md) reads it out.

### Every population ran, including the three that never had

| population | before | now |
|---|---|---|
| `pdfscans-ia-biodiversity` | killed at 24.9 GB, `rc=137` | completed, peak 3.9 GB |
| `pdfforms-gh-openpdf` | killed at 27.5 GB, `rc=137` | completed |
| `pdfscans-ia-americana` | hit the 45-minute cap, `rc=124`, cause unknown | completed, peak 5.2 GB, in 53 minutes |

The two allocation failures were `render` v0.19.0 walking a page's resources as
a tree when they are a graph; **v0.20.0's fix is confirmed on both**. The third
was never a defect — a slow population of large scans and a cap that was too
short — and that open question is closed.

**Both seams are behind these records, not in front of them.** They were taken
at `render` v0.20.0 with the per-channel measure, so all 23 populations are one
instrument and one library version, and nothing under `baseline/` is inherited
from the ink bisection at v0.19.0 any more.

## How it is checked

Exact 100% statement coverage including every error branch, `go vet`, `-race`,
and nine cross-compile targets. Nothing outside the standard library.
