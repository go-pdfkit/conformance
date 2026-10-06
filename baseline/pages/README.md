# The page records

**§34's twenty-three `pdfforms-*.txt` and `pdfscans-*.txt` are the whole corpus
compared against poppler**, which nothing in this repository had recorded until
2026-10-04: `compare` against `pdftoppm` at 72 dpi with `-cropbox`, one
population at a time, at the `go.mod` of the `.json` records beside this
directory. Read §34 for what they say and for the two things they are not --
their timing columns were measured under a load average between 7 and 59 and
are not to be read, and they are text rather than a machine-readable record.

## The two files §21 is measured from

These are `compare`'s output, unedited, for the control in §21: the same 250
documents drawn twice by **one binary built twice**, current in everything but
`gobig2`'s aggregate symbol-dictionary cap.

| file | cap | how it was built |
|---|---|---|
| `ia-biodiversity-16MP.txt` | 16 MP | `go mod edit -replace github.com/tannevaled/gobig2=<a worktree at v0.1.0>` |
| `ia-biodiversity-64MP.txt` | 64 MP | the released modules, as the `.json` records beside them |

`compare` compares **rendered pages**, where the `.json` records beside this
directory compare **extracted pictures**. §21 is about the gap between those
two instruments: a dropped ink layer is a picture that is *absent*, which the
picture instrument cannot report as a difference, and a page that is a fifth
wrong, which this one can.

The 16 MP file reproduces the previous run's figures for this population
exactly, which is what makes it a control rather than a second measurement.
