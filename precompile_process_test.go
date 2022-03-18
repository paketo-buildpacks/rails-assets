package railsassets_test

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	railsassets "github.com/paketo-buildpacks/rails-assets"
	"github.com/paketo-buildpacks/rails-assets/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPrecompileProcess(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("Execute", func() {
		var (
			workingDir string
			path       string
			executions []pexec.Execution
			executable *fakes.Executable

			precompileProcess railsassets.PrecompileProcess
		)

		it.Before(func() {
			var err error
			workingDir, err = os.MkdirTemp("", "working-dir")
			Expect(err).NotTo(HaveOccurred())

			executions = []pexec.Execution{}
			executable = &fakes.Executable{}
			executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
				executions = append(executions, execution)

				return nil
			}

			path = os.Getenv("PATH")
			os.Setenv("PATH", "/some/bin")

			logger := scribe.NewEmitter(bytes.NewBuffer(nil))

			precompileProcess = railsassets.NewPrecompileProcess(executable, logger)
		})

		it.After(func() {
			os.Setenv("PATH", path)

			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("runs the bundle exec assets:precompile process", func() {
			err := precompileProcess.Execute(workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(executions).To(HaveLen(1))
			Expect(executions[0].Args).To(Equal([]string{"exec", "rails", "assets:precompile", "assets:clean"}))
		})

		context("failure cases", func() {
			context("when bundle exec fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						return errors.New("bundle exec failed")
					}
				})
				it("prints the execution output and returns an error", func() {
					err := precompileProcess.Execute(workingDir)
					Expect(err).To(MatchError(ContainSubstring("failed to execute bundle exec")))
					Expect(err).To(MatchError(ContainSubstring("bundle exec failed")))
				})
			})
		})
	})
}
