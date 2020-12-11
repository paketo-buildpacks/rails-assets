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

func Build(
	buildProcess BuildProcess,
	calculator Calculator,
	logger LogEmitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		assetsLayer, err := context.Layers.Get(LayerNameAssets)
		if err != nil {
			return packit.BuildResult{}, err
		}

		var sum string
		assetsDir := filepath.Join(context.WorkingDir, "app", "assets")
		_, err = os.Stat(assetsDir)
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to stat %s: %w", assetsDir, err)
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
				Layers: []packit.Layer{assetsLayer},
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

		publicAssetsDir := filepath.Join("public", "assets")
		tmpAssetsCacheDir := filepath.Join("tmp", "assets", "cache")

		err = os.Symlink(filepath.Join(assetsLayer.Path, publicAssetsDir), filepath.Join(context.WorkingDir, publicAssetsDir))
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to symlink public/assets into working directory: %w", err)
		}

		err = os.Symlink(filepath.Join(assetsLayer.Path, tmpAssetsCacheDir), filepath.Join(context.WorkingDir, tmpAssetsCacheDir))
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to symlink tmp/assets/cache into working directory: %w", err)
		}

		assetsLayer.Launch = true
		assetsLayer.LaunchEnv.Default("RAILS_ENV", "production")

		assetsLayer.Metadata = map[string]interface{}{
			"built_at":  clock.Now().Format(time.RFC3339Nano),
			"cache_sha": sum,
		}

		return packit.BuildResult{
			Layers: []packit.Layer{assetsLayer},
		}, nil
	}
}
