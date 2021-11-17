package railsassets_test

import (
	"os"
	"path/filepath"
	"testing"

	railsassets "github.com/paketo-buildpacks/rails-assets"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDirectorySetup(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		setup      railsassets.EnvironmentSetup
		layerPath  string
		workingDir string
	)

	it.Before(func() {
		var err error
		layerPath, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		setup = railsassets.NewDirectorySetup()
	})

	it.After(func() {
		Expect(os.RemoveAll(layerPath)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("ResetLocal", func() {
		it.Before(func() {
			Expect(os.MkdirAll(filepath.Join(workingDir, "public", "assets"), os.ModePerm)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(workingDir, "public", "packs"), os.ModePerm)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(workingDir, "tmp", "cache", "assets"), os.ModePerm)).To(Succeed())
		})

		it("recreates directories", func() {
			err := setup.ResetLocal(workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "public")).To(BeADirectory())
			Expect(filepath.Join(workingDir, "tmp", "cache")).To(BeADirectory())

			Expect(filepath.Join(workingDir, "public", "assets")).NotTo(BeADirectory())
			Expect(filepath.Join(workingDir, "public", "packs")).NotTo(BeADirectory())
			Expect(filepath.Join(workingDir, "tmp", "cache", "assets")).NotTo(BeADirectory())
		})
	})

	context("ResetLayer", func() {
		it("creates the directories", func() {
			Expect(setup.ResetLayer(layerPath)).To(Succeed())

			Expect(filepath.Join(layerPath, "tmp-cache-assets")).To(BeADirectory())
			Expect(filepath.Join(layerPath, "public-assets")).To(BeADirectory())
			Expect(filepath.Join(layerPath, "public-packs")).To(BeADirectory())
		})
	})

	context("Link", func() {
		it.Before(func() {
			err := os.MkdirAll(filepath.Join(workingDir, "public"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			err = os.MkdirAll(filepath.Join(workingDir, "tmp", "cache"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
		})

		it("links the layer and working directory", func() {
			err := setup.Link(layerPath, workingDir)
			Expect(err).NotTo(HaveOccurred())

			link, err := os.Readlink(filepath.Join(workingDir, "tmp", "cache", "assets"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "tmp-cache-assets")))

			link, err = os.Readlink(filepath.Join(workingDir, "public", "assets"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "public-assets")))

			link, err = os.Readlink(filepath.Join(workingDir, "public", "packs"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "public-packs")))
		})
	})
}
