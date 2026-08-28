// Package conformance is where go-pdfkit is judged by implementations that are
// not its own.
//
// A file our own reader reads back perfectly can draw nothing anywhere else —
// that has happened here, and it is why the tools in this repository ask
// poppler rather than asking ourselves. What they need in order to ask is a
// corpus, and a corpus is only as honest as the population it was drawn from:
// government forms and arXiv figures between them contain almost no JBIG2 and
// almost no JPEG 2000, so a prevalence measured over those two says what is in
// them, not what is in the world.
//
// So the corpus is a thing this repository builds and records, not a directory
// somebody happens to have.
package conformance
