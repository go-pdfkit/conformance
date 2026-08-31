// Package poppler runs the implementation that judges ours, under a bound.
//
// It exists because of one document. pdfimages hangs for ever on
// pdfforms/gh-qpdf/qpdf_qtest_qpdf_shared-unnamed-field.pdf — 2496 bytes — and
// so do pdfimages -list and pdfinfo; two sweeps of this repository stalled on
// it and had to be killed by hand (conformance#21). That is a property of the
// judge rather than a defect of ours, and the consequence is what matters: a
// hang and a slow job are indistinguishable from outside, so a sweep that
// stops dead at document 900 of 2268 is waited on rather than investigated,
// and the wait has no end.
//
// So every invocation of a poppler tool in this repository goes through Run,
// and a document that exceeds the bound is recorded BY NAME with the tool that
// hung, never dropped and never retried. A named timeout is data about the
// corpus; a silent one is a gap a reader cannot tell from a bad score.
//
// The bound lives here rather than in each caller so that there is one setting
// and one reason for it, and so that a new caller cannot be written without
// one.
package poppler

import (
	"context"
	"os/exec"
	"time"
)

// Timeout is how long a poppler tool may take on one document before it is
// called a hang.
//
// It is NOT calibrated from timings, and that is deliberate twice over. The
// machine these runs are made on is shared, so a duration measured on it
// measures the other job as much as this one; and a bound read off a loaded
// machine would fire on documents that are merely large. So it is set far
// above any plausible handling of a single page, which makes a firing mean a
// hang rather than a slow document — and every firing is named with the tool,
// so a reader can judge that rather than take the bound on trust.
//
// It is a variable because a corpus of larger pages may want a larger one:
// both commands take a -timeout flag, and a baseline record carries the value
// its run used, beside the gate, because it is part of the instrument.
var Timeout = 2 * time.Minute

// Run runs one poppler tool and says whether it was the bound that stopped it.
//
// CommandContext kills the process when the deadline passes, and the deadline
// having passed is read off the CONTEXT rather than off the error, because a
// killed process reports a signal and a tool that merely failed reports a
// status — and telling a hang from a refusal is the whole point.
//
// The output is returned for the callers that want it; the ones that only
// want to know whether it worked discard it.
func Run(name string, args ...string) ([]byte, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return out, ctx.Err() == context.DeadlineExceeded, err
}

// DidNotFinish is what a hang is reported as, in the words the bound is set in.
func DidNotFinish(tool string) error {
	return &hang{tool: tool}
}

// A hang is a tool that did not answer, as an error.
type hang struct{ tool string }

func (h *hang) Error() string {
	return h.tool + " did not finish within " + Timeout.String()
}
