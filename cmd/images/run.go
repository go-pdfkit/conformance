package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/go-pdfkit/conformance/corpus"
	"github.com/go-pdfkit/conformance/images"
	"github.com/go-pdfkit/conformance/internal/poppler"
)

// judgeOne is a variable so a test can judge without poppler installed.
var judgeOne = images.Judge

// now is a variable so a test can know what the record will say it was taken.
var now = time.Now

// A baseline is a whole run written down: what was judged, against which
// judge, by which versions of us, and when.
//
// The counts alone are not a baseline. A number that fell between two runs
// means a regression only if everything else held; without the versions and
// the corpus beside it, a drop is as likely to be a newer poppler or a corpus
// that grew. So the record carries its own conditions.
type baseline struct {
	Corpus string `json:"corpus"`
	Taken  string `json:"taken"`
	// Pages is how many pages of each document the pictures were taken from.
	Pages int `json:"pages"`
	// Gate is the magnitude gate the comparison used, in levels of 255. It is
	// recorded because it IS the instrument: a record taken at a different
	// gate, or by the ink bisection that preceded any gate at all, is not
	// comparable with this one and must not be subtracted from it.
	Gate int `json:"gate"`
	// Timeout is how long a poppler tool was allowed on one document before
	// it was called a hang. It is recorded beside the gate and for the same
	// reason: it is part of the instrument. A run under a shorter bound can
	// name documents as hung that a longer one measures, and the populations
	// those documents are in would then read as smaller without saying why.
	Timeout string `json:"timeout"`
	// Judge is the other implementation, the one whose agreement is the
	// measurement. Empty when it would not say.
	Judge string `json:"judge"`
	// Modules are the versions this was built against.
	Modules     []module         `json:"modules"`
	Populations []images.Summary `json:"populations"`
}

// A module is one dependency and the version it was built at.
type module struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// run judges a corpus, one population at a time.
func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("images", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dir := fs.String("dir", "", "the corpus directory")
	only := fs.String("only", "", "judge just this population")
	pages := fs.Int("pages", 1, "pages of each document to take pictures from")
	limit := fs.Int("limit", 0, "judge no more than this many documents per population")
	judgeTimeout := fs.Duration("timeout", poppler.Timeout, "how long a poppler tool may take on one document before it is called a hang")
	asJSON := fs.Bool("json", false, "write the whole run as a baseline record instead of a report")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dir == "" {
		fmt.Fprintln(errOut, "images: -dir is needed")
		return 2
	}
	// Every poppler invocation is bounded, because pdfimages hangs on at
	// least one document of this corpus and a hang looks exactly like a slow
	// run; see conformance#21.
	poppler.Timeout = *judgeTimeout
	entries, err := corpus.Read(*dir)
	if err != nil {
		fmt.Fprintf(errOut, "images: %v\n", err)
		return 1
	}
	groups := map[string][]string{}
	for _, e := range entries {
		if *only == "" || e.Origin == *only {
			groups[e.Origin] = append(groups[e.Origin], filepath.Join(*dir, e.Path))
		}
	}
	if len(groups) == 0 {
		fmt.Fprintf(errOut, "images: %s holds no population named %q\n", *dir, *only)
		return 1
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	record := baseline{Corpus: *dir, Taken: now().UTC().Format(time.RFC3339),
		Pages: *pages, Gate: images.Gate, Timeout: poppler.Timeout.String(),
		Judge: judgeVersion(), Modules: modules()}
	for _, name := range names {
		paths := groups[name]
		if *limit > 0 && len(paths) > *limit {
			paths = paths[:*limit]
		}
		var rs []images.Result
		for _, p := range paths {
			rs = append(rs, judgeOne(p, images.Options{Pages: *pages})...)
		}
		summary := images.Summarize(name, len(paths), rs)
		if *asJSON {
			record.Populations = append(record.Populations, summary)
			continue
		}
		// A population is written as it finishes rather than at the end,
		// because a corpus takes hours and a run that is killed at hour three
		// should still have said what it learned in the first two.
		fmt.Fprintf(out, "%s\n", name)
		fmt.Fprint(out, images.Report(images.Tally(rs)))
		// Every hang is named in the report as well as in the record: a
		// document the judge would not answer about is not a document that
		// scored badly, and a report that leaves it out cannot be told from
		// one that has none.
		for _, h := range summary.Hung {
			fmt.Fprintf(out, "  hung  %s  %s page %d\n", h.Tool, h.Path, h.Page)
		}
	}
	if *asJSON {
		// Marshal fails only on a value it cannot encode, and this one is
		// numbers, strings and slices of them.
		b, _ := json.MarshalIndent(record, "", "  ")
		fmt.Fprintf(out, "%s\n", b)
	}
	return 0
}

// versionCommand is a variable so a test can ask without poppler installed.
var versionCommand = func() ([]byte, error) {
	// Bounded like every other poppler invocation in this repository. Asking
	// for a version is the one call that should never hang, which is exactly
	// the reasoning that leaves a sweep waiting for ever on the one that does.
	// A version that could not be taken is recorded as empty either way, so
	// the hang is not distinguished here: nothing is skipped because of it.
	out, _, err := poppler.Run("pdfimages", "-v")
	return out, err
}

// judgeVersion is which poppler the comparison was made against.
//
// It is worth recording because the judge is half the measurement: a filter
// whose agreement falls when poppler changes has not regressed, and telling
// those apart afterwards is impossible if nobody wrote down which poppler it
// was.
func judgeVersion() string {
	out, err := versionCommand()
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line)
}

// buildInfo is a variable so a test can see what happens without one.
var buildInfo = debug.ReadBuildInfo

// modules is the versions this was built against, which is what a later run
// compares itself to.
func modules() []module {
	info, ok := buildInfo()
	if !ok {
		return nil
	}
	out := make([]module, 0, len(info.Deps))
	for _, d := range info.Deps {
		out = append(out, module{Path: d.Path, Version: d.Version})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
