package railsassets_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitRails(t *testing.T) {
	suite := spec.New("rails", spec.Report(report.Terminal{}))
	suite("Detect", testDetect)
	suite.Run(t)
}
