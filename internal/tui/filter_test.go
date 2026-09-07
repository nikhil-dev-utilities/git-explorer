package tui

import (
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func orgNames(orgs []forge.Org) []string {
	names := make([]string, len(orgs))
	for i, o := range orgs {
		names[i] = o.Name
	}
	return names
}

func TestFilterOrgs(t *testing.T) {
	orgs := []forge.Org{{Name: "acme"}, {Name: "acme-labs"}, {Name: "Globex"}, {Name: "tf-network"}}

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty query returns everything", "", []string{"acme", "acme-labs", "Globex", "tf-network"}},
		{"substring narrows", "acme", []string{"acme", "acme-labs"}},
		{"substring is case-insensitive", "globex", []string{"Globex"}},
		{"substring excludes non-matches", "tf-", []string{"tf-network"}},
		{"no match returns empty, not nil-panics", "zzz-nope", nil},
		{"regex via leading slash", "/^tf-", []string{"tf-network"}},
		{"regex excludes non-matches", "/^acme$", []string{"acme"}},
		{"invalid regex matches nothing, does not panic", "/[", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orgNames(filterOrgs(orgs, tt.query))
			if len(got) != len(tt.want) {
				t.Fatalf("filterOrgs(%q) = %v, want %v", tt.query, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("filterOrgs(%q)[%d] = %q, want %q", tt.query, i, got[i], tt.want[i])
				}
			}
		})
	}
}
