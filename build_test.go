package railsassets_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
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

		installProcess *fakes.InstallProcess
		calculator     *fakes.Calculator
		entryResolver  *fakes.EntryResolver

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		installProcess = &fakes.InstallProcess{}

		buffer = bytes.NewBuffer(nil)
		logEmitter := railsassets.NewLogEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		calculator = &fakes.Calculator{}
		calculator.SumCall.Returns.String = "some-calculator-sha"

		entryResolver = &fakes.EntryResolver{}

		build = railsassets.Build(installProcess, calculator, logEmitter, clock, entryResolver)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that precompiles assets", func() {
		_, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))
		Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
	})

	context("when checksum matches", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(layersDir, fmt.Sprintf("%s.toml", railsassets.LayerNameRails)), []byte(fmt.Sprintf(`[metadata]
			cache_sha = "some-calculator-sha"
			built_at = "%s"
			`, timeStamp.Format(time.RFC3339Nano))), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("reuses the cached layer", func() {
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

			Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))

			Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
			Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		})
	})

	context("failure cases", func() {
		// TODO
	})
}
