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
			executions []pexec.Execution
			executable *fakes.Executable

			precompileProcess railsassets.PrecompileProcess

			hasRailsEnv      bool
			hasSecretKeyBase bool
			railsEnv         string
			secretKeyBase    string
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

			logger := scribe.NewEmitter(bytes.NewBuffer(nil))

			precompileProcess = railsassets.NewPrecompileProcess(executable, logger)

			railsEnv, hasRailsEnv = os.LookupEnv("RAILS_ENV")
			secretKeyBase, hasSecretKeyBase = os.LookupEnv("SECRET_KEY_BASE")
		})

		it.After(func() {
			if hasRailsEnv {
				os.Setenv("RAILS_ENV", railsEnv)
			}

			if hasSecretKeyBase {
				os.Setenv("SECRET_KEY_BASE", secretKeyBase)
			}

			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("runs the bundle exec assets:precompile process", func() {
			err := precompileProcess.Execute(workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(executions).To(HaveLen(1))
			Expect(executions[0].Args).To(Equal([]string{"exec", "rails", "assets:precompile", "assets:clean"}))
			Expect(executions[0].Env).To(ContainElement("RAILS_ENV=production"))
			Expect(executions[0].Env).To(ContainElement("SECRET_KEY_BASE=dummy"))
		})

           context("when a user sets their own RAILS_ENV", func() {
                it.Before(func() {
                  Expect(os.Setenv("RAILS_ENV", "staging")).To(Succeed())
                })
                it.After(func() {
                    Expect(os.Unsetenv("RAILS_ENV")).To(Succeed())
                })
		it("runs the bundle exec assets:precompile process while respecting RAILS_ENV", func() {
			err := precompileProcess.Execute(workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(executions).To(HaveLen(1))
			Expect(executions[0].Args).To(Equal([]string{"exec", "rails", "assets:precompile", "assets:clean"}))
			Expect(executions[0].Env).To(ContainElement("RAILS_ENV=staging"))
			Expect(executions[0].Env).To(ContainElement("SECRET_KEY_BASE=dummy"))
		})
	})

		it("runs the bundle exec assets:precompile process while respecting SECRET_KEY_BASE", func() {
			os.Setenv("SECRET_KEY_BASE", "dummy2")
			err := precompileProcess.Execute(workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(executions).To(HaveLen(1))
			Expect(executions[0].Args).To(Equal([]string{"exec", "rails", "assets:precompile", "assets:clean"}))
			Expect(executions[0].Env).To(ContainElement("RAILS_ENV=production"))
			Expect(executions[0].Env).To(ContainElement("SECRET_KEY_BASE=dummy2"))
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
