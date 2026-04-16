package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/grafana/hatch/pkg/target"
)

func cmdList(_ context.Context, version string, available *target.Set, args []string, stdout, stderr io.Writer) error {
	cf := commonFlags("list", stderr)
	if err := cf.fs.Parse(args); err != nil {
		return err
	}
	if err := ensureNoPositional(cf.fs, "list"); err != nil {
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
		arts, err := t.Generate(src)
		if err != nil {
			return fmt.Errorf("%s: %w", t.Name(), err)
		}
		fmt.Fprintf(stdout, "%s (%s):\n", t.DisplayName(), t.Name())
		if len(arts) == 0 {
			fmt.Fprintln(stdout, "  (nothing to generate)")
			continue
		}
		for _, a := range arts {
			fmt.Fprintf(stdout, "  %s  [%s]\n", a.Path, a.Mode)
		}
	}
	return nil
}
