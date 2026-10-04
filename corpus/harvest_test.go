package corpus

import "testing"

func TestAnIdentifierThatWouldMakeAnAwkwardFile(t *testing.T) {
	// The identifier comes out of a remote server's search results, and this
	// is where it becomes a path on disk and a row in a manifest.
	for _, c := range []struct{ id, want string }{
		// A dash at the start is read as a FLAG by every poppler tool.
		{"-v", "v.pdf"},
		{"--help", "help.pdf"},
		// A slash would make a directory; already handled, kept tested.
		{"a/b", "a_b.pdf"},
		// A dot at the start hides the file from a directory walk while the
		// manifest still lists it -- the disagreement §33 is about.
		{".hidden", "hidden.pdf"},
		// Nothing but dashes and dots: keep something rather than drop a
		// document for its name.
		{"-.-", "document.pdf"},
		// Everything else is left alone.
		{"sim_unitarian-register", "sim_unitarian-register.pdf"},
	} {
		if got := safeName(c.id); got != c.want {
			t.Errorf("safeName(%q) = %q, want %q", c.id, got, c.want)
		}
	}
}
