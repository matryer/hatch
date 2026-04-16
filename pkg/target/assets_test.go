package target_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/hatch/pkg/source"
	"github.com/grafana/hatch/pkg/target"
	"github.com/matryer/is"
)

func TestCopySkillAssets_CopiesSiblingsExceptSKILL(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "review-pr")
	is.NoErr(os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755))
	is.NoErr(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill"), 0o644))
	is.NoErr(os.WriteFile(filepath.Join(skillDir, "scripts", "review.sh"), []byte("#!/bin/sh\n"), 0o644))
	is.NoErr(os.WriteFile(filepath.Join(skillDir, "rubric.md"), []byte("criteria"), 0o644))

	sk := source.Primitive{Kind: source.KindSkill, Name: "review-pr", SourcePath: skillDir}
	arts, err := target.CopySkillAssets(sk, "dest")
	is.NoErr(err)

	byPath := map[string]string{}
	for _, a := range arts {
		byPath[a.Path] = a.Content
	}

	// SKILL.md is skipped (the target generates its own).
	_, hasSkillMD := byPath[filepath.Join("dest", "SKILL.md")]
	is.True(!hasSkillMD)

	is.Equal(byPath[filepath.Join("dest", "rubric.md")], "criteria")
	is.Equal(byPath[filepath.Join("dest", "scripts", "review.sh")], "#!/bin/sh\n")
}

func TestCopySkillAssets_SkipsHiddenDirs(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "review-pr")
	is.NoErr(os.MkdirAll(filepath.Join(skillDir, ".hidden"), 0o755))
	is.NoErr(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("s"), 0o644))
	is.NoErr(os.WriteFile(filepath.Join(skillDir, ".hidden", "secret.txt"), []byte("x"), 0o644))

	sk := source.Primitive{Kind: source.KindSkill, SourcePath: skillDir}
	arts, err := target.CopySkillAssets(sk, "dest")
	is.NoErr(err)
	is.Equal(len(arts), 0)
}

func TestCopySkillAssets_NestedDestDir(t *testing.T) {
	// When the caller passes a scoped destDir (e.g. backend/.claude/skills/x),
	// every artifact path must be prefixed with that destDir. This pins the
	// already-correct CopySkillAssets behaviour so the nested-paths feature
	// can rely on it.
	is := is.New(t)
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "review-pr")
	is.NoErr(os.MkdirAll(skillDir, 0o755))
	is.NoErr(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("s"), 0o644))
	is.NoErr(os.WriteFile(filepath.Join(skillDir, "rubric.md"), []byte("r"), 0o644))

	sk := source.Primitive{Kind: source.KindSkill, Name: "review-pr", SourcePath: skillDir}
	dest := filepath.Join("backend", ".claude", "skills", "review-pr")
	arts, err := target.CopySkillAssets(sk, dest)
	is.NoErr(err)
	is.Equal(len(arts), 1)
	is.Equal(arts[0].Path, filepath.Join(dest, "rubric.md"))
}

func TestCopySkillAssets_NoSourcePathReturnsNil(t *testing.T) {
	is := is.New(t)
	sk := source.Primitive{Kind: source.KindSkill}
	arts, err := target.CopySkillAssets(sk, "dest")
	is.NoErr(err)
	is.Equal(len(arts), 0)
}
