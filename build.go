package railsassets

import (
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/scribe"
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
//   the buildpack. These locations include public/assets and tmp/cache and all
//   extra directories defined by the user.
//   2. Calculate a checksum of the asset directories that appear in the
//   working directory. These directories include app/assets, lib/assets,
//   vendor/assets, app/javascript, and the user defined checksum directories.
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
//      * RAILS_LOG_TO_STDOUT=true : Rails will log to stdout
//   7. Attach build metadata onto the new "assets" layer so that it can be
//   referenced in future builds.
func Build(
	buildProcess BuildProcess,
	calculator Calculator,
	environmentSetup EnvironmentSetup,
	logger scribe.Emitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		err := environmentSetup.ResetLocal(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Debug.Process("Checking checksum paths for the following directories:")
		var checksumPaths []string

		paths := []string{
			filepath.Join(context.WorkingDir, "app", "assets"),
			filepath.Join(context.WorkingDir, "lib", "assets"),
			filepath.Join(context.WorkingDir, "vendor", "assets"),
			filepath.Join(context.WorkingDir, "app", "javascript"),
		}

		extraPaths := filepath.SplitList(os.Getenv("BP_RAILS_ASSETS_EXTRA_SOURCE_PATHS"))
		for _, path := range extraPaths {
			paths = append(paths, filepath.Join(context.WorkingDir, filepath.Clean(path)))
		}

		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				logger.Debug.Subprocess(path)
				checksumPaths = append(checksumPaths, path)
			}
		}
		logger.Debug.Break()

		sum, err := calculator.Sum(checksumPaths...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Debug.Process("Getting the layer associated with Rails assets:")
		assetsLayer, err := context.Layers.Get(LayerNameAssets)
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Debug.Subprocess(assetsLayer.Path)
		logger.Debug.Break()

		previousSum, _ := assetsLayer.Metadata["cache_sha"].(string)
		if sum == previousSum {
			logger.Process("Reusing cached layer %s", assetsLayer.Path)

			assetsLayer.Launch = true
			logger.Debug.Process("Symlinking asset directories to %s", context.WorkingDir)
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

		logger.Debug.Process("Symlinking asset directories to %s", context.WorkingDir)
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
		assetsLayer.LaunchEnv.Default("RAILS_LOG_TO_STDOUT", "true")
		logger.EnvironmentVariables(assetsLayer)

		assetsLayer.Metadata = map[string]interface{}{
			"cache_sha": sum,
		}

		return packit.BuildResult{
			Layers: []packit.Layer{assetsLayer},
		}, nil
	}
}
