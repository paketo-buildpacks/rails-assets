package railsassets_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitRails(t *testing.T) {
	suite := spec.New("railsassets", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("DirectorySetup", testDirectorySetup)
	suite("GemfileParser", testGemfileParser)
	suite("PrecompileProcess", testPrecompileProcess)
	suite.Run(t)
}
