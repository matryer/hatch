package cli_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestGen_WritesExpectedFilesAcrossTargets(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	stdout, _, err := invoke(t, "gen")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "CLAUDE.md"))
	is.True(strings.Contains(stdout, "AGENTS.md"))
	is.True(strings.Contains(stdout, ".github/copilot-instructions.md"))

	claudeMD, err := os.ReadFile("CLAUDE.md")
	is.NoErr(err)
	is.True(strings.Contains(string(claudeMD), "<!-- hatch:begin v1 -->"))
	is.True(strings.Contains(string(claudeMD), "Be concise."))

	skill, err := os.ReadFile(".claude/skills/review-pr/SKILL.md")
	is.NoErr(err)
	is.True(strings.Contains(string(skill), "description: Review a PR."))
}

func TestGen_Idempotent(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "gen")
	is.NoErr(err)
	first, err := os.ReadFile("CLAUDE.md")
	is.NoErr(err)

	_, _, err = invoke(t, "gen")
	is.NoErr(err)
	second, err := os.ReadFile("CLAUDE.md")
	is.NoErr(err)

	is.Equal(string(first), string(second))
}

func TestGen_PreservesUserContentAroundBlock(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	userContent := "# My notes\n\nHand-written content here.\n"
	is.NoErr(os.WriteFile("CLAUDE.md", []byte(userContent), 0o644))

	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	got, err := os.ReadFile("CLAUDE.md")
	is.NoErr(err)
	s := string(got)
	is.True(strings.Contains(s, "# My notes"))
	is.True(strings.Contains(s, "Hand-written content here."))
	is.True(strings.Contains(s, "<!-- hatch:begin v1 -->"))
	is.True(strings.Contains(s, "Be concise."))
}

func TestGen_TargetsFlag(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	stdout, _, err := invoke(t, "gen", "-targets", "claude")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "CLAUDE.md"))
	is.True(!strings.Contains(stdout, "AGENTS.md"))
}

func TestGen_UnknownTargetErrors(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)
	_, _, err := invoke(t, "gen", "-targets", "nosuch")
	is.True(err != nil)
}

func TestGen_UnknownTargetInListWritesNothing(t *testing.T) {
	// When the -targets list mixes a valid target with an unknown one,
	// the command must fail before touching the filesystem. Nothing is
	// written — not even the valid target's output.
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "gen", "-targets", "claude,nosuch")
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "nosuch"))

	// Claude's file must NOT exist — the error should fire before any
	// target-level writes.
	for _, path := range []string{
		"CLAUDE.md",
		".claude/skills/review-pr/SKILL.md",
		"AGENTS.md",
	} {
		_, statErr := os.Stat(path)
		is.True(os.IsNotExist(statErr))
	}
}

func TestGen_PositionalArgErrors(t *testing.T) {
	// Target selection is -targets only. A stray positional word must
	// fail loudly rather than be silently ignored and run against every
	// target.
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "gen", "claude")
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "unexpected argument"))

	// No files should have been written.
	for _, path := range []string{"CLAUDE.md", "AGENTS.md", ".claude/skills/review-pr/SKILL.md"} {
		_, statErr := os.Stat(path)
		is.True(os.IsNotExist(statErr))
	}
}

func TestGen_RespectsCanceledContext(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := invokeWithCtx(t, ctx, "", "gen")
	is.True(err != nil)
	is.True(errors.Is(err, context.Canceled))

	// Nothing should have been written.
	_, err = os.Stat("CLAUDE.md")
	is.True(os.IsNotExist(err))
}

func TestGen_NestedPath_EndToEnd(t *testing.T) {
	// Two scopes: root and backend. Each has every primitive type. After
	// `hatch gen`, both root-level and backend/-prefixed outputs should
	// exist for Claude/Codex/OpenCode; Copilot's scoped rule should land
	// at .github/instructions/backend-<name>.instructions.md (not under
	// backend/.github/, which Copilot wouldn't read).
	is := is.New(t)
	dir := t.TempDir()
	files := map[string]string{
		".hatch/_rules/global.md":               "GLOBAL RULE\n",
		".hatch/backend/_rules/api.md":          "BACKEND RULE\n",
		".hatch/backend/_skills/check/SKILL.md": "---\ndescription: check\n---\nbody\n",
	}
	for path, body := range files {
		full := dir + "/" + path
		is.NoErr(os.MkdirAll(filepath.Dir(full), 0o755))
		is.NoErr(os.WriteFile(full, []byte(body), 0o644))
	}
	t.Chdir(dir)

	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	// Root-level files contain the global rule and not the backend one.
	root, err := os.ReadFile("CLAUDE.md")
	is.NoErr(err)
	is.True(strings.Contains(string(root), "GLOBAL RULE"))
	is.True(!strings.Contains(string(root), "BACKEND RULE"))

	// Backend-prefixed files contain the backend rule and not the global one.
	beClaude, err := os.ReadFile("backend/CLAUDE.md")
	is.NoErr(err)
	is.True(strings.Contains(string(beClaude), "BACKEND RULE"))
	is.True(!strings.Contains(string(beClaude), "GLOBAL RULE"))

	beAgents, err := os.ReadFile("backend/AGENTS.md")
	is.NoErr(err)
	is.True(strings.Contains(string(beAgents), "BACKEND RULE"))

	// Backend skill landed under backend/.claude/skills/.
	_, err = os.Stat("backend/.claude/skills/check/SKILL.md")
	is.NoErr(err)

	// Copilot routed the scoped rule to a root .github/instructions file
	// with applyTo: backend/**, not backend/.github/.
	cp, err := os.ReadFile(".github/instructions/backend-api.instructions.md")
	is.NoErr(err)
	is.True(strings.Contains(string(cp), "backend/**"))
	is.True(strings.Contains(string(cp), "BACKEND RULE"))

	// No backend/.github tree should have been written.
	_, err = os.Stat("backend/.github")
	is.True(os.IsNotExist(err))
}

func TestGen_OutputIncludesBlockLineRange(t *testing.T) {
	// Block-mode writes should report the line range where the block
	// ended up, in the editor-jumpable `path:begin-end` notation.
	// File-mode writes print just the path.
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	// Pre-seed CLAUDE.md with a few lines of user content so the block
	// lands on a non-trivial line range.
	is.NoErr(os.WriteFile("CLAUDE.md", []byte("# Notes\n\nuser line one\nuser line two\n"), 0o644))

	stdout, _, err := invoke(t, "gen")
	is.NoErr(err)

	// File mode: bare path, no suffix.
	is.True(strings.Contains(stdout, "wrote .claude/skills/review-pr/SKILL.md\n"))

	// Block mode: "wrote CLAUDE.md:N-M". Exact line numbers depend on
	// content length, but the prefix and the general shape must match.
	var sawBlock bool
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "wrote CLAUDE.md:") && strings.Contains(line, "-") {
			sawBlock = true
		}
	}
	is.True(sawBlock) // wrote CLAUDE.md line should use path:begin-end notation
}

func TestGen_ShipsExecutableBashScriptInSkill(t *testing.T) {
	// A skill author can drop a bash script next to SKILL.md. After
	// `hatch gen`, every target that supports skill assets (claude, codex,
	// opencode) must have the script under its own skill dir, with content
	// preserved and the executable bit intact.
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(os.MkdirAll(filepath.Join(dir, ".hatch", "_skills", "run-checks", "scripts"), 0o755))
	is.NoErr(os.WriteFile(filepath.Join(dir, ".hatch", "_skills", "run-checks", "SKILL.md"),
		[]byte("---\ndescription: Run repo checks.\n---\nbody\n"), 0o644))

	scriptSrc := filepath.Join(dir, ".hatch", "_skills", "run-checks", "scripts", "check.sh")
	scriptBody := "#!/bin/sh\necho ok\n"
	is.NoErr(os.WriteFile(scriptSrc, []byte(scriptBody), 0o644))
	is.NoErr(os.Chmod(scriptSrc, 0o755))

	t.Chdir(dir)
	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	for _, rel := range []string{
		".claude/skills/run-checks/scripts/check.sh",
		".agents/skills/run-checks/scripts/check.sh",
		".opencode/skills/run-checks/scripts/check.sh",
	} {
		got, err := os.ReadFile(rel)
		is.NoErr(err)                     // skill script should be copied to target dir
		is.Equal(string(got), scriptBody) // script body must be preserved verbatim

		info, err := os.Stat(rel)
		is.NoErr(err)
		is.True(info.Mode()&0o100 != 0) // owner exec bit must survive the copy
	}
}

func TestGen_LegacyGenerateWordRemoved(t *testing.T) {
	// `hatch generate` (the old long form) should now be an unknown command.
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)
	_, _, err := invoke(t, "generate")
	is.True(err != nil)
}
