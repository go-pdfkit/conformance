// bentoparity says how much of BentoPDF's published tool list this fleet can be
// SHOWN to do, and refuses to say anything if a claim cannot be checked.
//
//	bentoparity -tools ~/src/bentopdf/docs/tools -pdfops ./pdfops -src ~/src/go-pdfkit
//
// The denominator is read from BentoPDF's own docs/tools directory and the
// verbs are read off a compiled binary's help output, so neither end of the
// fraction is anybody's recollection.
package main

import "os"

// osExit is a variable so the tests can reach the exit path without ending the
// test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }
