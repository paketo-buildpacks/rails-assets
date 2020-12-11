package railsassets

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
)

const (
	LayerNameRails = "rails"
)

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	Execute(workingDir string) error
}

//go:generate faux --interface Calculator --output fakes/calculator.go
type Calculator interface {
	Sum(paths ...string) (string, error)
}

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve([]packit.BuildpackPlanEntry) packit.BuildpackPlanEntry
}

func Build(
	buildProcess BuildProcess,
	calculator Calculator,
	logger LogEmitter,
	clock chronos.Clock,
	entries EntryResolver,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		entry := entries.Resolve(context.Plan.Entries)

		assetsLayer, err := context.Layers.Get(LayerNameRails)
		if err != nil {
			return packit.BuildResult{}, err
		}

		var sum string
		assetsDir := filepath.Join(context.WorkingDir, "app", "assets")
		_, err = os.Stat(assetsDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return packit.BuildResult{}, fmt.Errorf("failed to stat %s: %w", assetsDir, err)
			}
		} else {
			sum, err = calculator.Sum(assetsDir)
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		cachedSHA, ok := assetsLayer.Metadata["cache_sha"].(string)
		if ok && cachedSHA != "" && cachedSHA == sum {
			logger.Process("Reusing cached layer %s", assetsLayer.Path)
			logger.Break()

			return packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{entry},
				},
				Layers: []packit.Layer{assetsLayer},
			}, nil
		}

		// Unable to reuse cache layer, so bring back public/assets & tmp/assets/cache for the bundle command
		err = os.Symlink(filepath.Join(assetsLayer.Path, "public", "assets"), filepath.Join(context.WorkingDir, "public", "assets"))
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to symlink public/assets into working directory: %w", err)
		}
		err = os.Symlink(filepath.Join(assetsLayer.Path, "tmp", "assets", "cache"), filepath.Join(context.WorkingDir, "tmp", "assets", "cache"))
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to symlink tmp/assets/cache into working directory: %w", err)
		}

		os.Setenv("RAILS_ENV", "production")

		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			return buildProcess.Execute(context.WorkingDir)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		assetsLayer.LaunchEnv.Default("RAILS_ENV", "production")

		// assetsLayer.Launch = entry.Metadata["launch"] == true
		// assetsLayer.Build = entry.Metadata["build"] == true
		// assetsLayer.Cache = entry.Metadata["build"] == true
		assetsLayer.Metadata = map[string]interface{}{
			"built_at":  clock.Now().Format(time.RFC3339Nano),
			"cache_sha": sum,
		}

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{entry},
			},
			Layers: []packit.Layer{assetsLayer},
		}, nil
	}
}
