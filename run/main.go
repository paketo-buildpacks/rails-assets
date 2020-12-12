package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	railsassets "github.com/paketo-buildpacks/rails-assets"
)

func main() {
	logEmitter := railsassets.NewLogEmitter(os.Stdout)

	packit.Run(
		railsassets.Detect(railsassets.NewGemfileParser()),
		railsassets.Build(
			railsassets.NewPrecompileProcess(
				pexec.NewExecutable("bundle"),
				logEmitter,
			),
			fs.NewChecksumCalculator(),
			railsassets.NewDirectoriesSetup(),
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
