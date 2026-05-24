package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestCheck_AfterGen_NoDrift(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	stdout, _, err := invoke(t, "check")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "up to date"))
	is.True(!strings.Contains(stdout, "out-of-date"))
}

func TestCheck_NeverGenerated_ReportsMissing(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	stdout, _, err := invoke(t, "check")
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "out of date"))
	is.True(strings.Contains(stdout, "out-of-date"))
	is.True(strings.Contains(stdout, "(missing)"))
}

func TestCheck_StaleBlockFile_ReportsStale(t *testing.T) {
	// Tamper with a block-mode generated file (CLAUDE.md): after `hatch gen`
	// rewrite the hatch block to something different. check must catch it.
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	// Overwrite CLAUDE.md with hand-edited block content.
	tampered := "<!-- hatch:begin v1 -->\nnot what gen would write\n<!-- hatch:end v1 -->\n"
	is.NoErr(os.WriteFile("CLAUDE.md", []byte(tampered), 0o644))

	stdout, _, err := invoke(t, "check")
	is.True(err != nil)
	is.True(strings.Contains(stdout, "out-of-date  CLAUDE.md"))
	is.True(strings.Contains(stdout, "(stale)"))
}

func TestCheck_StaleFileMode_ReportsStale(t *testing.T) {
	// Tamper with a ModeFile artifact (a SKILL.md): check must catch
	// byte-level drift even when the file exists.
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	path := filepath.Join(".claude", "skills", "review-pr", "SKILL.md")
	is.NoErr(os.WriteFile(path, []byte("tampered\n"), 0o644))

	stdout, _, err := invoke(t, "check")
	is.True(err != nil)
	is.True(strings.Contains(stdout, "out-of-date  "+path))
	is.True(strings.Contains(stdout, "(stale)"))
}

func TestCheck_DoesNotWriteFiles(t *testing.T) {
	// check must never touch the filesystem — running it from a clean
	// project (no `hatch gen` first) must leave the project clean.
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "check")
	is.True(err != nil) // missing files → drift

	for _, path := range []string{
		"CLAUDE.md",
		"AGENTS.md",
		".claude/skills/review-pr/SKILL.md",
	} {
		_, statErr := os.Stat(path)
		is.True(os.IsNotExist(statErr)) // check must not create files
	}
}

func TestCheck_PreservesUserContentDetection(t *testing.T) {
	// User content around a hatch block is part of the file but not the
	// block — check must compare the file as a whole, so changes to user
	// content do NOT count as drift (since gen would preserve them).
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	userContent := "# My notes\n\nHand-written content here.\n"
	is.NoErr(os.WriteFile("CLAUDE.md", []byte(userContent), 0o644))

	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	// Edit the user-authored part outside the block; gen would still
	// produce the same overall file, so check should pass.
	with, err := os.ReadFile("CLAUDE.md")
	is.NoErr(err)
	edited := strings.Replace(string(with), "Hand-written content here.", "Hand-edited content here.", 1)
	is.NoErr(os.WriteFile("CLAUDE.md", []byte(edited), 0o644))

	stdout, _, err := invoke(t, "check")
	is.NoErr(err) // user-content edit outside the block is not drift
	is.True(!strings.Contains(stdout, "out-of-date  CLAUDE.md"))
}

func TestCheck_TargetsFlag(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)

	_, _, err := invoke(t, "gen", "-targets", "claude")
	is.NoErr(err)

	// Only claude was generated, but check defaults to all targets — so
	// missing AGENTS.md (Codex) should fail.
	_, _, err = invoke(t, "check")
	is.True(err != nil)

	// Narrowed to claude, the check passes.
	stdout, _, err := invoke(t, "check", "-targets", "claude")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "up to date"))
}

func TestCheck_PositionalArgErrors(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	scaffoldSource(t, dir)
	t.Chdir(dir)
	_, _, err := invoke(t, "check", "claude")
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "unexpected argument"))
}

func TestCheck_ExecBitDrift(t *testing.T) {
	// A skill ships an executable sibling script. After gen, the script
	// has the exec bit. Clearing the bit (chmod 644) is drift that check
	// must detect even though byte content matches.
	is := is.New(t)
	dir := t.TempDir()
	is.NoErr(os.MkdirAll(filepath.Join(dir, ".hatch", "_skills", "run-checks", "scripts"), 0o755))
	is.NoErr(os.WriteFile(filepath.Join(dir, ".hatch", "_skills", "run-checks", "SKILL.md"),
		[]byte("---\ndescription: Run repo checks.\n---\nbody\n"), 0o644))
	scriptSrc := filepath.Join(dir, ".hatch", "_skills", "run-checks", "scripts", "check.sh")
	is.NoErr(os.WriteFile(scriptSrc, []byte("#!/bin/sh\necho ok\n"), 0o644))
	is.NoErr(os.Chmod(scriptSrc, 0o755))

	t.Chdir(dir)
	_, _, err := invoke(t, "gen")
	is.NoErr(err)

	dst := ".claude/skills/run-checks/scripts/check.sh"
	is.NoErr(os.Chmod(dst, 0o644)) // clear exec bit

	stdout, _, err := invoke(t, "check")
	is.True(err != nil)
	is.True(strings.Contains(stdout, "out-of-date  "+dst))
	is.True(strings.Contains(stdout, "(exec bit)"))
}
