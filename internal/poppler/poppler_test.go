package poppler

import (
	"strings"
	"testing"
	"time"
)

func TestABoundThatFiresIsReadOffTheDeadlineAndNotTheError(t *testing.T) {
	// Both halves of conformance#21's claim have to hold on something that
	// really does not return: that it is STOPPED, and that it is SAID to have
	// hung. Neither can be taken on trust from a tool that always answers.
	//
	// It is not a poppler tool, deliberately. Poppler is absent from the
	// machine that runs these tests, and an absent tool fails instantly —
	// which would leave the branch that matters untested while looking
	// covered.
	was := Timeout
	defer func() { Timeout = was }()
	Timeout = 50 * time.Millisecond
	out, hung, err := Run("sleep", "30")
	if !hung {
		t.Fatalf("a tool that does not return was not called a hang: %v", err)
	}
	// The error alone would not have said so: a killed process reports a
	// signal, which is why the deadline is read off the context.
	if err == nil {
		t.Error("a killed process reported success")
	}
	if len(out) != 0 {
		t.Errorf("a tool that was killed produced %q", out)
	}
}

func TestAToolThatAnswersIsNotAHang(t *testing.T) {
	out, hung, err := Run("echo", "hello")
	if hung || err != nil {
		t.Fatalf("echo came back hung=%v, %v", hung, err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Errorf("its output was %q", out)
	}
}

func TestAHangSaysWhichToolAndWhatBoundItPassed(t *testing.T) {
	// The error is what reaches a report, so it has to carry both: which
	// invocation of the three hung, and the bound it is being judged against.
	was := Timeout
	defer func() { Timeout = was }()
	Timeout = 90 * time.Second
	got := DidNotFinish("pdfimages -list").Error()
	if !strings.Contains(got, "pdfimages -list") || !strings.Contains(got, "1m30s") {
		t.Errorf("a hang reads %q", got)
	}
}

func TestTheToolsErrorOutputCanBeAskedFor(t *testing.T) {
	// pdfimages prints its version on stderr, so the one invocation whose
	// whole purpose is to record WHICH poppler judged a run comes back empty
	// from Run. The judge is half the measurement, so it has to be asked on
	// both streams.
	out, hung, err := Combined("sh", "-c", "echo out; echo err 1>&2")
	if hung || err != nil {
		t.Fatalf("hung=%v, %v", hung, err)
	}
	if !strings.Contains(string(out), "err") {
		t.Errorf("the error output is missing: %q", out)
	}
}

func TestCombinedIsBoundedToo(t *testing.T) {
	was := Timeout
	defer func() { Timeout = was }()
	Timeout = 50 * time.Millisecond
	if _, hung, err := Combined("sleep", "30"); !hung {
		t.Fatalf("a tool that does not return was not called a hang: %v", err)
	}
}
