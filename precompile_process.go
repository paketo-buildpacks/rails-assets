package railsassets

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/paketo-buildpacks/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) error
}

type PrecompileProcess struct {
	executable Executable
	logger     LogEmitter
}

func NewPrecompileProcess(executable Executable, logger LogEmitter) PrecompileProcess {
	return PrecompileProcess{
		executable: executable,
		logger:     logger,
	}
}

func (p PrecompileProcess) Execute(workingDir string) error {
	os.Setenv("RAILS_ENV", "production")

	buffer := bytes.NewBuffer(nil)
	args := []string{"exec", "rails", "assets:precompile", "assets:clean"}

	p.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))
	err := p.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		return fmt.Errorf("failed to execute bundle exec output:\n%s\nerror: %s", buffer.String(), err)
	}

	return nil
}
