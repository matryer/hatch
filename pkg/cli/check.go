package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/grafana/hatch/pkg/block"
	"github.com/grafana/hatch/pkg/target"
)

// cmdCheck is the implementation of `hatch check`. It re-derives what a
// fresh `hatch gen` would write and compares it to the files already on
// disk, without touching the filesystem. Exits non-zero when any file is
// out of date — the intended use is a CI gate that asserts the checked-in
// generated files match the current `.hatch/` source.
func cmdCheck(ctx context.Context, version string, available *target.Set, args []string, stdout, stderr io.Writer) error {
	cf := commonFlags("check", stderr)
	if err := cf.fs.Parse(args); err != nil {
		return err
	}
	if err := ensureNoPositional(cf.fs, "check"); err != nil {
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

	byPath := map[string][]pending{}
	for _, t := range targets.All() {
		if err := ctx.Err(); err != nil {
			return err
		}
		arts, err := t.Generate(src)
		if err != nil {
			return fmt.Errorf("%s: %w", t.Name(), err)
		}
		for _, a := range arts {
			byPath[a.Path] = append(byPath[a.Path], pending{artifact: a, source: t.Name()})
		}
	}

	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var drifted []string
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		merged, err := mergeArtifacts(p, byPath[p])
		if err != nil {
			return err
		}
		reason, err := checkArtifact(merged)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		if reason != "" {
			drifted = append(drifted, p)
			fmt.Fprintf(stdout, "out-of-date  %s  (%s)\n", p, reason)
		}
	}

	if len(drifted) > 0 {
		return fmt.Errorf("%d file(s) out of date; run `hatch gen` to update", len(drifted))
	}
	fmt.Fprintf(stdout, "all %d generated file(s) up to date\n", len(paths))
	return nil
}

// checkArtifact returns the empty string when the file on disk already
// matches what `hatch gen` would write, or a short reason string ("missing",
// "stale", "exec bit") describing the drift. It never writes.
func checkArtifact(a target.Artifact) (string, error) {
	switch a.Mode {
	case target.ModeFile:
		got, err := os.ReadFile(a.Path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "missing", nil
			}
			return "", err
		}
		if string(got) != a.Content {
			return "stale", nil
		}
		if a.Executable {
			info, err := os.Stat(a.Path)
			if err != nil {
				return "", err
			}
			if info.Mode()&0o100 == 0 {
				return "exec bit", nil
			}
		}
		return "", nil
	case target.ModeBlock:
		got, err := os.ReadFile(a.Path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "missing", nil
			}
			return "", err
		}
		want, err := block.Plan(a.Path, block.CurrentMarker, a.Content)
		if err != nil {
			return "", err
		}
		if !bytes.Equal(got, want) {
			return "stale", nil
		}
		return "", nil
	default:
		return "", fmt.Errorf("unknown write mode %q", a.Mode)
	}
}
