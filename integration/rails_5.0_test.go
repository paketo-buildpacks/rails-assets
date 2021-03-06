package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testRails50(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}

		name       string
		source     string
		buildpacks = []string{
			settings.Buildpacks.MRI.Online,
			settings.Buildpacks.Bundler.Online,
			settings.Buildpacks.BundleInstall.Online,
			settings.Buildpacks.NodeEngine.Online,
			settings.Buildpacks.RailsAssets.Online,
			settings.Buildpacks.Puma.Online,
		}
	)

	it.Before(func() {
		imageIDs = make(map[string]struct{})
		containerIDs = make(map[string]struct{})

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		source, err = occam.Source(filepath.Join("testdata", "5.0"))
		Expect(err).NotTo(HaveOccurred())

	})

	it.After(func() {
		for id := range containerIDs {
			Expect(settings.Docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id := range imageIDs {
			Expect(settings.Docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(settings.Docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("creates a working OCI image with compiled rails assets", func() {
		image, logs, err := settings.Pack.WithNoColor().Build.
			WithBuildpacks(buildpacks...).
			WithPullPolicy("never").
			Execute(name, source)
		Expect(err).NotTo(HaveOccurred(), logs.String())

		imageIDs[image.ID] = struct{}{}

		container, err := settings.Docker.Container.Run.
			WithEnv(map[string]string{
				"PORT":            "8080",
				"SECRET_KEY_BASE": "some-secret",
			}).
			WithPublish("8080").
			WithPublishAll().
			Execute(image.ID)
		Expect(err).NotTo(HaveOccurred())

		containerIDs[container.ID] = struct{}{}

		Eventually(container).Should(BeAvailable())

		response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusOK))

		document, err := goquery.NewDocumentFromReader(response.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.Body.Close()).To(Succeed())

		var path string
		document.Find("script").Each(func(i int, selection *goquery.Selection) {
			path, _ = selection.Attr("src")
		})

		Eventually(container).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

		Expect(logs).To(ContainLines(
			MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
			"  Executing build process",
			"    Running 'bundle exec rails assets:precompile assets:clean'",
			MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			"",
			"  Configuring launch environment",
			`    RAILS_ENV                -> "production"`,
			`    RAILS_SERVE_STATIC_FILES -> "true"`,
		))
	})

	context("when executing a rebuild", func() {
		it("reuses the assets layer", func() {
			build := settings.Pack.WithNoColor().Build.
				WithBuildpacks(buildpacks...).
				WithPullPolicy("never")

			firstImage, firstLogs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), firstLogs.String())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(6))
			Expect(firstImage.Buildpacks[4].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[4].Layers).To(HaveKey("assets"))

			container, err := settings.Docker.Container.Run.
				WithCommand(fmt.Sprintf("ls -alR /layers/%s/assets/public/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))).
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			secondImage, secondLogs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), secondLogs.String)

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(6))
			Expect(secondImage.Buildpacks[4].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[4].Layers).To(HaveKey("assets"))

			container, err = settings.Docker.Container.Run.
				WithCommand(fmt.Sprintf("ls -alR /layers/%s/assets/public/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))).
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[container.ID] = struct{}{}

			Expect(secondImage.Buildpacks[4].Layers["assets"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[4].Layers["assets"].Metadata["built_at"]))
			Expect(secondImage.Buildpacks[4].Layers["assets"].Metadata["cache_sha"]).To(Equal(firstImage.Buildpacks[4].Layers["assets"].Metadata["cache_sha"]))

			// TODO: why is the image id changing?
			// Expect(secondImage.ID).To(Equal(firstImage.ID), fmt.Sprintf("%s\n\n%s", firstLogs, secondLogs))

			Expect(secondLogs).To(ContainLines(
				fmt.Sprintf("%s %s", settings.Buildpack.Name, "1.2.3"),
				fmt.Sprintf("  Reusing cached layer /layers/%s/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})
	})
}
