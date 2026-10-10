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

**And it checks, because the sentence above promised something nothing read
back.** "What makes a number reproducible and lets a document that changed
underneath be noticed" was true of what the manifest RECORDS and false of what
anything here did with it: the hash was written and never compared, and a file
in no row was invisible to every tool in this repository.

```
$ harvest -check -dir /Users/Shared/pdfscans
ia-americana/calcflh_000254.pdf: on disk and in no manifest row
ia-americana/epn11-1968countclip2restricted.pdf: on disk and in no manifest row
ia-biodiversity/bidragtillknne38suom.pdf: on disk and in no manifest row
ia-medical/b22346703.pdf: on disk and in no manifest row
4 disagreement(s)
```

Four files, and **three of them are documents our reader refuses** — which is
the whole of why `images` and `compare`, which walk the manifest, count 63
refusals where a sweep that walks the directory counts 66. Two true numbers for
one corpus and no way to tell which a figure came from. See
[`baseline/README.md`](baseline/README.md) §33.

It reports a row with no file, a file that will not open, a size or a digest
that moved, and a file in no row; it exits non-zero when a corpus disagrees, so
it can be the first line of a measuring script rather than something to
remember. `/Users/Shared/pdfforms` comes back clean — all 2 268 rows — and so
does every one of `pdfscans`'s 1 012, **so no document has changed underneath**.

The digest is compared over **as many characters as the manifest recorded**. The
forms corpus, gathered before this repository existed, keeps sixteen of them
under the header `sha256-8`; the scans corpus keeps all sixty-four. Sixteen hex
characters is sixty-four bits, which answers *did this file change* perfectly
well — and a check that demanded all sixty-four would report every one of those
2 268 rows as changed and be deleted by the first person who ran it.

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

## Every reader, not one: `judges`

`compare` and `images` ask poppler how well we read the world's files. `judges`
asks the other direction — how well the world reads **ours** — and asks
**everyone the machine has**. A PDF that is the smallest, or that our own reader
reads back perfectly, is worth nothing if one reader in the field disagrees with
the others about it. That is a claim every producer in `go-pdfkit` —
`html2pdf`, `render`, `ops`, `gotex` — has to be able to make, which is why the
harness lives here and not in the producer that first needed it.

```
judges -pdfs 'out/*.pdf,out/bench/*.pdf' -pdfium /path/to/pdfium_test \
       -out out/judges -report JUDGES.md -results judges.json
```

| judge | what it is | gives |
|---|---|---|
| `qpdf --check` | structural validator | clean / warnings / errors |
| poppler (`pdfinfo` / `pdftoppm` / `pdftotext`) | the reference the per-judge Δ is taken against | pages, text, renders |
| MuPDF (`mutool`) | independent parser and rasteriser | pages, text, renders |
| Ghostscript (`gs`) | PostScript-lineage interpreter | text, renders |
| pdfium (`pdfium_test`) | Chrome's engine — `-pdfium /path` or `PDFIUM_TEST` | pages, text, renders |
| pdf.js (`judges/pdfjs-*.mjs` under node) | Firefox's engine | pages, text, renders |
| Quartz (`sips`) | macOS ImageIO — Preview's engine | page-1 render |

A judge whose binary is absent is **skipped with a note, never faked**. Each
cell reads `pages · text ratio · Δworst (page)`: the pages the judge reports,
its extracted text as a ratio of poppler's, and the largest distance of its
renders from poppler's over a sample of pages — the first, the middle and the
last, at 96 dpi, as the share of pixels whose grey level moves by more than 48
of 255 after both are box-downsampled to 400 px wide — with the page it
happened on. `⚠n` is n lines the judge complained on; `❌` is a judge that
would not process the file, with the first thing it said.

Four things are decided rather than defaulted.

**Poppler is the reference, and the `consensus` column is why it is not the
truth.** A distance needs a second point and poppler is the reader every
machine this runs on has; but poppler is one reader, so each sampled page also
carries the mean pairwise distance between *all* judges' renders of it, no
reader privileged. A page every reader draws differently is a page to look at,
whichever one is "right".

**Text is counted without its whitespace.** Judges disagree wildly on it for
reasons that are not about the file — Ghostscript's `txtwrite` pads lines to
reproduce the column layout, pdfium separates every glyph run — while the
glyphs they *recover* are what the comparison is about. And pdfium's `--txt` is
UTF-32LE with a byte-order mark, four bytes a character, verified with `xxd`:
it is decoded before it is counted, or it reads as four times the text.

**Quartz is composited over white before it is compared.** `sips` renders a
page on a transparent background, and a transparent pixel converted straight to
grey is black: every Quartz render would read as a 99% mismatch against an
opaque one. `sips` has no page selection either, so Quartz is judged on page 1
only and its cell says so.

**A judge's narration is not a warning.** `pdfium_test` says "Processing PDF
file x." and "Processed N pages." on stderr as it goes; those lines are
progress, and are struck before the rest is counted.

Judge a **control** beside your own output. `html2pdf` judges Chrome's PDFs of
the same pages alongside its own: a judge that disagrees on the control too is
judge noise, and one that disagrees only on ours is a defect. The table is
written above an `<!-- BEGIN ANALYSIS -->` marker, and whatever a reader writes
beneath it — which cells were noise, which were defects, where they were fixed
— survives the next run.

Every judge runs under `-timeout` (three minutes per judge per file), and one
that does not answer is reported as `hung`, by tool, rather than as the first
warning it printed before it was killed. The reason is the next section's.

pdf.js is two node scripts under [`judges/`](judges/); they are not Go, and CI
does not build them. Once, on the machine that judges:

```
cd judges && npm ci        # pdfjs-dist + @napi-rs/canvas, from the lock
```

`-nodedir judges` (the default) then finds them, and a machine without them
simply has no pdf.js column. From another repository:

```
go run github.com/go-pdfkit/conformance/cmd/judges@latest \
    -pdfs 'out/*.pdf' -nodedir /path/to/conformance/judges -pdfium "$PDFIUM_TEST"
```

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

#### A bound measured for extracted pictures, at last

The invitation above is answered by the run recorded in
[`baseline/`](baseline/README.md), and the answer is a **derivation** rather than
a borrowed constant.

**Every direct bucket that differs across the 23 populations is `DCTDecode`**,
with no exception any more, and each sits inside a narrow band:

| | peak | `mse` | `mean` |
|---|---:|---:|---:|
| the 13 differing `DCTDecode` direct buckets | 3 to **4** | 0.111 to **0.654** | −0.58 to +0.38 |

**The exception this table used to carry is gone.** It read
`gh-qpdf/(samples)`, 2 pictures, peak **32**, `mse` **227.6**, `mean` **exactly
0.0000** — and a mean of exactly nought beside a peak of 32 is what comparing
two DIFFERENT pictures looks like. It was the known mis-pairing:
`qpdf_qtest_qpdf_form-xobjects-no-resources-out.pdf` draws four 15×15 grey
pictures and the name `Im1` reaches two of them, so the ambiguity rule refused
an identity and the size fallback drew from a hat.

`render` v0.26.0 publishes each picture's object number, the ambiguity rule is
retired, and those two pictures now pair correctly and come out **exact**:
`gh-qpdf` reads 41 of 41 rather than 39 of 41, and its agreement 95.2% rather
than 92.1%. The bound below is therefore over the DCTDecode band alone, which
is what it was always trying to be.

**And the ISO bound does not transfer, but its arithmetic does.** FFmpeg's
`dct.c:259` fails an IDCT when `err_inf > 1 || omse > 0.02 || fabs(ome) > 0.0015`,
all measured per output sample against a FLOAT reference. Two implementations
each within `omse` 0.02 of that reference differ from each other by at most
(√0.02 + √0.02)² = **0.08**. Our `DCTDecode` band reaches **0.654**, eight times
that — and the excess is not non-conformance. It is composition: a three-
component picture puts three IDCTs and a YCbCr-to-RGB mix between the
coefficients and the pixels, and the mix has coefficients up to 1.772. The
baseline's §14 measures the same thing from the other side: a ONE-component
JPEG, which is one IDCT and nothing else, has **not one picture of 36 outside the
gate**, at a worst peak of 1.

So the honest bound for this instrument, on extracted pictures, is **`mse` at
most 0.66 and `|mean|` at most 0.58 for a composed colour picture, and ffmpeg's
0.02 for a single grey IDCT** — and the two differ by the composition rather than
by anybody's conformance. Neither is made a pass criterion here: `peak` remains
the criterion, and these are now recorded WITH a measurement behind them instead
of as an open question.

The same reading is why `D` = 2 cannot be defended for a colour picture on the
standard's authority. It is derived from `err_inf > 1` — each implementation
within one level of the reference, so two of them within two of each other — and
that is a statement about ONE IDCT. Applied to three plus a colour mix it is
tighter than its own derivation supports, which is why 217 pictures differ at a
peak of 3 or 4 and none at more. **Changing it would move every figure in the
baseline, so it is stated here and not acted on.**

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

> **We make that conversion now.** `render` v0.27.0 reads `CalGray` and
> `CalRGB` rather than treating them as their device namesakes, so the phrase
> *"a colour conversion we did not make"* describes the instrument's history
> and not its present. The bucketing rule is unchanged and still right — the
> two sides are still answering different questions for `ICCBased`, where
> poppler consults a profile through little-cms and we do not. The baseline's
> §16 separates the two cases and measures what reading the calibrated spaces
> was worth: the worst `DCTDecode converted` peak falls from 110 to 33.



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
pictures counted apart, and it now has both: 3084 of `(samples)`'s 4292
pictures are colour-converted and are tallied on their own, leaving a
decoder figure of 69.7% over 915.

## What the landed record says under all this

**Everything below is at `render` v0.21.0**, which put a JPEG's chroma back the
way every other reader does. The landed baseline is now at v0.28.0; where this
section says something about `/CalRGB` or about the pairing, read the baseline's
§15 and §16 for what became of it. The figures the previous revision of this file
carried were at v0.20.0 and are quoted here only as the previous run's.
[`baseline/README.md`](baseline/README.md) is the whole of it; this is the part
that changes what this document claims.

**The improvement was being quoted as 61.2% → 99.3% corpus-wide. It is not
that, and the correction is not a smaller improvement — it is a different
quantity.** That figure is `DCTDecode` **alone**, at a **gate of 4**, over the
559 pictures `pdfimages -list` calls `gray` or `rgb` — a denominator
[conformance#20](https://github.com/go-pdfkit/conformance/issues/20) showed
admits `/CalRGB`, so it does not exist in this repository any more. Measured
with the landed instrument at gate 2 over all 23 populations:

| | v0.20.0 | v0.21.0 |
|---|---:|---:|
| `DCTDecode` agreement | 34.0% (146 of 430) | **43.2%** (184 of 426) |
| whole fleet | 84.5% (3347 of 3962) | **85.5%** (3385 of 3957) |

**The rate understates it, because the gate is a cliff and what changed was
magnitude.** The per-population medians of the worst channel on the direct
`DCTDecode` rows:

```
v0.20.0   16  18  23  23  26  31  34  47  50  58  62  171  244  255      (14 rows)
v0.21.0    3   3   3   3   3   3   4   4   4    4      171  233  255     (13 rows)
```

Ten of thirteen rows now sit **one or two levels above a gate of two**, where
eleven of fourteen used to sit between 16 and 62; the row that vanished is
`uk-govuk`, 11 differing pictures to none, **92.3% → 100.0%**. The two lists are
each sorted, so they are **not** paired population for population;
[`baseline/README.md`](baseline/README.md) pairs them. **Three rows stayed
gross** — `gh-qpdf` 171 → 171, `fr-impots` 255 → 255 and `gh-pypdf` 244 → 233 —
and none of the three is chroma reconstruction. **That is the `DCTDecode`
finding now, and it is a much smaller thing than it was.**

**And the only filter that moved is the only filter the change touched.** Every
other filter's `exact` count is identical picture for picture — 638, 1074, 1215,
12, 10, 250, 2 — and so is the fleet's 1990 bit-identical; the only denominator
that moved is the one `(samples)` picture #20 reclassified. That is a check on
the instrument as much as on the library.

**The fleet figure is over direct-comparable pictures**, first page of each
document, converted bucket excluded by construction — and it spans three changes
rather than one, since the denominators differ by #20's five moved pictures and
`gfx` moved v0.16.0 to v0.19.0 as well. Holding the bucketing fixed, v0.21.0
reads `DCTDecode` **42.8%** and the fleet **85.4%**; the rest is the instrument
declining to score four pictures it should never have scored.

**JPEG 2000 is still what a conformant lossy decoder looks like**: 1215 of 1225
within two levels, **7** of them bit-equal. It agrees almost everywhere and is
identical almost nowhere, which is what ISO/IEC 15444 promises — the transform
is specified, the rounding is not.

**The band came back, and the previous run's argument against it did not
survive its own library.** That run found the empty band gone: 2 of 21 direct
rows under 1%, largest gap 6.8, and the two low rows carrying peak medians of
**18 and 23** — *"sparse gross error"*, it said. At v0.21.0 the twenty direct
rows have **ten under 1%**, a gap of **64.5** above them, and the low group's
peak medians are **3 and 4**. What made that group "gross" was the chroma
defect. **It does not put the 1% back and nothing here proposes to**: with `N`
at 0 the criterion is the peak and the share is a report, so a band in the
share is evidence about `D` and not about `N`. It is recorded rather than
spent.

**The colour-space rule moved five pictures.** Exactly five, and it is
measurable rather than asserted: the per-filter `pictures` and `unmatched`
counts are **identical filter for filter** across the two runs, so every
difference in the direct/converted split is that rule and nothing else. Four
are `DCTDecode` — the four `/CalRGB` pictures the issue named — and one is
`(samples)`. `calibrated`, at 1802, is a different quantity and is not that
count.

**And one thing the record says about itself.** ~~`refused` is 4 across 3280
documents, all in `ia-biodiversity`, and the same four documents as before~~ —
**it is 1**, in the records re-taken 2026-10-04. One document of 3 280:
`bulletinno38tasm.pdf`, whose page names three 9 449 × 13 701 pictures against
a per-page budget the `Images` path spends four bytes a pixel of. `render.Page`
**draws** that page; it is the extraction API that refuses it. The baseline's
§31 has the measurement and
[go-pdfkit/render#100](https://github.com/go-pdfkit/render/issues/100) holds the
question.

~~The instrument folds "cannot read" and "declined to decode" into one count and
should not.~~ It no longer asserts the other half either: `Ours` is documented
as "ours produced nothing and the judge did", and the page path used to set it
**without asking poppler at all**. It asks now — pictures back is `Ours`, a
tool that will not finish is `Hung`, a refusal is `Neither` — and the first
thing that changed was a test demanding `Ours` for page nine of a one-page
document. See §31.

~~**`hung` is 0 in all 23 populations, and that is not "nothing hangs".**~~
**`hung` is 1, and the reason it used to be 0 is the reason given here.** The
document that hangs draws no picture on its first page, so `images` never asked
poppler about it at all — and §35 made it ask, on exactly the pages we draw no
picture for. `pdfimages` does not come back from
`qpdf_qtest_qpdf_shared-unnamed-field.pdf`, and §34's page comparison reached
the same file, page and tool independently. The sentence above was right and it
described a hole rather than a corpus.

~~The bound is unexercised by `images` on this corpus~~ — **§22 of the baseline
strikes that too: the bound fired three times, and it is a property of the
MACHINE and not of the corpus.**

## How much of another toolkit we actually do: `bentoparity`

The same question as the rest of this repository, asked of a feature list
rather than of a page: **what can be shown**, by something that is not our own
say-so.

```
bentoparity -tools ~/src/bentopdf/docs/tools -pdfops ./pdfops -src ~/src/go-pdfkit
```

```
covered       72
not covered   42
parity       63.2% (72/114), every claim verified
```

⛔ **It refuses to report a figure if a single claim cannot be checked.** A
parity number assembled from a mapping nobody verifies is a vibe with a decimal
point in it. The previous version of this measurement was kept by hand, was
wrong by under-counting, and the only reason anybody found out is that somebody
said *"not sure that map is up to date"*.

### Neither end of the fraction is anybody's recollection

| | |
| --- | --- |
| the **denominator** | read from BentoPDF's own `docs/tools` directory, one Markdown file per tool. A list typed out here would be a snapshot of what somebody believed the day they typed it — and the denominator is exactly what must not quietly drift. |
| a **verb** claim | has to appear in `pdfops --help`, read off a binary compiled from the source being measured. Not a README, not a note. |
| a **symbol** claim | has to appear in that repository's **Go source**. A README saying a function exists is a claim, not the function. |

A tool with no claim counts as **not covered**. Absence is the default and the
conservative direction: this must never make the fleet look better than it is.



### ⛔ The figure was right, the verification was not, and a stale clone invented a defect

This README published **"parity 60.5% (69/114), every claim verified"**. Re-running
`bentoparity` before a merge, it **refused**:

```
no figure reported: an unverified claim is not a capability
```

Twelve `verb:` claims named verbs that were not in `pdfops --help` — `poster`,
`outline` ×2, `resize`, `move`, `onepage`, `interleave`, `metadata` ×2,
`attach`, `detach`, `attachments`.

**Two separate faults, and they had been hiding each other.**

| | |
| --- | --- |
| the figure was **arithmetic** | `69` is also the number of `{Tool:` rows in the table. A total kept by counting rows agreed exactly with one the tool would not print, so no amount of re-reading the number could reveal it — only re-running the instrument did. ⛔ A total that matches **for the wrong reason** is the hardest kind of wrong to see. |
| the instrument read a **37-day-old** checkout | the clone the binary was built from sat 37 days behind `origin/main`, and `git pull` had failed silently on it — no upstream tracking, the error on stderr inside a chain whose output was tailed. The missing verbs had been added in the meantime by a PR titled *"Seven verbs over library functions that were already written and tested"*. |

So the first conclusion — *"the CLI is twelve verbs behind the library"* — was
**wrong, and it was filed as a bug against `go-pdfkit/ops` before being
checked**. The up-to-date binary has every one of the twelve. The claims are
`verb:` claims again, because a verb claim asks a **compiled binary** and is
the stronger of the two forms.

What the figure in this README now carries, and did not before: it is **pasted
from the tool's own output**, together with **the commit of every repository it
was measured against** — which is the only thing that would have caught the
stale clone.
### Three refusals before it counts anything

- **a tool claimed twice.** Silent otherwise: the map keeps the last entry and
  the total comes out short of what the table appears to say. The check exists
  because ten entries were added and the figure rose by **nine**.
- **a claim for a tool the other project does not have.** It inflates nothing by
  itself, but it means the table is being written against something other than
  the list, and that mistake has a direction nobody notices.
- **a claim that cannot be shown.** An unverified claim is not a capability.

And one in the other direction: a repository that is checked out but
**unreadable** is reported as unreadable, never as *"does not contain X"*. That
false negative reads exactly like a real gap, and a capability measurement must
not fail in the direction that invents one.

### Measured against

BentoPDF **v2.8.8**, commit `3a5f146`, 2026-10-03 — **114** tools.

## What it comes to today

A number that is not written down cannot be regressed against. `baseline/`
holds a whole run of `images` over both corpora — the counts per population per
filter, and beside them the corpus, the poppler that judged it, **the gate the
comparison used**, **the bound the judge was held to**, every module version it
was built against and when it was taken, because a figure that drops between
two runs means a regression only if everything else held.

```
images -dir /Users/Shared/pdfscans -only ia-medical -json
```

[`baseline/README.md`](baseline/README.md) reads it out, and
[its §33](baseline/README.md) says how the whole set is re-taken and what it
costs — one population at a time, because the judge is held to a wall-clock
bound.

### Three counts about the measurement rather than about a picture

Added 2026-10-04 with the records above, and each of them is a number the
instrument needed to say about ITSELF.

| | |
|---|---|
| `sizePaired` | how many pictures were matched to the judge's by SIZE rather than by object number. The weaker matcher, so a large share is a run whose other numbers are worth less. It is **0** here, and §32 says why that zero was not the good news it looked like. |
| `unseen` | how many pictures the judge took out that ours produced nothing for — **the one direction the pairing could not see at all**, because it walks our pictures and appends one result for each. **5 510** over 23 populations; §32 names the three causes, §35 is the 29% of them that a page we draw NOTHING for used to hide, and [render#101](https://github.com/go-pdfkit/render/issues/101), [#104](https://github.com/go-pdfkit/render/issues/104) and [#108](https://github.com/go-pdfkit/render/issues/108) hold them. |
| `repeated` | how many of the judge's rows name an object we returned once. `pdfimages` lists one row per DRAW and `Images` returns one entry per object, so this is a difference of **unit** and not of fidelity. **3 371**, and counting it apart is what keeps `unseen` meaningful. |

All three are `omitempty`, so a record written before them keeps its shape.

### Every population runs, and has since v0.20.0

| population | at v0.19.0 | at v0.20.0 | at v0.21.0 |
|---|---|---|---|
| `pdfscans-ia-biodiversity` | killed at 24.9 GB, `rc=137` | completed, peak 3.9 GB | completed, peak **3.6 GB** |
| `pdfforms-gh-openpdf` | killed at 27.5 GB, `rc=137` | completed | completed |
| `pdfscans-ia-americana` | hit the 45-minute cap, `rc=124`, cause unknown | completed, peak 5.2 GB | completed, peak **5.9 GB** |

The two allocation failures were `render` v0.19.0 walking a page's resources as
a tree when they are a graph, and v0.20.0's fix has now held over two runs. The
third was never a defect — a slow population of large scans and a cap that was
too short.

**Every seam is behind these records, not in front of them.** All 23
populations were taken at `render` v0.21.0 with the per-channel measure, the
colour-space rule of
[conformance#20](https://github.com/go-pdfkit/conformance/issues/20) and the
bound of
[conformance#21](https://github.com/go-pdfkit/conformance/issues/21) — one
instrument and one set of library versions, with nothing inherited from the ink
bisection at v0.19.0 or from the chroma defect at v0.20.0.

## How it is checked

Exact 100% statement coverage including every error branch, `go vet`, `-race`,
and nine cross-compile targets. Nothing outside the standard library. The
readers `judges` shells out to are stood in for under test, so the whole
harness — every judge, every refusal, the hang — is exercised on a runner
that has none of them.
