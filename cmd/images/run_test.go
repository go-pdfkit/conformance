package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-pdfkit/conformance/corpus"
	"github.com/go-pdfkit/conformance/images"
)

// tinyCorpus writes a manifest naming two populations and a file for each.
func tinyCorpus(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "one.pdf"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := corpus.Write(dir, []corpus.Entry{
		{Path: "alpha/one.pdf", Origin: "alpha", Source: "u", SHA256: "x"},
		{Path: "beta/one.pdf", Origin: "beta", Source: "u", SHA256: "x"},
	}); err != nil {
		t.Fatal(err)
	}
	return dir
}

// judge stands in for the comparison so these run without poppler.
func judge(t *testing.T, rs ...images.Result) *int {
	t.Helper()
	was := judgeOne
	t.Cleanup(func() { judgeOne = was })
	n := 0
	judgeOne = func(string, images.Options) []images.Result {
		n++
		return rs
	}
	return &n
}

func TestRunReportsEachPopulationSeparately(t *testing.T) {
	judge(t, images.Result{Name: "I", Filter: "JBIG2Decode", Share: 0})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t)}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "alpha\n") || !strings.Contains(got, "beta\n") {
		t.Errorf("got %q", got)
	}
	if strings.Count(got, "JBIG2Decode") != 2 {
		t.Errorf("the filter is not reported once per population: %q", got)
	}
}

func TestRunLooksAtOnePopulation(t *testing.T) {
	judge(t, images.Result{Name: "I", Share: 0})
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", tinyCorpus(t), "-only", "beta"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if got := out.String(); strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Errorf("got %q", got)
	}
}

func TestRunStopsAtTheLimit(t *testing.T) {
	// A corpus is large and a first look should not have to be a whole one.
	n := judge(t, images.Result{Name: "I", Share: 0})
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	var entries []corpus.Entry
	for _, name := range []string{"a.pdf", "b.pdf", "c.pdf"} {
		os.WriteFile(filepath.Join(dir, "alpha", name), []byte("x"), 0o644)
		entries = append(entries, corpus.Entry{
			Path: "alpha/" + name, Origin: "alpha", Source: "u", SHA256: "x"})
	}
	if err := corpus.Write(dir, entries); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"-dir", dir, "-limit", "2"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if *n != 2 {
		t.Errorf("it looked at %d documents, not 2", *n)
	}
}

func TestRunRefusesWhatItCannotDo(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		code int
		want string
	}{
		{"no corpus named", nil, 2, "-dir is needed"},
		{"a flag it does not have", []string{"-nope"}, 2, ""},
		{"a corpus whose manifest says nothing it can read", nil, 1, "images:"},
		{"a population that is not in it", []string{"-only", "gamma"}, 1, "no population"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.args
			switch {
			case len(args) > 0 && args[0] == "-only":
				args = append([]string{"-dir", tinyCorpus(t)}, args...)
			case tc.name == "a corpus whose manifest says nothing it can read":
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "MANIFEST.tsv"),
					[]byte("nothing\tit\tknows\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				args = []string{"-dir", dir}
			}
			var out, errOut bytes.Buffer
			if code := run(args, &out, &errOut); code != tc.code {
				t.Fatalf("exit %d, want %d: %s", code, tc.code, errOut.String())
			}
			if tc.want != "" && !strings.Contains(errOut.String(), tc.want) {
				t.Errorf("said %q", errOut.String())
			}
		})
	}
}

func TestMainRunsTheCommand(t *testing.T) {
	was := osExit
	defer func() { osExit = was }()
	code := -1
	osExit = func(c int) { code = c }
	old := os.Args
	os.Args = []string{"images"}
	defer func() { os.Args = old }()
	main()
	if code != 2 {
		t.Errorf("running it with no corpus exited %d", code)
	}
}
