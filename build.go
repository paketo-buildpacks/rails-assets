package railsassets

import (
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
type InstallProcess interface {
	Execute(workingDir string) error
}

func Build(
	installProcess InstallProcess,
	logger LogEmitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			return installProcess.Execute(context.WorkingDir)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		return packit.BuildResult{}, nil
	}
}
