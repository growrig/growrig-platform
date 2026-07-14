package domain

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Blue Dream", "blue-dream"},
		{"OG Kush #2", "og-kush-2"},
		{"  Trailing/Leading  ", "trailing-leading"},
		{"Genovese Basil (Batch 3)", "genovese-basil-batch-3"},
		{"already-slug", "already-slug"},
		{"multi   spaces", "multi-spaces"},
		{"!!!", ""}, // no alphanumeric content
		{"", ""},
	}
	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
