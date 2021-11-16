package railsassets_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
	railsassets "github.com/paketo-buildpacks/rails-assets"
	"github.com/paketo-buildpacks/rails-assets/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string
		buffer     *bytes.Buffer
		timeStamp  time.Time

		clock chronos.Clock

		buildProcess     *fakes.BuildProcess
		calculator       *fakes.Calculator
		environmentSetup *fakes.EnvironmentSetup

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(workingDir, "app", "assets"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		buildProcess = &fakes.BuildProcess{}

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewLogger(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		calculator = &fakes.Calculator{}
		calculator.SumCall.Returns.String = "some-calculator-sha"

		environmentSetup = &fakes.EnvironmentSetup{}

		build = railsassets.Build(buildProcess, calculator, environmentSetup, logger, clock)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that precompiles assets", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			Layers:     packit.Layers{Path: layersDir},
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				{
					Path:      filepath.Join(layersDir, "assets"),
					Name:      "assets",
					Launch:    true,
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{
						"RAILS_ENV.default":                "production",
						"RAILS_SERVE_STATIC_FILES.default": "true",
					},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Metadata: map[string]interface{}{
						"built_at":  timeStamp.Format(time.RFC3339Nano),
						"cache_sha": "some-calculator-sha",
					},
				},
			},
		}))

		Expect(buildProcess.ExecuteCall.CallCount).To(Equal(1))
		Expect(buildProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Configuring launch environment"))
		Expect(buffer.String()).To(ContainSubstring(`RAILS_ENV                -> "production"`))
		Expect(buffer.String()).To(ContainSubstring(`RAILS_SERVE_STATIC_FILES -> "true"`))
	})

	context("when checksum matches", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(layersDir, fmt.Sprintf("%s.toml", railsassets.LayerNameAssets)), []byte(fmt.Sprintf(`
[metadata]
	cache_sha = "some-calculator-sha"
	built_at = "%s"
			`, timeStamp.Format(time.RFC3339Nano))), 0600)
			Expect(err).NotTo(HaveOccurred())

			Expect(os.MkdirAll(filepath.Join(layersDir, "assets", "env.launch"), os.ModePerm)).To(Succeed())

			err = os.WriteFile(filepath.Join(layersDir, "assets", "env.launch", "RAILS_ENV.default"), []byte("production"), 0600)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(layersDir, "assets", "env.launch", "RAILS_SERVE_STATIC_FILES.default"), []byte("true"), 0600)
			Expect(err).NotTo(HaveOccurred())

			calculator.SumCall.Returns.String = "some-calculator-sha"
		})

		it("reuses the cached layer", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Path:      filepath.Join(layersDir, "assets"),
						Name:      "assets",
						Launch:    true,
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{
							"RAILS_ENV.default":                "production",
							"RAILS_SERVE_STATIC_FILES.default": "true",
						},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Metadata: map[string]interface{}{
							"built_at":  timeStamp.Format(time.RFC3339Nano),
							"cache_sha": "some-calculator-sha",
						},
					},
				},
			}))

			Expect(calculator.SumCall.Receives.Paths).To(Equal([]string{
				filepath.Join(workingDir, "app", "assets"),
			}))

			Expect(buildProcess.ExecuteCall.CallCount).To(Equal(0))

			Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
		})

		context("when there is a lib/assets directory", func() {
			it.Before(func() {
				Expect(os.RemoveAll(filepath.Join(workingDir, "app", "assets"))).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(workingDir, "lib", "assets"), os.ModePerm)).To(Succeed())
			})

			it("uses that directory to calculate the checksum", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(calculator.SumCall.Receives.Paths).To(Equal([]string{
					filepath.Join(workingDir, "lib", "assets"),
				}))
			})
		})

		context("when there is a vendor/assets directory", func() {
			it.Before(func() {
				Expect(os.RemoveAll(filepath.Join(workingDir, "app", "assets"))).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(workingDir, "vendor", "assets"), os.ModePerm)).To(Succeed())
			})

			it("uses that directory to calculate the checksum", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(calculator.SumCall.Receives.Paths).To(Equal([]string{
					filepath.Join(workingDir, "vendor", "assets"),
				}))
			})
		})

		context("when there is a app/javascript directory", func() {
			it.Before(func() {
				Expect(os.RemoveAll(filepath.Join(workingDir, "app", "assets"))).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(workingDir, "app", "javascript"), os.ModePerm)).To(Succeed())
			})

			it("uses that directory to calculate the checksum", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(calculator.SumCall.Receives.Paths).To(Equal([]string{
					filepath.Join(workingDir, "app", "javascript"),
				}))
			})
		})

		context("failure cases", func() {
			context("when environment linking fails", func() {
				it.Before(func() {
					environmentSetup.LinkCall.Returns.Error = errors.New("some-error")
				})

				it("returns the error", func() {
					_, err := build(packit.BuildContext{})
					Expect(err).To(MatchError("some-error"))
				})
			})
		})
	})

	context("failure cases", func() {
		context("when environment setup fails", func() {
			it.Before(func() {
				environmentSetup.ResetLocalCall.Returns.Error = errors.New("some-error")
			})

			it("returns the error", func() {
				_, err := build(packit.BuildContext{})
				Expect(err).To(MatchError("some-error"))
			})
		})

		context("when calculator sum fails", func() {
			it.Before(func() {
				calculator.SumCall.Returns.Error = errors.New("some-error")
			})

			it("returns the error", func() {
				_, err := build(packit.BuildContext{})
				Expect(err).To(MatchError("some-error"))
			})
		})

		context("when reset layer fails", func() {
			it.Before(func() {
				environmentSetup.ResetLayerCall.Returns.Error = errors.New("some-error")
			})

			it("returns the error", func() {
				_, err := build(packit.BuildContext{})
				Expect(err).To(MatchError("some-error"))
			})
		})

		context("when precompile process fails", func() {
			it.Before(func() {
				buildProcess.ExecuteCall.Returns.Error = errors.New("some-error")
			})

			it("returns the error", func() {
				_, err := build(packit.BuildContext{})
				Expect(err).To(MatchError("some-error"))
			})
		})
	})
}
