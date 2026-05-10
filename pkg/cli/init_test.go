package cli_test

import (
	"os"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestInit_CreatesEmptyDirsByDefault(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	t.Chdir(dir)

	_, _, err := invoke(t, "init")
	is.NoErr(err)

	// The four primitive container subdirectories should exist.
	for _, rel := range []string{
		".hatch/_rules",
		".hatch/_skills",
		".hatch/_commands",
		".hatch/_agents",
	} {
		info, err := os.Stat(rel)
		is.NoErr(err) // subdir should exist
		is.True(info.IsDir())
	}

	// No example files should have been written.
	for _, rel := range []string{
		".hatch/_rules/coding-style.md",
		".hatch/_skills/review-pr/SKILL.md",
		".hatch/_commands/commit.md",
		".hatch/_agents/security-auditor.md",
	} {
		_, err := os.Stat(rel)
		is.True(os.IsNotExist(err)) // example file must not exist on bare init
	}
}

func TestInit_ExamplesFlagScaffoldsAllPrimitives(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	t.Chdir(dir)

	stdout, _, err := invoke(t, "init", "-examples")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "created"))

	for _, rel := range []string{
		".hatch/_rules/coding-style.md",
		".hatch/_skills/review-pr/SKILL.md",
		".hatch/_skills/review-pr/scripts/checkout-pr.sh",
		".hatch/_commands/commit.md",
		".hatch/_agents/security-auditor.md",
	} {
		_, err := os.Stat(rel)
		is.NoErr(err) // scaffold file should exist
	}

	// The seeded bash script must be executable so the round-tripped copy
	// in each target's skill dir inherits the exec bit.
	info, err := os.Stat(".hatch/_skills/review-pr/scripts/checkout-pr.sh")
	is.NoErr(err)
	is.True(info.Mode()&0o100 != 0) // seeded script must be executable
}

func TestInit_ExamplesPlusGen_ShipsSkillScriptToTargets(t *testing.T) {
	// End-to-end: hatch init -examples seeds a skill that ships a bash
	// script; hatch gen must copy that script into each target's skill dir,
	// executable.
	is := is.New(t)
	dir := t.TempDir()
	t.Chdir(dir)

	_, _, err := invoke(t, "init", "-examples")
	is.NoErr(err)
	_, _, err = invoke(t, "gen")
	is.NoErr(err)

	for _, rel := range []string{
		".claude/skills/review-pr/scripts/checkout-pr.sh",
		".agents/skills/review-pr/scripts/checkout-pr.sh",
		".opencode/skills/review-pr/scripts/checkout-pr.sh",
	} {
		info, err := os.Stat(rel)
		is.NoErr(err)                   // script must be copied to target skill dir
		is.True(info.Mode()&0o100 != 0) // and stay executable
	}
}

func TestInit_PathFlag_ScaffoldsNestedDirs(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	t.Chdir(dir)

	_, _, err := invoke(t, "init", "-path", "backend")
	is.NoErr(err)

	for _, rel := range []string{
		".hatch/backend/_rules",
		".hatch/backend/_skills",
		".hatch/backend/_commands",
		".hatch/backend/_agents",
	} {
		info, err := os.Stat(rel)
		is.NoErr(err)
		is.True(info.IsDir())
	}
	// Bare .hatch/_rules should NOT be created when -path is given.
	_, err = os.Stat(".hatch/_rules")
	is.True(os.IsNotExist(err))
}

func TestInit_PathFlag_RejectsKnownPrimitiveContainerName(t *testing.T) {
	// `hatch init -path _rules` would write the new dirs at
	// .hatch/_rules/_rules/, .hatch/_rules/_skills/ etc — which the walker
	// can't load because the outer _rules is treated as a primitive
	// container, not a path component. Reject loudly.
	is := is.New(t)
	dir := t.TempDir()
	t.Chdir(dir)

	for _, p := range []string{"_rules", "_skills", "_commands", "_agents"} {
		_, _, err := invoke(t, "init", "-path", p)
		is.True(err != nil)
	}

	// But a non-primitive _xxx name is allowed.
	_, _, err := invoke(t, "init", "-path", "_workflows")
	is.NoErr(err)
	info, err := os.Stat(".hatch/_workflows/_rules")
	is.NoErr(err)
	is.True(info.IsDir())
}

func TestInit_PathFlag_RejectsTraversalAndAbsolute(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	t.Chdir(dir)

	for _, p := range []string{"../foo", "/abs", "foo//bar", "foo/../bar"} {
		_, _, err := invoke(t, "init", "-path", p)
		is.True(err != nil)
	}
}

func TestInit_ExamplesFlagSkipsExistingFiles(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	t.Chdir(dir)

	// First init -examples creates files.
	_, _, err := invoke(t, "init", "-examples")
	is.NoErr(err)

	// Overwrite a file with custom content.
	rulePath := ".hatch/_rules/coding-style.md"
	is.NoErr(os.WriteFile(rulePath, []byte("custom content"), 0o644))

	// Second init -examples should NOT overwrite the user's custom content.
	_, _, err = invoke(t, "init", "-examples")
	is.NoErr(err)
	got, err := os.ReadFile(rulePath)
	is.NoErr(err)
	is.Equal(string(got), "custom content")
}
