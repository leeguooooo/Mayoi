package git

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// CommitPatch returns a binary-capable patch against the first parent. A root
// commit is compared with the empty tree.
func (r *Repo) CommitPatch(hash string) (string, error) {
	hash, err := r.commitRevision(hash)
	if err != nil {
		return "", err
	}
	parents, err := r.run(nil, "rev-list", "--parents", "-n", "1", hash, "--")
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(parents))
	args := []string{"diff-tree", "--no-color", "--root", "--no-commit-id", "--binary", "--no-ext-diff", "--no-textconv", "-p", "-r"}
	if len(fields) > 1 {
		args = append(args, fields[1])
	}
	args = append(args, hash, "--")
	out, err := r.run(nil, args...)
	return string(out), err
}

// ComparePatch returns binary-capable changes from one commit to another.
func (r *Repo) ComparePatch(from, to string) (string, error) {
	a, err := r.commitRevision(from)
	if err != nil {
		return "", err
	}
	b, err := r.commitRevision(to)
	if err != nil {
		return "", err
	}
	args := []string{"diff", "--no-color", "--binary", "--no-ext-diff", "--no-textconv", a, b}
	out, err := r.run(nil, append(args, "--")...)
	return string(out), err
}

// RemoteWebURL maps known public hosting services to credential-free HTTPS
// links. hash is a commit ID, never a branch name or a URL path.
func RemoteWebURL(rawURL, hash string) (string, error) {
	if len(hash) != 40 && len(hash) != 64 {
		return "", fmt.Errorf("invalid commit ID")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", fmt.Errorf("invalid commit ID")
	}
	if !strings.Contains(rawURL, "://") {
		left, right, ok := strings.Cut(rawURL, ":")
		if !ok || !strings.Contains(left, "@") {
			return "", fmt.Errorf("unsupported remote URL")
		}
		rawURL = "ssh://" + left + "/" + right
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid remote URL")
	}
	if u.Scheme != "https" && u.Scheme != "ssh" {
		return "", fmt.Errorf("unsupported remote URL scheme")
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "gitlab.com" && host != "bitbucket.org" {
		return "", fmt.Errorf("unsupported remote hosting service")
	}
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid repository path")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid repository path")
		}
	}
	segment := "/commit/"
	switch host {
	case "gitlab.com":
		segment = "/-" + segment
	case "bitbucket.org":
		segment = "/commits/"
	}
	return (&url.URL{Scheme: "https", Host: host, Path: path + segment + hash}).String(), nil
}

// commitRevision resolves one revision before it enters diff arguments.
// ResolveCommit returns the full hash of the commit a revision names, as
// HEAD or a short hash.
func (r *Repo) ResolveCommit(rev string) (string, error) { return r.commitRevision(rev) }

func (r *Repo) commitRevision(ref string) (string, error) {
	if ref == "" || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\x00\r\n") {
		return "", fmt.Errorf("invalid revision %q", ref)
	}
	out, err := r.run(nil, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	return strings.TrimSpace(string(out)), err
}

// CommitRemoteURL prefers a unique decoration, the current upstream, origin,
// then the first remote. Only the chosen remote's fetch URL is read.
func (r *Repo) CommitRemoteURL(cm Commit) (string, error) {
	names, err := r.Remotes()
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("this repository has no remote")
	}
	decorated := map[string]bool{}
	for _, ref := range cm.Refs {
		if ref.Kind != RefRemote {
			continue
		}
		match := ""
		for _, name := range names {
			if strings.HasPrefix(ref.Name, name+"/") && len(name) > len(match) {
				match = name
			}
		}
		if match != "" {
			decorated[match] = true
		}
	}
	chosen := ""
	if len(decorated) == 1 {
		for name := range decorated {
			chosen = name
		}
	}
	if chosen == "" {
		out, err := r.run(nil, "for-each-ref", "--format=%(HEAD)%1f%(upstream:remotename)", "refs/heads")
		if err != nil {
			return "", err
		}
		for _, row := range strings.Split(string(out), "\n") {
			if name, ok := strings.CutPrefix(row, "*\x1f"); ok {
				for _, configured := range names {
					if name == configured {
						chosen = name
					}
				}
			}
		}
	}
	if chosen == "" {
		chosen = names[0]
		for _, name := range names {
			if name == "origin" {
				chosen = name
			}
		}
	}
	out, err := r.run(nil, "remote", "get-url", "--", chosen)
	if err != nil {
		return "", err
	}
	return RemoteWebURL(strings.TrimSuffix(string(out), "\n"), cm.Hash)
}
