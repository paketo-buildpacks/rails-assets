package railsassets

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go

// Executable defines the interface for executing a program as a child process.
type Executable interface {
	Execute(pexec.Execution) error
}

// PrecompileProcess performs the "rails assets:precompile" build process.
type PrecompileProcess struct {
	executable Executable
	logger     scribe.Emitter
}

// NewPrecompileProcess initializes an instance of PrecompileProcess.
func NewPrecompileProcess(executable Executable, logger scribe.Emitter) PrecompileProcess {
	return PrecompileProcess{
		executable: executable,
		logger:     logger,
	}
}

// Execute runs "bundle exec rails assets:precompile assets:clean" as a child
// process. If the process fails, the error message will include the entire
// output of the child process.
func (p PrecompileProcess) Execute(workingDir string) error {
	buffer := bytes.NewBuffer(nil)
	args := []string{"exec", "rails", "assets:precompile", "assets:clean"}

	p.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))
	err := p.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: p.logger.ActionWriter,
		Stderr: p.logger.ActionWriter,
		Env:    processPrecompileEnv(os.Environ()),
	})
	if err != nil {
		return fmt.Errorf("failed to execute bundle exec output:\n%s\nerror: %s", buffer.String(), err)
	}

	return nil
}

func processPrecompileEnv(env []string) []string {
	hasRailsEnv := false
	hasSecretKeyBase := false
	for _, pair := range env {
		if strings.HasPrefix(pair, "RAILS_ENV=") {
			hasRailsEnv = true
		}

		if strings.HasPrefix(pair, "SECRET_KEY_BASE=") {
			hasSecretKeyBase = true
		}
	}

	if !hasRailsEnv {
		env = append(env, "RAILS_ENV=production")
	}

	if !hasSecretKeyBase {
		env = append(env, "SECRET_KEY_BASE=dummy")
	}

	return env
}
