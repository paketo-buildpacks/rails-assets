package railsassets

import (
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
)

const (
	// LayerNameAssets is the name of the layer that is used to store asset
	// contents.
	LayerNameAssets = "assets"
)

//go:generate faux --interface BuildProcess --output fakes/build_process.go
//go:generate faux --interface Calculator --output fakes/calculator.go
//go:generate faux --interface EnvironmentSetup --output fakes/environment_setup.go

// BuildProcess defines the interface for executing the "rails
// assets:precompile" build process.
type BuildProcess interface {
	Execute(workingDir string) error
}

// Calculator defines the interface for calculating a checksum of a given set
// of file paths.
type Calculator interface {
	Sum(paths ...string) (string, error)
}

// EnvironmentSetup defines the interface for setting up the working directory
// and linking into layers created by the build phase.
type EnvironmentSetup interface {
	ResetLocal(workingDir string) error
	ResetLayer(layerPath string) error
	Link(layerPath, workingDir string) error
}

// Build will return a packit.BuildFunc that will be invoked during the build
// phase of the buildpack lifecycle.
//
// Build will perform the following steps to execute its build process:
//   1. Reset the local working directory locations that will be modified by
//   the buildpack. These locations include public/assets and tmp/cache.
//   2. Calculate a checksum of the asset directories that appear in the
//   working directory. These directories include app/assets, lib/assets,
//   vendor/assets, and app/javascript.
//   3. Compare the calculated checksum against the recorded value on the
//   "assets" layer metadata.
//   3a. If the checksum matches the recorded value, the build process
//   completes without modifying the existing layer contents.
//   4. If the checksum does not match, then the "assets" layer contents are
//   cleared.
//   5. The "rails assets:precompile" build process is executed.
//   6. The launch environment is configured with the following environment variables:
//      * RAILS_ENV=production : run Rails in its "production" configuration
//      * RAILS_SERVE_STATIC_FILES : configure Rails to serve static files
//      itself instead of expecting that a file server like NGINX will serve
//      them
//   7. Attach build metadata onto the new "assets" layer so that it can be
//   referenced in future builds.
func Build(
	buildProcess BuildProcess,
	calculator Calculator,
	environmentSetup EnvironmentSetup,
	logger scribe.Logger,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		err := environmentSetup.ResetLocal(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		var checksumPaths []string
		for _, path := range []string{
			filepath.Join(context.WorkingDir, "app", "assets"),
			filepath.Join(context.WorkingDir, "lib", "assets"),
			filepath.Join(context.WorkingDir, "vendor", "assets"),
			filepath.Join(context.WorkingDir, "app", "javascript"),
		} {
			if _, err := os.Stat(path); err == nil {
				checksumPaths = append(checksumPaths, path)
			}
		}

		sum, err := calculator.Sum(checksumPaths...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		assetsLayer, err := context.Layers.Get(LayerNameAssets)
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

		logger.Process("Configuring launch environment")
		logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(assetsLayer.LaunchEnv))
		logger.Break()

		assetsLayer.Metadata = map[string]interface{}{
			"built_at":  clock.Now().Format(time.RFC3339Nano),
			"cache_sha": sum,
		}

		return packit.BuildResult{
			Layers: []packit.Layer{assetsLayer},
		}, nil
	}
}
