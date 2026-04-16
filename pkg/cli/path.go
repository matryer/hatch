package cli

import (
	"fmt"
	"path"
	"strings"

	"github.com/grafana/hatch/pkg/source"
)

// validatePathFlag validates the -path argument shared by `hatch init` and
// `hatch new`. The flag value is a forward-slash relative path naming a
// nested scope under .hatch/ (e.g. "backend", "services/api"). Returns the
// normalised path, or an error explaining what's wrong.
//
// Empty input is allowed and returns "" — that selects the root scope,
// which is today's behaviour.
//
// Rejected: absolute paths, parent traversal (`..`), empty path components
// (e.g. `foo//bar`), and any component matching one of the four known
// primitive container names (_rules/_skills/_commands/_agents). The
// last rule prevents creating a primitive-container-named scope dir
// that the walker can't load. Other `_`-prefixed components are allowed,
// since unknown `_xxx` dirs are treated as ordinary scope path components
// by the walker.
func validatePathFlag(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("-path %q is absolute; use a forward-slash relative path like 'backend' or 'services/api'", p)
	}
	// Use forward-slash semantics for splitting regardless of OS.
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("-path %q contains an empty component", p)
		}
		if part == "." || part == ".." {
			return "", fmt.Errorf("-path %q must not contain %q", p, part)
		}
		if part == source.RulesDir || part == source.SkillsDir || part == source.CommandsDir || part == source.AgentsDir {
			return "", fmt.Errorf("-path %q: %q is a hatch primitive container name, not a valid scope component", p, part)
		}
	}
	// path.Clean catches the case where a tricky combination still
	// resolves outside the original (paranoia after the per-component
	// checks).
	cleaned := path.Clean(p)
	if cleaned != p {
		return "", fmt.Errorf("-path %q must be in canonical form (got %q after normalisation)", p, cleaned)
	}
	return p, nil
}
