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

## What is compared, and what "exact" asserts

**The comparison reduces every pixel to one bit.** `difference` in
[`images/images.go`](images/images.go) asks of each pixel, on each side,
whether it is *ink* — alpha at least 128, luminance below 128 — and counts the
pixels whose answer differs. So `Share` is **not** the fraction of pixels that
differ. It is the fraction whose black/white classification differs, and
**`exact` means no pixel's classification differs.**

For a **bilevel** picture those are the same claim. A one-bit stencil, a CCITT
page, a JBIG2 page hold nothing the bisection can lose, so `exact` there is
bit equality, and the 100% both JBIG2 columns read means what a reader takes
it to mean.

For a **greyscale or colour** picture it is strictly weaker, and weaker in one
particular direction. A decoder that renders every pixel of a scan at
luminance 120 where poppler renders 20 scores **0.000** — perfect agreement —
because both sides are ink. That is an error of 100 levels on every single
pixel, and this instrument cannot see it. **A systematic level or chroma shift
is the characteristic failure of a lossy decoder**, so on `DCTDecode`,
`JPXDecode` and `(samples)` the measure is blind in exactly the direction
those codecs fail.

Every published figure that comes from comparing pixels inherits that.
`exact`, `agreement`, `differing`, `median`, `worst` and `inverted` are
statements about ink classification, bit equality only where the picture is
bilevel. The counts that do not compare pixels — `documents`, `refused`,
`unopenable`, `declined`, `unmatched`, `remapped` — are unaffected, so
**`refused` is 0 in the full sense**, and so is the reading that nothing in
the corpus is refused that poppler can read.

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
(`index.js:153`). Our `Share` is a count over a predicate with no severity in
it at all, which is the one construction all of them independently avoid.

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

## What should replace it

The rule this repository should adopt, and the reasons are ours rather than
borrowed:

> **Compare per channel. A pixel counts as differing only when some channel
> differs by more than `D`. A picture agrees when at most `N` such pixels
> differ, and when the aggregate error is within bound. `D` and `N` default to
> 0 and are raised per case with a recorded reason.**

**`D` should be 2, not pdfium's 3.** pdfium compares *rendered pages*, so its
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
PEAK limits are **0**, per OpenJPEG's `tests/conformance/CMakeLists.txt:309`,
so JPX is required exact on most conformance files and permitted a small
bounded error on a few.

**Whatever a per-case exception is raised to, 25 is a ceiling it must not
cross**, for Cairo's stated reason.

**An aggregate term is warranted, and it should be two terms.** pdfium found
the mirror of our defect — a per-pixel gate with an unbounded count, where
many small forgiven differences accumulate — and answered it with a
mean-squared error (`testing/utils/pixel_diff_util.h:11-12`,
`testing/image_diff.cpp:171-183`). We should take that, and take a **signed
mean** with it, as FFmpeg does at `dct.c:259` where `fabs(ome) > 0.0015` sits
beside `omse > 0.02`. The two catch different things: MSE catches accumulated
noise, the signed mean catches *bias*. Bias is the failure this instrument is
documented above as being blind to, so it is the term we most specifically
need, and it is cheap.

**`N` should default to 0.** That is the field's default wherever a count
budget exists at all, and pdfium — doing our exact job — requires its
percentage to be zero. A nonzero `N` should be per population and per filter,
each one carrying a written reason, as pdfium's 142 scoped entries in
`testing/SUPPRESSIONS_EXACT_MATCHING` do and as pdf.js's nine
`knownPartialMismatch` tests do. **If `N` is ever raised, it should be a count
within a window rather than a share of the whole picture**, so that the
threshold does not loosen as the picture grows.

**`Inverted` must survive the change.** `Share == 1` detects that ours and
theirs are exact complements, which is the stencil polarity convention and not
a disagreement; a magnitude measure would report it as a maximal error and
lose the distinction. Whatever replaces `difference`, polarity stays its own
signal.

The design and the re-measurement it requires are
[conformance#16](https://github.com/go-pdfkit/conformance/issues/16). This
document states the rule and does not yet claim it: **no per-channel figure
has been measured**, and every number below is the ink-classification measure.

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
fuzzy", not "is the source data lossy".** So the replacement is one rule,
above, with `D` and `N` at 0 for everything and raised per case. `DCTDecode`
and `JPXDecode` then reach `D = 2` as an exception that carries the ISO
citation as its recorded reason — the same outcome the split produced, arrived
at the way the evidence supports, and reversible if a measurement ever says
otherwise. It also stops the split doing harm in the other direction. Under
it a lossless filter can never be granted anything however good the reason,
and `(samples)` is a live candidate: its 84.9% is two implementations doing
different ICC conversion read through a bisection — `fr-cerfa` differs on 535
`(samples)` pictures at a median of 0.286 — which is not a decode defect. What
that needs is not a wider count budget but comparison per channel, with
colour-converted pictures counted in their own column; the split forbids even
asking.

## What the landed record says under all this

The band analysis that produced the 1% is kept below rather than deleted,
because it is the record of how the number was reached and a withdrawn
derivation is still evidence about the corpus. **It is a statement about the
ink-classification statistic and about nothing else.**

`baseline/` records, per population and filter, the median and worst of the
differing shares. Eighteen rows are a lossy filter with something to say, and
sorted by median they fall in two groups: fourteen from 0.000011 to 0.001966,
then nothing, then four from 0.047161 to 0.898821 — a factor of 24 of empty
band. One population was re-measured picture by picture, `fr-cerfa` at
`render` v0.20.0, and the gap is there too: 100 of its 105 differing
`DCTDecode` pictures fall between 0.00000046 and 0.0086996, then 0.094990 and
four larger.

**Four fifths of that tail is not evidence.** Each of the 105 was checked
against what `pdfimages -list` says its page holds, because
[conformance#13](https://github.com/go-pdfkit/conformance/issues/13) showed
`match` manufactures disagreements when a page draws many pictures of one
size:

| | pictures | ≤ 0.0087 | above 1% |
|---|---:|---:|---|
| **pairing unambiguous** | 95 | 94 | **0.094990** |
| pairing ambiguous | 10 | 6 | 0.250000, 0.375000, 0.571429, 0.750000 |

All four largest are 7×1 and 8×1 slivers on page 1 of `cerfa_12626.pdf`, where
`pdfimages` extracts 2122 pictures of size 8×1 and 81 of size 7×1, so they are
paired at random and their shares are two to six differing pixels out of
eight. **`fr-cerfa`'s worst `DCTDecode` disagreement is 0.095, not the 0.750
that `baseline/` records.** The other five populations' worst pictures have
not been audited for pairing.

**No confirmed decode defect is known in this corpus.** The two concentrations
that looked like candidates are not: of the 173 `(samples)` complements in
`fr-cerfa`, 144 are this instrument's own matcher and 29 are `pdfimages`
writing a one-bit `/Indexed` picture black only for an exactly `#000000`
palette entry; the 28 in `us-opm` are the stencil convention, where poppler's
own `pdftoppm` agrees with `go-pdfkit` against `pdfimages`. What is left above
the band is a set of **unexplained disagreements**, which is not the same as
defects, and nobody has established that any of them is ours to fix.

**And the record cannot be recomputed under any tolerance.** It holds, per
filter, the exact count and the median and worst of the shares that differed.
A median says half, so half of `ia-medical`'s 426 differing JPX pictures is
all it can tell about how many sit under any cut. Eleven of the eighteen lossy
rows have a *worst* below 1%, so all 81 of their differing pictures would be
inside; four have a *median* above it; three are split in a proportion the
record does not give. **Recording the count under the threshold is something a
run must do, and the run that does it should be the one that measures per
channel.**

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
