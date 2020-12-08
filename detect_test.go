package railsassets_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"io/ioutil"

	"github.com/paketo-buildpacks/packit"
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
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(workingDir, "app", "assets"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(workingDir, "Gemfile"), []byte{}, 0644)
		Expect(err).NotTo(HaveOccurred())

		gemfileParser = &fakes.Parser{}

		detect = railsassets.Detect(gemfileParser)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when the Gemfile lists rails and the app/assets directory exists", func() {
		it.Before(func() {
			gemfileParser.ParseCall.Returns.HasRails = true
		})
		it("detects", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "gems",
						Metadata: railsassets.BuildPlanMetadata{
							Launch: true,
						},
					},
					{
						Name: "bundler",
						Metadata: railsassets.BuildPlanMetadata{
							Launch: true,
						},
					},
					{
						Name: "mri",
						Metadata: railsassets.BuildPlanMetadata{
							Launch: true,
						},
					},
				},
			}))
		})
	})

	context("when the Gemfile does not list rails", func() {
		it.Before(func() {
			gemfileParser.ParseCall.Returns.HasRails = false
		})

		it("detect should fail with error", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail))
		})
	})

	context("when the app/assets directory does not exist", func() {
		it.Before(func() {
			gemfileParser.ParseCall.Returns.HasRails = true
			err := os.RemoveAll(filepath.Join(workingDir, "app/assets"))
			Expect(err).NotTo(HaveOccurred())
		})

		it("detect should fail with error", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail))
		})
	})

	context("failure cases", func() {
		context("when the gemfile parser fails", func() {
			it.Before(func() {
				gemfileParser.ParseCall.Returns.Err = errors.New("some-error")
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
