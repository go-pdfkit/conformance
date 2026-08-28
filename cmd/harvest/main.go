// harvest extends a corpus of real PDFs from the Internet Archive.
//
// A prevalence is only as honest as the population it was measured over. The
// go-pdfkit corpora are government forms and arXiv figures, and between them
// they hold almost no JBIG2 and almost no JPEG 2000 — so a figure like "six
// files in sixteen hundred use JBIG2" says what is in those two populations,
// not what is in the world. Mass digitisation is where a scanned page lives,
// and a scanned page is a fax.
//
//	harvest -dir /Users/Shared/pdfscans -origin ia-americana \
//	    -query 'collection:americana AND format:"Text PDF"' -want 250
package main

import (
	"os"
)

// osExit is a variable so the tests can reach the exit path without ending the
// test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }
