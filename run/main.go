package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	railsassets "github.com/paketo-buildpacks/rails-assets"
)

func main() {
	logger := scribe.NewLogger(os.Stdout)

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
