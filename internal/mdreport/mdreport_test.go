package mdreport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAFirstRunLeavesRoomForTheAnalysis(t *testing.T) {
	path := filepath.Join(t.TempDir(), "REPORT.md")
	if err := Write(path, "# table\n\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "# table\n\n"+Placeholder {
		t.Errorf("got %q", got)
	}
}

func TestARerunKeepsWhatWasWrittenUnderTheMarker(t *testing.T) {
	// The table is regenerated; the reader's notes beneath it are the reason
	// the file exists, and a run that lost them would have to be run again
	// by someone who no longer remembers what they said.
	path := filepath.Join(t.TempDir(), "REPORT.md")
	old := "# old table\n\n" + Marker + "\n\nquartz is judge noise here: it disagrees on the control too.\n"
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, "# new table\n\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if strings.Contains(s, "old table") {
		t.Errorf("the old table survived: %q", s)
	}
	if !strings.HasPrefix(s, "# new table\n\n"+Marker) || !strings.Contains(s, "judge noise") {
		t.Errorf("the analysis did not: %q", s)
	}
}

func TestAFileWithoutAMarkerIsReplacedWhole(t *testing.T) {
	// There is nothing to preserve in it, and a file with no marker is one
	// this package did not write.
	path := filepath.Join(t.TempDir(), "REPORT.md")
	if err := os.WriteFile(path, []byte("someone else's file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, "# table\n\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "# table\n\n"+Placeholder {
		t.Errorf("got %q", got)
	}
}

func TestAReportThatCannotBeWrittenSaysSo(t *testing.T) {
	if err := Write(filepath.Join(t.TempDir(), "no", "such", "dir", "REPORT.md"), "x"); err == nil {
		t.Error("a report was written into a directory that does not exist")
	}
}
