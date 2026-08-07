package railsassets_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	railsassets "github.com/paketo-buildpacks/rails-assets"
	"github.com/paketo-buildpacks/rails-assets/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir    string
		gemfileParser *fakes.Parser
		detect        packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(filepath.Join(workingDir, "Gemfile"), []byte{}, 0600)
		Expect(err).NotTo(HaveOccurred())

		gemfileParser = &fakes.Parser{}

		detect = railsassets.Detect(gemfileParser, railsassets.NewNodeLockfileChecker())
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when the Gemfile lists rails", func() {
		it.Before(func() {
			gemfileParser.ParseCall.Returns.HasRails = true
		})

		context("when the app/assets directory is present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "app", "assets"), os.ModePerm)).To(Succeed())
			})

			it("detects", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{},
					Requires: []packit.BuildPlanRequirement{
						{Name: "mri", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "bundler", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "gems", Metadata: railsassets.BuildPlanMetadata{Build: true}},
					},
				}))
			})

			context("when the working directory contains a yarn.lock file", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "yarn.lock"), nil, 0600)).To(Succeed())
				})

				it("detects with node_modules", func() {
					result, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.Plan).To(Equal(packit.BuildPlan{
						Provides: []packit.BuildPlanProvision{},
						Requires: []packit.BuildPlanRequirement{
							{Name: "mri", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "bundler", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "gems", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "node", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "node_modules", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						},
					}))
				})
			})

			context("when the working directory contains a package-lock.json file", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "package-lock.json"), nil, 0600)).To(Succeed())
				})

				it("detects with node_modules", func() {
					result, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.Plan).To(Equal(packit.BuildPlan{
						Provides: []packit.BuildPlanProvision{},
						Requires: []packit.BuildPlanRequirement{
							{Name: "mri", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "bundler", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "gems", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "node", Metadata: railsassets.BuildPlanMetadata{Build: true}},
							{Name: "node_modules", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						},
					}))
				})
			})
		})

		context("when the lib/assets directory is present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "lib", "assets"), os.ModePerm)).To(Succeed())
			})

			it("detects", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{},
					Requires: []packit.BuildPlanRequirement{
						{Name: "mri", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "bundler", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "gems", Metadata: railsassets.BuildPlanMetadata{Build: true}},
					},
				}))
			})
		})

		context("when the vendor/assets directory is present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "vendor", "assets"), os.ModePerm)).To(Succeed())
			})

			it("detects", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{},
					Requires: []packit.BuildPlanRequirement{
						{Name: "mri", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "bundler", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "gems", Metadata: railsassets.BuildPlanMetadata{Build: true}},
					},
				}))
			})
		})

		context("when the app/javascript directory is present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "app", "javascript"), os.ModePerm)).To(Succeed())
			})

			it("detects", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{},
					Requires: []packit.BuildPlanRequirement{
						{Name: "mri", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "bundler", Metadata: railsassets.BuildPlanMetadata{Build: true}},
						{Name: "gems", Metadata: railsassets.BuildPlanMetadata{Build: true}},
					},
				}))
			})
		})

		context("when there are no asset directories", func() {
			it("fails with an error message", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(packit.Fail.WithMessage("failed to find assets in app/assets, app/javascript, lib/assets, or vendor/assets")))
			})
		})
	})

	context("when the Gemfile does not list rails", func() {
		it.Before(func() {
			gemfileParser.ParseCall.Returns.HasRails = false

			Expect(os.MkdirAll(filepath.Join(workingDir, "app", "javascript"), os.ModePerm)).To(Succeed())
		})

		it("fails with an error message", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail.WithMessage("failed to find rails gem in Gemfile")))
		})
	})

	context("failure cases", func() {
		context("when the gemfile parser fails", func() {
			it.Before(func() {
				gemfileParser.ParseCall.Returns.Err = errors.New("some-error")

				Expect(os.MkdirAll(filepath.Join(workingDir, "lib", "assets"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError("failed to parse Gemfile: some-error"))
			})
		})
	})
}
