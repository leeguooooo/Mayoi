package git

import (
	"fmt"
	"strings"

	"mayoi/internal/remote"
)

// WorktreeInUse reports a branch checked out in another worktree.
type WorktreeInUse struct{ Branch, Path string }

func (e *WorktreeInUse) Error() string {
	return fmt.Sprintf("branch %q is checked out in %s", e.Branch, e.Path)
}

// WorktreeForBranch returns the checkout owning a local branch, in the same
// filesystem namespace as the repository. NUL records preserve unusual paths.
func (r *Repo) WorktreeForBranch(branch string) (string, error) {
	out, err := r.run(nil, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		// Git before 2.36, as Ubuntu 22.04's, has no -z.
		if out, err = r.run(nil, "worktree", "list", "--porcelain"); err != nil {
			return "", err
		}
		out = []byte(nulRecords(string(out)))
	}
	return remote.On(r.Root, worktreeForBranch(r.Root, string(out), branch)), nil
}

// nulRecords writes the porcelain list of worktrees, a field a line, as -z
// does, a field ended by a NUL: a path with a newline goes on over the
// lines up to its HEAD, or bare.
func nulRecords(out string) string {
	var b strings.Builder
	inPath := false
	for i, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if inPath && line != "" && !strings.HasPrefix(line, "HEAD ") && line != "bare" {
			b.WriteString("\n" + line)
			continue
		}
		if i > 0 {
			b.WriteByte(0)
		}
		b.WriteString(line)
		inPath = strings.HasPrefix(line, "worktree ")
	}
	b.WriteByte(0)
	return b.String()
}

func worktreeForBranch(root, out, branch string) string {
	var path string
	for _, field := range strings.Split(out, "\x00") {
		switch {
		case field == "":
			path = ""
		case strings.HasPrefix(field, "worktree "):
			path = strings.TrimPrefix(field, "worktree ")
		case field == "branch refs/heads/"+branch && path != "":
			return path
		}
	}
	return ""
}

// PruneWorktrees forgets the worktrees whose folders are gone, which keep
// their branches checked out until it does.
func (r *Repo) PruneWorktrees() error {
	_, err := r.run(nil, "worktree", "prune")
	return err
}
