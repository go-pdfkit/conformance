# Reference timings

One row per page: how long each renderer took, and whether the two could be
compared at all. Written by

    compare -dir /Users/Shared/pdfscans -timings pdfscans.tsv
    compare -dir /Users/Shared/pdfforms -timings pdfforms.tsv

and read back by

    compare -dir /Users/Shared/pdfscans -against baseline/timings/pdfscans.tsv

which reports the pages that got slower. **These files are here because a
byte-identity sweep cannot see a slowdown**, and one hid here for three of them:
`render` v0.49.0 took a 256-entry colour-conversion memo away from any
single-component picture carrying a soft mask, and 86 pages of this corpus ran two
to twelve times slower with byte-identical output. §24 of `../README.md` tells that
story; this directory is what stops it happening quietly again.

## What these were taken with

| | |
|---|---|
| taken | 2026-09-27T14:19Z .. 2026-09-27T14:51Z (UTC) |
| `go-pdfkit/render` | v0.52.0 |
| `go-images/jpeg2000` | v0.8.0 |
| judge | pdftoppm 26.04.0 |
| resolution | 72 dpi, first page of each document |
| machine | Apple M4 Max, 16 cores |
| load | 15.53 at the start, 11.09 at the end |
| note | `pdfforms` is the SECOND of two takes. In the first, poppler drew nothing for 20 `gh-pdfbox` pages, which `compare` reported as its own note (`20 they drew nothing`) and excluded as not comparable; the re-run had none of them. A reference file is where that kind of accident would be hardest to notice later, so the clean take is the one kept. |

## What a row is not

**A single timing.** One measurement per page cannot separate a regression from a
scheduling accident, and the load above says the machine was not quiet. Two rows of
the report that produced these files were chased and were noise:
`cerfa_11055.pdf` came out at 20.98× and takes 16 ms in every release;
`indianhealthcare00unit_3.pdf` at 3.85× is 5% faster than it was.

Three thresholds were tried against that noise, and they were not enough. A ratio
alone offered 99 candidates where two were real. An absolute floor on the increase
(`-slower`, default 100 ms) removes the small-page accidents, because an accident
on a 16 ms page cannot add 100 ms; it cannot touch the large-page ones, and a run
of this corpus still offered three candidates on pages of 150 ms and up whose
re-measured ratios were 0.98×, 0.98× and 1.01×. Their increases cleared
every threshold the one real regression cleared. **No arithmetic on one pair of
numbers separates them.**

So `-against` draws each candidate again. `-confirm N` (default 3) re-renders only
the pages that crossed the thresholds and compares the *minimum* of those tries,
because an accident can only ever add time. The report marks each row:

| | |
|---|---|
| `!` | drawn again, still slower — believe it |
| `~` | drawn again, not slower — noise |
| (none) | nothing could draw it again; it is still a candidate, and no one has checked it |

The unmarked case is deliberate: a page that could not be re-measured is kept, not
dropped, because an answer nobody obtained must not read as a negative one.

`-confirm 0` reports the candidates unchecked, which is what this tool did before.
Replace these files only from a run you would be willing to defend.

## Columns

`population`, `document`, `page`, `ours_ns`, `theirs_ns`, `share`, `hung`.

`share` is the fraction of pixels that differ materially, or `-1` when the two
could not be compared. `-against` uses it to leave out pages whose comparability
changed: a page that was blank and now draws is slower and is not a regression.
`hung` is the tool the judge did not finish on, or `-`.
