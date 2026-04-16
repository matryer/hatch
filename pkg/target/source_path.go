package target

import (
	"github.com/grafana/hatch/pkg/source"
)

// SourceFilePathFor is SourceFilePath with a nil-path fallback for
// synthetic primitives. If the primitive has no on-disk SourcePath
// (currently only hatch's auto-injected meta skill), returns "" so
// callers can omit the `source:` metadata rather than point at a file
// that doesn't exist.
func SourceFilePathFor(scopePath string, p source.Primitive) string {
	if p.SourcePath == "" {
		return ""
	}
	return SourceFilePath(scopePath, p.Kind, p.Name)
}

// SourceFilePath returns the repo-root-relative forward-slash path of
// the .hatch/ source file that produced a primitive, given its scope,
// kind, and name. Used to embed a `source:` metadata field in every
// generated frontmatter block so a reader can jump from a generated
// output straight to its authoring source.
//
// The name may contain forward slashes (namespaced commands); they're
// preserved, mirroring how the command's subdirectory was loaded.
func SourceFilePath(scopePath string, kind source.Kind, name string) string {
	dir := primitiveContainerDir(kind)
	base := scopeHatchPath(scopePath)
	switch kind {
	case source.KindSkill:
		return base + "/" + dir + "/" + name + "/SKILL.md"
	default:
		return base + "/" + dir + "/" + name + ".md"
	}
}

// primitiveContainerDir returns the underscore-prefixed container name
// hatch uses under .hatch/ for the given kind.
func primitiveContainerDir(kind source.Kind) string {
	switch kind {
	case source.KindRule:
		return source.RulesDir
	case source.KindSkill:
		return source.SkillsDir
	case source.KindCommand:
		return source.CommandsDir
	case source.KindAgent:
		return source.AgentsDir
	default:
		return "_" + string(kind) + "s"
	}
}

// scopeHatchPath returns the `.hatch[/scope]` prefix for the given
// scope path. Uses forward slashes regardless of OS — the output is a
// repo-root-relative path embedded in YAML, not a filesystem path.
func scopeHatchPath(scopePath string) string {
	if scopePath == "" {
		return ".hatch"
	}
	return ".hatch/" + scopePath
}
