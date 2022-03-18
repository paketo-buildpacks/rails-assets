package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	railsassets "github.com/paketo-buildpacks/rails-assets"
)

func main() {
	logger := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	packit.Run(
		railsassets.Detect(railsassets.NewGemfileParser()),
		railsassets.Build(
			railsassets.NewPrecompileProcess(
				pexec.NewExecutable("bundle"),
				logger,
			),
			fs.NewChecksumCalculator(),
			railsassets.NewDirectorySetup(),
			logger,
			chronos.DefaultClock,
		),
	)
}
