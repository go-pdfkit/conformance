// Copyright (c) the go-pdfkit authors.
// SPDX-License-Identifier: BSD-3-Clause

package survey

import (
	"testing"
)

// featuresOf surveys a one-page document with the content given and returns
// what the page was counted as using.
func featuresOf(t *testing.T, content string) map[string]int {
	t.Helper()
	c := Survey([]string{page(t, content, nil)}, 0)
	if c.Pages != 1 {
		t.Fatalf("%d pages surveyed, want 1", c.Pages)
	}
	return c.Features
}

func TestAPageIsCountedByWhatItUses(t *testing.T) {
	for _, tc := range []struct {
		why     string
		content string
		want    string
	}{
		{"a clip", "0 0 10 10 re W n", FeatureClip},
		{"an even-odd clip", "0 0 10 10 re W* n", FeatureClip},
		{"an XObject", "/Im0 Do", FeatureImage},
		{"an inline image", "BI /W 1 /H 1 ID \x00 EI", FeatureImage},
		{"a shading", "/Sh0 sh", FeatureShading},
		{"a pattern fill", "/Pattern cs /P0 scn 0 0 5 5 re f", FeaturePattern},
		{"a pattern stroke", "/Pattern CS /P0 SCN 0 0 5 5 re S", FeaturePattern},
	} {
		got := featuresOf(t, tc.content)
		if got[tc.want] != 1 {
			t.Errorf("%s: %v does not name %q", tc.why, got, tc.want)
		}
		if got[FeaturePlain] != 0 {
			t.Errorf("%s: counted as plain as well as %q", tc.why, tc.want)
		}
	}
}

func TestAPlainPageIsCountedAsPlain(t *testing.T) {
	// ⛔ The control. Without it, a counter that marked every page with every
	// feature would pass every case above.
	got := featuresOf(t, "0 0 10 10 re f BT /F0 12 Tf (hi) Tj ET 1 1 m 9 9 l S")
	if got[FeaturePlain] != 1 {
		t.Errorf("a page of paths and text was not counted plain: %v", got)
	}
	for _, f := range []string{FeatureClip, FeatureImage, FeatureShading, FeaturePattern} {
		if got[f] != 0 {
			t.Errorf("a page of paths and text was counted as using %q: %v", f, got)
		}
	}
}

func TestAPageUsingSeveralFeaturesIsCountedUnderEach(t *testing.T) {
	// The counts are per feature, not a classification: a page that clips AND
	// draws a picture is one page in both rows and in neither "plain".
	got := featuresOf(t, "0 0 10 10 re W n /Im0 Do")
	if got[FeatureClip] != 1 || got[FeatureImage] != 1 {
		t.Errorf("%v, want one clip and one image", got)
	}
	if got[FeaturePlain] != 0 {
		t.Errorf("counted plain as well: %v", got)
	}
}

func TestAPlainColourIsNotAPattern(t *testing.T) {
	// ⛔ scn sets a colour in whatever space is current, and only names a
	// PATTERN when its last operand is a name. Counting every scn would have
	// made pattern the commonest feature in any corpus instead of the rarest.
	got := featuresOf(t, "/DeviceRGB cs 0.2 0.4 0.6 scn 0 0 5 5 re f")
	if got[FeaturePattern] != 0 {
		t.Errorf("a plain scn colour was counted as a pattern: %v", got)
	}
	if got[FeaturePlain] != 1 {
		t.Errorf("want the page counted plain: %v", got)
	}
}

func TestFeatureNamesComeBackSorted(t *testing.T) {
	c := Counts{Features: map[string]int{"shading": 1, "clip": 2, "image": 3}}
	got := c.FeatureNames()
	want := []string{"clip", "image", "shading"}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%v, want %v", got, want)
		}
	}
}
