package railsassets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface Parser --output fakes/parser.go
type Parser interface {
	Parse(path string) (hasRails bool, err error)
}

type BuildPlanMetadata struct {
	Build bool `toml:"build"`
}

func Detect(gemfileParser Parser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		_, err := os.Stat(filepath.Join(context.WorkingDir, "app", "assets"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail
			}

			return packit.DetectResult{}, fmt.Errorf("failed to stat app/assets: %w", err)
		}

		hasRails, err := gemfileParser.Parse(filepath.Join(context.WorkingDir, "Gemfile"))
		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("failed to parse Gemfile: %w", err)
		}

		if !hasRails {
			return packit.DetectResult{}, packit.Fail
		}

		// For Rails 5, we only need a Node.js runtime
		nodeOrModules := packit.BuildPlanRequirement{
			Name: "node",
			Metadata: BuildPlanMetadata{
				Build: true,
			},
		}

		// For Rails 6, we need a Node.js runtime, yarn, and we want to run yarn
		// install ahead of asset compilation. We can detect this case by the
		// presence of a yarn.lock file. In that case, we will switch the node
		// requirement to node_modules, which should trigger the yarn-install
		// buildpack detection.
		_, err = os.Stat(filepath.Join(context.WorkingDir, "yarn.lock"))
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, fmt.Errorf("failed to stat yarn.lock: %w", err)
			}
		}
		if err == nil {
			nodeOrModules = packit.BuildPlanRequirement{
				Name: "node_modules",
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			}
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "gems",
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: "bundler",
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: "mri",
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
					nodeOrModules,
				},
			},
		}, nil
	}
}
