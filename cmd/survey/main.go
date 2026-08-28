// survey says what a corpus is actually made of, one population at a time.
//
//	survey -dir /Users/Shared/pdfscans
//	survey -dir /Users/Shared/pdfforms -only fr-cerfa
//
// A prevalence is per population or it is not a prevalence, so the manifest's
// origin is what the counts are grouped by. Where a corpus has no manifest, the
// top-level directories are taken as the populations.
package main

import "os"

// osExit is a variable so the tests can reach the exit path without ending the
// test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }
