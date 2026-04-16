package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/grafana/hatch/pkg/block"
	"github.com/grafana/hatch/pkg/target"
)

// cmdClean re-derives what a fresh `hatch gen` would write from the current
// source tree, then removes those files (for ModeFile) or strips just the
// hatch block (for ModeBlock). No manifest is kept.
func cmdClean(ctx context.Context, version string, available *target.Set, args []string, stdout, stderr io.Writer) error {
	cf := commonFlags("clean", stderr)
	if err := cf.fs.Parse(args); err != nil {
		return err
	}
	if err := ensureNoPositional(cf.fs, "clean"); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	targets, err := selectTargets(available, *cf.targetsList)
	if err != nil {
		return err
	}
	src, err := loadSource(!*cf.noHatchSkill, version)
	if err != nil {
		return err
	}

	for _, t := range targets.All() {
		if err := ctx.Err(); err != nil {
			return err
		}
		arts, err := t.Generate(src)
		if err != nil {
			return fmt.Errorf("%s: %w", t.Name(), err)
		}
		for _, a := range arts {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := cleanArtifact(a); err != nil {
				fmt.Fprintf(stderr, "hatch: %s: %s\n", a.Path, err)
				continue
			}
			fmt.Fprintf(stdout, "removed %s (%s)\n", a.Path, a.Mode)
		}
	}
	return nil
}

func cleanArtifact(a target.Artifact) error {
	switch a.Mode {
	case target.ModeFile:
		if err := os.Remove(a.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
		pruneEmptyDirs(filepath.Dir(a.Path))
		return nil
	case target.ModeBlock:
		return block.Strip(a.Path, block.CurrentMarker)
	default:
		return fmt.Errorf("unknown write mode %q", a.Mode)
	}
}

// pruneEmptyDirs walks upward from dir, removing directories that become
// empty, stopping at the current working directory.
func pruneEmptyDirs(dir string) {
	cwd, _ := os.Getwd()
	absCwd, _ := filepath.Abs(cwd)
	for {
		absDir, _ := filepath.Abs(dir)
		if absDir == absCwd || len(absDir) <= len(absCwd) {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
