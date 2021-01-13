package railsassets

import (
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
)

const (
	LayerNameAssets = "assets"
)

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	Execute(workingDir string) error
}

//go:generate faux --interface Calculator --output fakes/calculator.go
type Calculator interface {
	Sum(paths ...string) (string, error)
}

//go:generate faux --interface EnvironmentSetup --output fakes/environment_setup.go
type EnvironmentSetup interface {
	ResetLocal(workingDir string) error
	ResetLayer(layerPath string) error
	Link(layerPath, workingDir string) error
}

func Build(
	buildProcess BuildProcess,
	calculator Calculator,
	environmentSetup EnvironmentSetup,
	logger LogEmitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		assetsLayer, err := context.Layers.Get(LayerNameAssets)
		if err != nil {
			return packit.BuildResult{}, err
		}

		appAssetsDir := filepath.Join(context.WorkingDir, "app", "assets")
		sum, err := calculator.Sum(appAssetsDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = environmentSetup.ResetLocal(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		previousSum, _ := assetsLayer.Metadata["cache_sha"].(string)
		if sum == previousSum {
			logger.Process("Reusing cached layer %s", assetsLayer.Path)
			logger.Break()

			err = environmentSetup.Link(assetsLayer.Path, context.WorkingDir)
			if err != nil {
				return packit.BuildResult{}, err
			}

			return packit.BuildResult{
				Layers: []packit.Layer{assetsLayer},
			}, nil
		}

		err = environmentSetup.ResetLayer(assetsLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = environmentSetup.Link(assetsLayer.Path, context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			return buildProcess.Execute(context.WorkingDir)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		assetsLayer.Launch = true
		assetsLayer.LaunchEnv.Default("RAILS_ENV", "production")
		assetsLayer.LaunchEnv.Default("RAILS_SERVE_STATIC_FILES", "true")

		logger.Environment(assetsLayer.LaunchEnv)

		assetsLayer.Metadata = map[string]interface{}{
			"built_at":  clock.Now().Format(time.RFC3339Nano),
			"cache_sha": sum,
		}

		return packit.BuildResult{
			Layers: []packit.Layer{assetsLayer},
		}, nil
	}
}
