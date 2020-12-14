package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testCaching(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		// Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}

		name   string
		source string
	)

	it.Before(func() {
		imageIDs = make(map[string]struct{})
		containerIDs = make(map[string]struct{})

		pack = occam.NewPack()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		for id := range containerIDs {
			Expect(docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id := range imageIDs {
			Expect(docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("reuses the assets layer", func() {
		var err error
		var container occam.Container

		source, err = occam.Source(filepath.Join("testdata", "default_app"))
		Expect(err).NotTo(HaveOccurred())

		build := pack.WithNoColor().Build.
			WithBuildpacks(
				settings.Buildpacks.MRI.Online,
				settings.Buildpacks.Bundler.Online,
				settings.Buildpacks.BundleInstall.Online,
				settings.Buildpacks.NodeEngine.Online,
				settings.Buildpacks.Yarn.Online,
				settings.Buildpacks.RailsAssets.Online,
			).
			WithPullPolicy("never")

		firstImage, firstLogs, err := build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred(), firstLogs.String())

		imageIDs[firstImage.ID] = struct{}{}

		Expect(firstImage.Buildpacks).To(HaveLen(6))
		Expect(firstImage.Buildpacks[5].Key).To(Equal(settings.Buildpack.ID))
		Expect(firstImage.Buildpacks[5].Layers).To(HaveKey("assets"))

		container, err = docker.Container.Run.
			WithCommand(fmt.Sprintf("ls -alR /layers/%s/assets/public/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))).
			Execute(firstImage.ID)
		Expect(err).NotTo(HaveOccurred())

		containerIDs[container.ID] = struct{}{}

		secondImage, secondLogs, err := build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred(), secondLogs.String)

		imageIDs[secondImage.ID] = struct{}{}

		Expect(secondImage.Buildpacks).To(HaveLen(6))
		Expect(secondImage.Buildpacks[5].Key).To(Equal(settings.Buildpack.ID))
		Expect(secondImage.Buildpacks[5].Layers).To(HaveKey("assets"))

		container, err = docker.Container.Run.
			WithCommand(fmt.Sprintf("ls -alR /layers/%s/assets/public/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))).
			Execute(secondImage.ID)
		Expect(err).NotTo(HaveOccurred())

		containerIDs[container.ID] = struct{}{}

		Expect(secondImage.Buildpacks[5].Layers["assets"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[5].Layers["assets"].Metadata["built_at"]))
		Expect(secondImage.Buildpacks[5].Layers["assets"].Metadata["cache_sha"]).To(Equal(firstImage.Buildpacks[5].Layers["assets"].Metadata["cache_sha"]))

		Expect(secondImage.ID).To(Equal(firstImage.ID), fmt.Sprintf("%s\n\n%s", firstLogs, secondLogs))

		Expect(secondLogs).To(ContainLines(
			fmt.Sprintf("%s %s", settings.Buildpack.Name, "1.2.3"),
			fmt.Sprintf("  Reusing cached layer /layers/%s/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
		))
	})
}
