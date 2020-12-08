package railsassets_test

import (
	"bytes"
	"io/ioutil"
	"os"
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

		build = railsassets.Build(installProcess, logEmitter, clock)
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

	context("failure cases", func() {
		// TODO
	})
}
