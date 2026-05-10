package target

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/hatch/pkg/source"
)

// CopySkillAssets returns artifacts for every sibling file of a skill's
// SKILL.md that should be copied through verbatim to the target's skill
// directory. Skipped: SKILL.md itself, hidden dotfiles, and the root
// directory marker.
func CopySkillAssets(sk source.Primitive, destDir string) ([]Artifact, error) {
	if sk.SourcePath == "" {
		return nil, nil
	}
	info, err := os.Stat(sk.SourcePath)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	var out []Artifact
	err = filepath.WalkDir(sk.SourcePath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != sk.SourcePath && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if d.Name() == "SKILL.md" {
			return nil
		}
		rel, err := filepath.Rel(sk.SourcePath, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		out = append(out, Artifact{
			Path:       filepath.Join(destDir, rel),
			Mode:       ModeFile,
			Content:    string(data),
			Executable: info.Mode()&0o100 != 0,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
