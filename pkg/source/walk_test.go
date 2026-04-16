package source_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/matryer/is"
)

// writeTree creates a nested file structure under root from a map of
// relative paths to contents. Used to build test fixtures inline.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoad_MissingSourceDirErrors(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	_, err := source.Load(dir)
	is.True(err != nil)
}

func TestLoad_Rules(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/_rules/alpha.md": "alpha body",
		".hatch/_rules/beta.md":  "beta body",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Root().Rules), 2)
	// Sorted alphabetically by name.
	is.Equal(src.Root().Rules[0].Name, "alpha")
	is.Equal(src.Root().Rules[1].Name, "beta")
}

func TestLoad_SkillsAsDirectories(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/_skills/review-pr/SKILL.md": "---\ndescription: review\n---\nbody\n",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Root().Skills), 1)
	is.Equal(src.Root().Skills[0].Name, "review-pr")
	is.Equal(src.Root().Skills[0].Description, "review")
	// SourcePath points to the skill directory, not the SKILL.md file.
	info, err := os.Stat(src.Root().Skills[0].SourcePath)
	is.NoErr(err)
	is.True(info.IsDir())
}

func TestLoad_CommandsAndAgents(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/_commands/commit.md": "---\ndescription: c\n---\nbody\n",
		".hatch/_agents/security.md": "---\ndescription: a\n---\nbody\n",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Root().Commands), 1)
	is.Equal(len(src.Root().Agents), 1)
	is.Equal(src.Root().Commands[0].Name, "commit")
	is.Equal(src.Root().Agents[0].Name, "security")
}

func TestLoad_NamespacedCommands(t *testing.T) {
	// Claude Code supports namespaced slash commands via subdirectories under
	// .claude/commands/ (e.g. .claude/commands/opsx/apply.md → /opsx:apply).
	// To express that in hatch source, `_commands/<ns>/<name>.md` must load
	// with Name="<ns>/<name>". Flat files in the same _commands/ dir stay
	// unnamespaced.
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/_commands/commit.md":      "---\ndescription: c\n---\nflat body\n",
		".hatch/_commands/opsx/apply.md":  "---\ndescription: apply\n---\napply body\n",
		".hatch/_commands/opsx/verify.md": "---\ndescription: verify\n---\nverify body\n",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Root().Commands), 3)
	// Sorted by name lexicographically: commit, opsx/apply, opsx/verify.
	is.Equal(src.Root().Commands[0].Name, "commit")
	is.Equal(src.Root().Commands[1].Name, "opsx/apply")
	is.Equal(src.Root().Commands[2].Name, "opsx/verify")
	is.Equal(src.Root().Commands[1].Description, "apply")
}

func TestLoad_IgnoresNonMarkdownFiles(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/_rules/real.md":   "content",
		".hatch/_rules/notes.txt": "ignore me",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Root().Rules), 1)
}

// findScope returns the scope with the given Path, or nil if absent. Used
// by the nested-path tests below to assert that a particular scope was
// loaded with the expected primitives.
func findScope(s *source.Source, path string) *source.Scope {
	for i := range s.Scopes {
		if s.Scopes[i].Path == path {
			return &s.Scopes[i]
		}
	}
	return nil
}

func TestLoad_NestedPath_RulesOnly(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/backend/_rules/go.md": "Backend Go rule",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	// Root scope present (and empty).
	is.True(src.Root() != nil)
	is.Equal(len(src.Root().Rules), 0)
	// Backend scope present with one rule.
	backend := findScope(src, "backend")
	is.True(backend != nil)
	is.Equal(len(backend.Rules), 1)
	is.Equal(backend.Rules[0].Name, "go")
}

func TestLoad_NestedPath_DeepPath(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/services/api/_commands/deploy.md": "---\ndescription: deploy\n---\nbody\n",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	api := findScope(src, "services/api")
	is.True(api != nil)
	is.Equal(len(api.Commands), 1)
	is.Equal(api.Commands[0].Name, "deploy")
}

func TestLoad_NestedPath_RootAndNestedCoexist(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/_rules/global.md":      "global",
		".hatch/backend/_rules/api.md": "backend",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Scopes), 2)
	// Root scope is first.
	is.Equal(src.Scopes[0].Path, "")
	is.Equal(len(src.Scopes[0].Rules), 1)
	is.Equal(src.Scopes[0].Rules[0].Name, "global")
	// Backend scope follows.
	is.Equal(src.Scopes[1].Path, "backend")
	is.Equal(src.Scopes[1].Rules[0].Name, "api")
}

func TestLoad_NestedPath_MultipleSiblingsSortedLexicographically(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/frontend/_rules/x.md": "f",
		".hatch/backend/_rules/x.md":  "b",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Scopes), 3) // root + backend + frontend
	is.Equal(src.Scopes[0].Path, "")
	is.Equal(src.Scopes[1].Path, "backend")
	is.Equal(src.Scopes[2].Path, "frontend")
}

func TestLoad_NestedPath_PassthroughContainerDropped(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/services/api/_rules/r.md": "api",
		".hatch/services/web/_rules/r.md": "web",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	// services/ itself has no primitive containers — it should NOT appear.
	is.True(findScope(src, "services") == nil)
	is.True(findScope(src, "services/api") != nil)
	is.True(findScope(src, "services/web") != nil)
}

func TestLoad_NestedPath_HiddenDirIgnored(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/.cache/_rules/x.md": "should be ignored",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.True(findScope(src, ".cache") == nil)
}

func TestLoad_NestedPath_StrayMarkdownIgnored(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/backend/notes.md":       "stray",
		".hatch/backend/_rules/real.md": "real",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	backend := findScope(src, "backend")
	is.True(backend != nil)
	is.Equal(len(backend.Rules), 1)
	is.Equal(backend.Rules[0].Name, "real")
}

func TestLoad_NestedPath_UnknownUnderscoreDirIsScopeComponent(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	// _workflows is NOT one of the four known primitive container names,
	// so the walker should treat it as an ordinary scope path component.
	writeTree(t, dir, map[string]string{
		".hatch/_workflows/_rules/x.md": "rule under unknown _xxx dir",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	scope := findScope(src, "_workflows")
	is.True(scope != nil)
	is.Equal(len(scope.Rules), 1)
	is.Equal(scope.Rules[0].Name, "x")
}

func TestLoad_NestedPath_RulesAsPathComponentNowLegal(t *testing.T) {
	// The underscore rename frees the bare names: a directory literally
	// called "rules" is now a valid scope path component because the
	// primitive container is "_rules", not "rules".
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/rules/_rules/x.md": "rules-of-the-rules-scope",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	scope := findScope(src, "rules")
	is.True(scope != nil)
	is.Equal(len(scope.Rules), 1)
	is.Equal(scope.Rules[0].Name, "x")
}

func TestLoad_NestedPath_NoRecursionIntoPrimitiveContainer(t *testing.T) {
	// .hatch/_rules/nested/inner.md must NOT be loaded — primitive
	// containers don't recurse — and the walker must not treat
	// .hatch/_rules/_rules/x.md as a nested scope either.
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/_rules/top.md":          "real",
		".hatch/_rules/nested/inner.md": "should be ignored",
		".hatch/_rules/_rules/x.md":     "should also be ignored",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	is.Equal(len(src.Scopes), 1) // only the root scope
	is.Equal(len(src.Root().Rules), 1)
	is.Equal(src.Root().Rules[0].Name, "top")
}

func TestLoad_NestedPath_SkillWithAssets(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".hatch/backend/_skills/review-pr/SKILL.md":         "---\ndescription: review\n---\nbody\n",
		".hatch/backend/_skills/review-pr/scripts/check.sh": "#!/bin/sh\n",
	})
	src, err := source.Load(dir)
	is.NoErr(err)
	backend := findScope(src, "backend")
	is.True(backend != nil)
	is.Equal(len(backend.Skills), 1)
	is.Equal(backend.Skills[0].Name, "review-pr")
	// SourcePath should point inside the backend scope, not the root.
	info, err := os.Stat(backend.Skills[0].SourcePath)
	is.NoErr(err)
	is.True(info.IsDir())
	is.True(strings.Contains(backend.Skills[0].SourcePath, filepath.Join("backend", "_skills", "review-pr")))
}
