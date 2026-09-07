package clone

import "testing"

func TestParseCloneURL(t *testing.T) {
	want := repoIdentity{Host: "github.com", Owner: "acme", Name: "api"}

	tests := []struct {
		name string
		url  string
		want repoIdentity
		ok   bool
	}{
		{"ssh scp-like", "git@github.com:acme/api.git", want, true},
		{"https", "https://github.com/acme/api.git", want, true},
		{"https no .git suffix", "https://github.com/acme/api", want, true},
		{"ssh:// URL form", "ssh://git@github.com/acme/api.git", want, true},
		{"mixed case normalizes", "https://GitHub.com/Acme/API.git", want, true},
		{"private host", "git@ghe.corp.internal:platform/infra.git",
			repoIdentity{Host: "ghe.corp.internal", Owner: "platform", Name: "infra"}, true},
		{"garbage", "not-a-url-at-all", repoIdentity{}, false},
		{"empty", "", repoIdentity{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCloneURL(tt.url)
			if ok != tt.ok {
				t.Fatalf("parseCloneURL(%q) ok = %v, want %v", tt.url, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("parseCloneURL(%q) = %+v, want %+v", tt.url, got, tt.want)
			}
		})
	}
}

func TestParseCloneURL_SSHAndHTTPSOfSameRepoMatch(t *testing.T) {
	ssh, ok := parseCloneURL("git@github.com:acme/api.git")
	if !ok {
		t.Fatal("ssh form failed to parse")
	}
	https, ok := parseCloneURL("https://github.com/acme/api.git")
	if !ok {
		t.Fatal("https form failed to parse")
	}
	if ssh != https {
		t.Errorf("ssh identity %+v != https identity %+v, want them equal", ssh, https)
	}
}
