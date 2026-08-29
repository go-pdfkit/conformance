// images takes each picture out of a corpus twice — with go-pdfkit and with
// poppler — and says, per image filter, how many came out identical.
//
//	images -dir /Users/Shared/pdfscans -only ia-medical -pages 2
//
// A page is a composition, and a composition hides its parts: a picture that
// is wholly wrong can move a page's number by a few percent and pass. This
// asks about the pictures.
package main

import "os"

// osExit is a variable so the tests can reach the exit path without ending the
// test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }
