package pacman

import (
	"slices"
	"testing"
)

func TestMissing(t *testing.T) {
	got := Missing([]string{"linux", "linux-headers"}, []string{"linux", "linux-lts", "linux-rt"})
	want := []string{"linux-lts", "linux-rt"}

	if !slices.Equal(got, want) {
		t.Errorf("Missing() = %v, want %v", got, want)
	}

	if got := Missing([]string{"a"}, []string{"a", "a"}); len(got) != 0 {
		t.Errorf("Missing() = %v, want an empty result", got)
	}
}

func TestAddIgnore(t *testing.T) {
	tests := []struct {
		name    string
		content string
		names   []string
		want    string
		changed bool
	}{
		{
			"appends to an existing entry",
			"IgnorePkg = linux\n",
			[]string{"linux-headers"},
			"IgnorePkg = linux linux-headers\n",
			true,
		},
		{
			"preserves inline comments",
			"IgnorePkg =  linux   # comment here\n",
			[]string{"linux-headers"},
			"IgnorePkg = linux linux-headers # comment here\n",
			true,
		},
		{
			"keeps surrounding indentation",
			"  IgnorePkg = linux\n",
			[]string{"linux-headers"},
			"  IgnorePkg = linux linux-headers\n",
			true,
		},
		{
			"key spelling is case insensitive",
			"ignorepkg = linux\n",
			[]string{"linux-headers"},
			"ignorepkg = linux linux-headers\n",
			true,
		},
		{
			"no-op when every name is present",
			"IgnorePkg = linux linux-headers\n",
			[]string{"linux", "linux-headers"},
			"IgnorePkg = linux linux-headers\n",
			false,
		},
		{
			"no-op without names",
			"IgnorePkg = linux\n",
			nil,
			"IgnorePkg = linux\n",
			false,
		},
		{
			"inserts before the first section after [options]",
			"[repo]\n\n[options]\nSigLevel = Required\n\n[core]\nX = 1\n",
			[]string{"linux", "linux-headers"},
			"[repo]\n\n[options]\nSigLevel = Required\nIgnorePkg = linux linux-headers\n\n[core]\nX = 1\n",
			true,
		},
		{
			"appends after the options section at the end of file",
			"[options]\nSigLevel = Required\n",
			[]string{"linux"},
			"[options]\nSigLevel = Required\nIgnorePkg = linux\n",
			true,
		},
		{
			"local repository options section is case insensitive",
			"[repo]\n\n[OPTIONS]\nSigLevel = Required\n",
			[]string{"linux"},
			"[repo]\n\n[OPTIONS]\nSigLevel = Required\nIgnorePkg = linux\n",
			true,
		},
		{
			"no-op when the options section is absent",
			"[repo]\nSigLevel = Required\n",
			[]string{"linux"},
			"[repo]\nSigLevel = Required\n",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := AddIgnore(tt.content, tt.names)

			if got != tt.want {
				t.Errorf("AddIgnore() =\n%q\nwant\n%q", got, tt.want)
			}
			if changed != tt.changed {
				t.Errorf("AddIgnore() changed = %v, want %v", changed, tt.changed)
			}
		})
	}
}

func TestAddIgnoreOnlyFirstLineIsUpdated(t *testing.T) {
	content := "IgnorePkg = a\nb\nIgnorePkg = c\nF"

	got, changed := AddIgnore(content, []string{"z"})

	want := "IgnorePkg = a z\nb\nIgnorePkg = c\nF"
	if !changed || got != want {
		t.Errorf("AddIgnore() = %q (%v), want %q (true)", got, changed, want)
	}
}

func TestParseIgnored(t *testing.T) {
	content := `# IgnorePkg = commented-out
IgnorePkg = a b
  IGNOREPKG =  c   # note
IgnorePkg = "quoted name"
`

	// Quoted values keep the quote characters in the tokens: pacman's own
	// config tokenizer does the same.
	got := parseIgnored(content)
	want := []string{"a", "b", "c", `"quoted`, `name"`}

	if !slices.Equal(got, want) {
		t.Errorf("parseIgnored() = %v, want %v", got, want)
	}
}

func TestParseIgnoredEmpty(t *testing.T) {
	if got := parseIgnored(""); len(got) != 0 {
		t.Errorf("parseIgnored(empty) = %v, want an empty result", got)
	}
}

func TestSplitComment(t *testing.T) {
	tests := []struct {
		value       string
		wantValue   string
		wantComment string
	}{
		{"a b", "a b", ""},
		{"a b # note", "a b", "# note"},
		{"  # leading comment", "", "# leading comment"},
		{"a#tight", "a", "#tight"},
		{"", "", ""},
	}

	for _, tt := range tests {
		value, comment := splitComment(tt.value)
		if value != tt.wantValue || comment != tt.wantComment {
			t.Errorf("splitComment(%q) = (%q, %q), want (%q, %q)",
				tt.value, value, comment, tt.wantValue, tt.wantComment)
		}
	}
}

func TestOptionsSectionEnd(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  int
		found bool
	}{
		{
			name:  "present with sections after",
			lines: []string{"[options]", "A = 1", "", "[core]"},
			want:  2,
			found: true,
		},
		{
			name:  "present ending the file",
			lines: []string{"[options]", "A = 1", ""},
			want:  2,
			found: true,
		},
		{
			name:  "absent",
			lines: []string{"[repo]", "A = 1"},
			want:  0,
			found: false,
		},
		{
			name:  "empty",
			lines: nil,
			want:  0,
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := optionsSectionEnd(tt.lines)
			if got != tt.want || found != tt.found {
				t.Errorf("optionsSectionEnd() = (%d, %v), want (%d, %v)", got, found, tt.want, tt.found)
			}
		})
	}
}
