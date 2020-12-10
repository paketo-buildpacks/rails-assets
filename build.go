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

		railsLayer, err := context.Layers.Get(LayerNameRails)
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

		cachedSHA, ok := railsLayer.Metadata["cache_sha"].(string)
		if ok && cachedSHA != "" && cachedSHA == sum {
			logger.Process("Reusing cached layer %s", railsLayer.Path)
			logger.Break()

			return packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{entry},
				},
				Layers: []packit.Layer{railsLayer},
			}, nil
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

		railsLayer.LaunchEnv.Default("RAILS_ENV", "production")

		// railsLayer.Launch = entry.Metadata["launch"] == true
		// railsLayer.Build = entry.Metadata["build"] == true
		// railsLayer.Cache = entry.Metadata["build"] == true
		railsLayer.Metadata = map[string]interface{}{
			"built_at":  clock.Now().Format(time.RFC3339Nano),
			"cache_sha": sum,
		}

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{entry},
			},
			Layers: []packit.Layer{railsLayer},
		}, nil
	}
}
