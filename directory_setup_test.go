package railsassets_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	railsassets "github.com/paketo-buildpacks/rails-assets"
	"github.com/sclevine/spec"
)

func testDirectoriesSetup(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		setup      railsassets.EnvironmentSetup
		layerPath  string
		workingDir string
	)

	it.Before(func() {
		var err error
		layerPath, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		setup = railsassets.NewDirectorySetup()
	})

	it.After(func() {
		Expect(os.RemoveAll(layerPath)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("ResetLocal", func() {
		it("recreates directories", func() {
			err := setup.ResetLocal(workingDir)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(workingDir, "public"))
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(workingDir, "tmp", "cache"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	context("ResetLayer", func() {
		it("recreates directories", func() {
			err := setup.ResetLayer(layerPath)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(layerPath, "tmp-cache-assets"))
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(layerPath, "public-assets"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	context("Link", func() {
		it.Before(func() {
			err := os.MkdirAll(filepath.Join(workingDir, "public"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			err = os.MkdirAll(filepath.Join(workingDir, "tmp", "cache"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
		})

		it("recreates directories", func() {
			err := setup.Link(layerPath, workingDir)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(filepath.Join(workingDir, "tmp", "cache", "assets"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "tmp-cache-assets")))

			link, err = os.Readlink(filepath.Join(workingDir, "public", "assets"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "public-assets")))
		})
	})
}
