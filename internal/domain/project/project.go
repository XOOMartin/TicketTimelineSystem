package project

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalid = errors.New("invalid project")

// Project is one of the products the teams work on. It doubles as the team
// a user belongs to.
type Project string

const (
	BlackBox Project = "BlackBox"
	SPinno   Project = "SPinno"
)

// All lists the projects in display order.
func All() []Project { return []Project{BlackBox, SPinno} }

// Parse accepts any casing, e.g. "blackbox" or "SPINNO".
func Parse(s string) (Project, error) {
	s = strings.TrimSpace(s)
	for _, p := range All() {
		if strings.EqualFold(s, string(p)) {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w: must be BlackBox or SPinno", ErrInvalid)
}

func (p Project) String() string { return string(p) }

// Repo is one of a project's code repositories. SPinno's tickets each belong
// to one, with its own ticket numbers: the spec, whose tickets are the
// parent issues, and the backend and web (frontend), whose tickets (bugs
// found, say) sit under a spec ticket. A spec ticket can sit under another
// spec ticket too. BlackBox has none.
type Repo string

const (
	RepoSpec    Repo = "spec"
	RepoBackend Repo = "backend"
	RepoWeb     Repo = "web"
)

var repoLabels = map[Repo]string{
	RepoSpec:    "Spinno next Spec",
	RepoBackend: "Spinno next (backend)",
	RepoWeb:     "Spinno next web (frontend)",
}

// Repos lists the project's repos in display order; none for BlackBox.
func (p Project) Repos() []Repo {
	if p == SPinno {
		return []Repo{RepoSpec, RepoBackend, RepoWeb}
	}
	return nil
}

// ParseRepo accepts one of the project's repos, in any casing, or "" (no
// repo: any BlackBox ticket, and SPinno tickets from before repos).
func (p Project) ParseRepo(s string) (Repo, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	for _, r := range p.Repos() {
		if strings.EqualFold(s, string(r)) {
			return r, nil
		}
	}
	if len(p.Repos()) == 0 {
		return "", fmt.Errorf("%w: %s has no repos", ErrInvalid, p)
	}
	return "", fmt.Errorf("%w: %s's repo must be spec, backend or web", ErrInvalid, p)
}

func (r Repo) Label() string { return repoLabels[r] }

// HasParent reports whether the repo's tickets can sit under a spec ticket:
// all of SPinno's (a spec under a bigger spec, a bug under its spec).
func (r Repo) HasParent() bool { return r == RepoSpec || r == RepoBackend || r == RepoWeb }
