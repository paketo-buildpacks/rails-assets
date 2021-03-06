package railsassets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface Parser --output fakes/parser.go

// Parser defines the interface for determining if the Gemfile contains the
// "rails" gem.
type Parser interface {
	Parse(path string) (hasRails bool, err error)
}

// BuildPlanMetadata declares the set of metadata included in build plan
// requirements.
type BuildPlanMetadata struct {

	// Build is set to true when the build plan requirement should be made
	// available during the build phase of the buildpack lifecycle.
	Build bool `toml:"build"`
}

// Detect will return a packit.DetectFunc that will be invoked during the
// detect phase of the buildpack lifecycle.
//
// The detection criteria is twofold:
//   1. An assets directory must be present in the application source code.
//   These directories include app/assets, lib/assets, vendor/assets, and
//   app/javascript.
//   2. The Gemfile must reference the "rails" gem.
//
// If both of these criteria are met, then the buildpack will require "node",
// "mri", "bundler", and "gems" as build-time build plan requirements.
//
// Additionally, for Rails 6, we want to run yarn install ahead of asset
// compilation. We can detect this case by the presence of a yarn.lock file. In
// that case, the buildpack will also require "node_modules" as a build-time
// build plan requirement.
func Detect(gemfileParser Parser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		hasAssetsDirectory := false
		for _, path := range []string{
			filepath.Join(context.WorkingDir, "app", "assets"),
			filepath.Join(context.WorkingDir, "lib", "assets"),
			filepath.Join(context.WorkingDir, "vendor", "assets"),
			filepath.Join(context.WorkingDir, "app", "javascript"),
		} {
			_, err := os.Stat(path)
			if err == nil {
				hasAssetsDirectory = true
				break
			} else {
				if !errors.Is(err, os.ErrNotExist) {
					return packit.DetectResult{}, fmt.Errorf("failed to stat app/assets: %w", err)
				}
			}
		}

		if !hasAssetsDirectory {
			return packit.DetectResult{}, packit.Fail.WithMessage("failed to find assets in app/assets, app/javascript, lib/assets, or vendor/assets")
		}

		hasRails, err := gemfileParser.Parse(filepath.Join(context.WorkingDir, "Gemfile"))
		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("failed to parse Gemfile: %w", err)
		}

		if !hasRails {
			return packit.DetectResult{}, packit.Fail.WithMessage("failed to find rails gem in Gemfile")
		}

		requirements := []packit.BuildPlanRequirement{
			{
				Name: "node",
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
			{
				Name: "bundler",
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			},
			{
				Name: "gems",
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			},
		}

		_, err = os.Stat(filepath.Join(context.WorkingDir, "yarn.lock"))
		if err == nil {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "node_modules",
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			})
		} else {
			if !errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, fmt.Errorf("failed to stat yarn.lock: %w", err)
			}
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{},
				Requires: requirements,
			},
		}, nil
	}
}
