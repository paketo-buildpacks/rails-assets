package railsassets

import (
	"os"
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

func Build(buildProcess BuildProcess, logger LogEmitter, clock chronos.Clock) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		os.Setenv("RAILS_ENV", "production")

		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			return buildProcess.Execute(context.WorkingDir)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		railsLayer, err := context.Layers.Get(LayerNameRails, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}
		railsLayer.LaunchEnv.Default("RAILS_ENV", "production")

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		return packit.BuildResult{
			Layers: []packit.Layer{railsLayer},
		}, nil
	}
}
