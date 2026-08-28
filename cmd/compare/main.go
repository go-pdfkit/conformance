// compare draws each page twice — with go-pdfkit and with poppler — and says
// how far apart the two pictures are, and how long each took.
//
//	compare -dir /Users/Shared/pdfscans -only ia-biodiversity -pages 3
//
// We are not a fit judge of our own output: a file our own reader reads back
// perfectly can draw nothing anywhere else.
package main

import "os"

// osExit is a variable so the tests can reach the exit path without ending the
// test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }
