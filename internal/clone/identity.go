package clone

import "strings"

// repoIdentity is the (host, owner, name) triple used to tell whether two clone URLs —
// possibly in different protocols — refer to the same Repo. Comparing URLs as raw
// strings is wrong: an ssh URL and an https URL for the identical Repo are textually
// nothing alike but must compare equal here.
type repoIdentity struct {
	Host  string
	Owner string
	Name  string
}

// parseCloneURL extracts a repoIdentity from a clone URL in any of the forms
// forge/github's CloneURL produces (git@host:owner/name.git, https://host/owner/name.git)
// or the ssh://git@host/owner/name.git variant some tools use. GitHub hostnames,
// owners, and repo names are all case-insensitive, so every component is lowercased —
// two URLs differing only in case must still compare equal.
func parseCloneURL(raw string) (repoIdentity, bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")

	switch {
	case strings.HasPrefix(s, "https://"):
		return parseURLForm(strings.TrimPrefix(s, "https://"))
	case strings.HasPrefix(s, "http://"):
		return parseURLForm(strings.TrimPrefix(s, "http://"))
	case strings.HasPrefix(s, "ssh://"):
		return parseURLForm(stripUserinfo(strings.TrimPrefix(s, "ssh://")))
	default:
		// scp-like: [user@]host:owner/name
		return parseSCPForm(s)
	}
}

// parseURLForm handles "host/owner/name" (what's left after stripping a scheme and any
// userinfo).
func parseURLForm(hostAndPath string) (repoIdentity, bool) {
	parts := strings.SplitN(hostAndPath, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return repoIdentity{}, false
	}
	return repoIdentity{
		Host:  strings.ToLower(parts[0]),
		Owner: strings.ToLower(parts[1]),
		Name:  strings.ToLower(parts[2]),
	}, true
}

// parseSCPForm handles "[user@]host:owner/name".
func parseSCPForm(s string) (repoIdentity, bool) {
	if i := strings.Index(s, "@"); i != -1 {
		s = s[i+1:]
	}
	i := strings.Index(s, ":")
	if i == -1 {
		return repoIdentity{}, false
	}
	host, path := s[:i], s[i+1:]
	parts := strings.SplitN(path, "/", 2)
	if host == "" || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return repoIdentity{}, false
	}
	return repoIdentity{
		Host:  strings.ToLower(host),
		Owner: strings.ToLower(parts[0]),
		Name:  strings.ToLower(parts[1]),
	}, true
}

func stripUserinfo(s string) string {
	if i := strings.Index(s, "@"); i != -1 {
		return s[i+1:]
	}
	return s
}
