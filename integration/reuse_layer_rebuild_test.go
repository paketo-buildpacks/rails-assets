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

func testReusingLayerRebuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		docker occam.Docker
		pack   occam.Pack

		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}

		name   string
		source string
	)

	it.Before(func() {
		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		docker = occam.NewDocker()
		pack = occam.NewPack().WithNoColor()
		imageIDs = map[string]struct{}{}
		containerIDs = map[string]struct{}{}

		source, err = occam.Source(filepath.Join("testdata", "6.1"))
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

	context("when an app is rebuilt and does not change", func() {
		it("reuses a layer from a previous build", func() {
			var (
				logs        fmt.Stringer
				firstImage  occam.Image
				secondImage occam.Image

				firstContainer  occam.Container
				secondContainer occam.Container
			)

			build := pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.MRI.Online,
					settings.Buildpacks.Bundler.Online,
					settings.Buildpacks.BundleInstall.Online,
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.Yarn.Online,
					settings.Buildpacks.YarnInstall.Online,
					settings.Buildpacks.RailsAssets.Online,
					settings.Buildpacks.Puma.Online,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL": "DEBUG",
				})

			var err error
			firstImage, logs, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(8))
			Expect(firstImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[6].Layers).To(HaveKey("assets"))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
			))

			Expect(logs).To(ContainLines(
				"  Executing build process",
				"    Running 'bundle exec rails assets:precompile assets:clean'",
			))

			Expect(logs).To(ContainLines(
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			))

			Expect(logs).To(ContainLines(
				"  Configuring launch environment",
				`    RAILS_ENV                -> "production"`,
				`    RAILS_LOG_TO_STDOUT      -> "true"`,
				`    RAILS_SERVE_STATIC_FILES -> "true"`,
			))

			firstContainer, err = docker.Container.Run.
				WithEnv(map[string]string{
					"PORT":            "8080",
					"SECRET_KEY_BASE": "some-secret",
				}).
				WithPublish("8080").
				WithPublishAll().
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[firstContainer.ID] = struct{}{}

			Eventually(firstContainer).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", firstContainer.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			document, err := goquery.NewDocumentFromReader(response.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Body.Close()).To(Succeed())

			var path string
			document.Find("script").Each(func(_ int, selection *goquery.Selection) {
				path, _ = selection.Attr("src")
			})

			Eventually(firstContainer).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

			// Second pack build
			secondImage, logs, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(8))
			Expect(secondImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[6].Layers).To(HaveKey("assets"))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
			))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("  Reusing cached layer /layers/%s/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))

			secondContainer, err = docker.Container.Run.
				WithEnv(map[string]string{
					"PORT":            "8080",
					"SECRET_KEY_BASE": "some-secret",
				}).
				WithPublish("8080").
				WithPublishAll().
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(secondContainer).Should(BeAvailable())

			response, err = http.Get(fmt.Sprintf("http://localhost:%s", secondContainer.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			document, err = goquery.NewDocumentFromReader(response.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Body.Close()).To(Succeed())

			document.Find("script").Each(func(_ int, selection *goquery.Selection) {
				path, _ = selection.Attr("src")
			})

			Eventually(secondContainer).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

			containerIDs[secondContainer.ID] = struct{}{}

			Expect(secondImage.Buildpacks[6].Layers["assets"].SHA).To(Equal(firstImage.Buildpacks[6].Layers["assets"].SHA))
		})

		context("when the app has assets in extra destination paths", func() {
			it.Before(func() {
				file, err := os.OpenFile(filepath.Join(source, "config", "initializers", "assets.rb"), os.O_APPEND|os.O_RDWR, 0600)
				Expect(err).NotTo(HaveOccurred())

				_, err = file.WriteString("Rails.application.config.assets.prefix = \"custom\"")
				Expect(err).NotTo(HaveOccurred())
			})

			it("reuses extra assets from previous builds", func() {
				var (
					logs        fmt.Stringer
					firstImage  occam.Image
					secondImage occam.Image

					firstContainer  occam.Container
					secondContainer occam.Container
				)

				build := pack.Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.MRI.Online,
						settings.Buildpacks.Bundler.Online,
						settings.Buildpacks.BundleInstall.Online,
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.Yarn.Online,
						settings.Buildpacks.YarnInstall.Online,
						settings.Buildpacks.RailsAssets.Online,
						settings.Buildpacks.Puma.Online,
					).
					WithEnv(map[string]string{
						"BP_LOG_LEVEL": "DEBUG",
						"BP_RAILS_ASSETS_EXTRA_DESTINATION_PATHS": "public/custom",
					})

				var err error
				firstImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String)

				imageIDs[firstImage.ID] = struct{}{}

				Expect(firstImage.Buildpacks).To(HaveLen(8))
				Expect(firstImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
				Expect(firstImage.Buildpacks[6].Layers).To(HaveKey("assets"))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				))

				Expect(logs).To(ContainLines(
					"  Executing build process",
					"    Running 'bundle exec rails assets:precompile assets:clean'",
				))

				Expect(logs).To(ContainLines(
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				))

				Expect(logs).To(ContainLines(
					"  Configuring launch environment",
					`    RAILS_ENV                -> "production"`,
					`    RAILS_LOG_TO_STDOUT      -> "true"`,
					`    RAILS_SERVE_STATIC_FILES -> "true"`,
				))

				firstContainer, err = docker.Container.Run.
					WithEnv(map[string]string{
						"PORT":            "8080",
						"SECRET_KEY_BASE": "some-secret",
					}).
					WithPublish("8080").
					WithPublishAll().
					Execute(firstImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[firstContainer.ID] = struct{}{}

				Eventually(firstContainer).Should(BeAvailable())

				response, err := http.Get(fmt.Sprintf("http://localhost:%s", firstContainer.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))

				document, err := goquery.NewDocumentFromReader(response.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(response.Body.Close()).To(Succeed())

				var path string
				document.Find("script").Each(func(_ int, selection *goquery.Selection) {
					path, _ = selection.Attr("src")
				})

				Eventually(firstContainer).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

				// Second pack build
				secondImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				imageIDs[secondImage.ID] = struct{}{}

				Expect(secondImage.Buildpacks).To(HaveLen(8))
				Expect(secondImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
				Expect(secondImage.Buildpacks[6].Layers).To(HaveKey("assets"))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				))

				Expect(logs).To(ContainLines(
					fmt.Sprintf("  Reusing cached layer /layers/%s/assets", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				))

				secondContainer, err = docker.Container.Run.
					WithEnv(map[string]string{
						"PORT":            "8080",
						"SECRET_KEY_BASE": "some-secret",
					}).
					WithPublish("8080").
					WithPublishAll().
					Execute(secondImage.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(secondContainer).Should(BeAvailable())

				response, err = http.Get(fmt.Sprintf("http://localhost:%s", secondContainer.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))

				document, err = goquery.NewDocumentFromReader(response.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(response.Body.Close()).To(Succeed())

				document.Find("script").Each(func(_ int, selection *goquery.Selection) {
					path, _ = selection.Attr("src")
				})

				Eventually(secondContainer).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

				containerIDs[secondContainer.ID] = struct{}{}

				Expect(secondImage.Buildpacks[6].Layers["assets"].SHA).To(Equal(firstImage.Buildpacks[6].Layers["assets"].SHA))
			})
		})
	})

	context("when an app is rebuilt and there is a change", func() {
		it("rebuilds the layer", func() {
			var (
				logs        fmt.Stringer
				firstImage  occam.Image
				secondImage occam.Image

				firstContainer  occam.Container
				secondContainer occam.Container
			)

			build := pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.MRI.Online,
					settings.Buildpacks.Bundler.Online,
					settings.Buildpacks.BundleInstall.Online,
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.Yarn.Online,
					settings.Buildpacks.YarnInstall.Online,
					settings.Buildpacks.RailsAssets.Online,
					settings.Buildpacks.Puma.Online,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL": "DEBUG",
				})

			var err error
			firstImage, logs, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(8))
			Expect(firstImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[6].Layers).To(HaveKey("assets"))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
			))

			Expect(logs).To(ContainLines(
				"  Executing build process",
				"    Running 'bundle exec rails assets:precompile assets:clean'",
			))

			Expect(logs).To(ContainLines(
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			))

			Expect(logs).To(ContainLines(
				"  Configuring launch environment",
				`    RAILS_ENV                -> "production"`,
				`    RAILS_LOG_TO_STDOUT      -> "true"`,
				`    RAILS_SERVE_STATIC_FILES -> "true"`,
			))

			firstContainer, err = docker.Container.Run.
				WithEnv(map[string]string{
					"PORT":            "8080",
					"SECRET_KEY_BASE": "some-secret",
				}).
				WithPublish("8080").
				WithPublishAll().
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[firstContainer.ID] = struct{}{}

			Eventually(firstContainer).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", firstContainer.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			file, err := os.OpenFile(filepath.Join(source, "app", "javascript", "application.js"), os.O_APPEND|os.O_RDWR, 0600)
			Expect(err).NotTo(HaveOccurred())

			_, err = file.WriteString("// HERE IS A COMMENT")
			Expect(err).NotTo(HaveOccurred())

			Expect(file.Close()).To(Succeed())

			// Second pack build
			secondImage, logs, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(8))
			Expect(secondImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[6].Layers).To(HaveKey("assets"))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
			))

			Expect(logs).To(ContainLines(
				"  Executing build process",
				"    Running 'bundle exec rails assets:precompile assets:clean'",
			))

			Expect(logs).To(ContainLines(
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			))

			Expect(logs).To(ContainLines(
				"  Configuring launch environment",
				`    RAILS_ENV                -> "production"`,
				`    RAILS_LOG_TO_STDOUT      -> "true"`,
				`    RAILS_SERVE_STATIC_FILES -> "true"`,
			))

			secondContainer, err = docker.Container.Run.
				WithEnv(map[string]string{
					"PORT":            "8080",
					"SECRET_KEY_BASE": "some-secret",
				}).
				WithPublish("8080").
				WithPublishAll().
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}

			Eventually(secondContainer).Should(BeAvailable())

			response, err = http.Get(fmt.Sprintf("http://localhost:%s", secondContainer.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			document, err := goquery.NewDocumentFromReader(response.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Body.Close()).To(Succeed())

			var path string
			document.Find("script").Each(func(_ int, selection *goquery.Selection) {
				path, _ = selection.Attr("src")
			})

			Eventually(secondContainer).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

			Expect(secondImage.Buildpacks[6].Layers["assets"].SHA).NotTo(Equal(firstImage.Buildpacks[6].Layers["assets"].SHA))
		})

		context("when the change is on the the extra source assets", func() {
			it.Before(func() {
				err := os.MkdirAll(filepath.Join(source, "custom-assets"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				err = os.WriteFile(filepath.Join(source, "custom-assets", "hello.js"), []byte("console.log(\"Hello, world!\")\n"), 0755)
				Expect(err).NotTo(HaveOccurred())
			})

			it("rebuilds the layer", func() {
				var (
					logs        fmt.Stringer
					firstImage  occam.Image
					secondImage occam.Image

					firstContainer  occam.Container
					secondContainer occam.Container
				)

				build := pack.Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.MRI.Online,
						settings.Buildpacks.Bundler.Online,
						settings.Buildpacks.BundleInstall.Online,
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.Yarn.Online,
						settings.Buildpacks.YarnInstall.Online,
						settings.Buildpacks.RailsAssets.Online,
						settings.Buildpacks.Puma.Online,
					).
					WithEnv(map[string]string{
						"BP_LOG_LEVEL":                       "DEBUG",
						"BP_RAILS_ASSETS_EXTRA_SOURCE_PATHS": "custom-assets",
					})

				var err error
				firstImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String)

				imageIDs[firstImage.ID] = struct{}{}

				Expect(firstImage.Buildpacks).To(HaveLen(8))
				Expect(firstImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
				Expect(firstImage.Buildpacks[6].Layers).To(HaveKey("assets"))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				))

				Expect(logs).To(ContainLines(
					"  Executing build process",
					"    Running 'bundle exec rails assets:precompile assets:clean'",
				))

				Expect(logs).To(ContainLines(
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				))

				Expect(logs).To(ContainLines(
					"  Configuring launch environment",
					`    RAILS_ENV                -> "production"`,
					`    RAILS_LOG_TO_STDOUT      -> "true"`,
					`    RAILS_SERVE_STATIC_FILES -> "true"`,
				))

				firstContainer, err = docker.Container.Run.
					WithEnv(map[string]string{
						"PORT":            "8080",
						"SECRET_KEY_BASE": "some-secret",
					}).
					WithPublish("8080").
					WithPublishAll().
					Execute(firstImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[firstContainer.ID] = struct{}{}

				Eventually(firstContainer).Should(BeAvailable())

				response, err := http.Get(fmt.Sprintf("http://localhost:%s", firstContainer.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))

				file, err := os.OpenFile(filepath.Join(source, "custom-assets", "hello.js"), os.O_APPEND|os.O_RDWR, 0600)
				Expect(err).NotTo(HaveOccurred())

				_, err = file.WriteString("// HERE IS A COMMENT")
				Expect(err).NotTo(HaveOccurred())

				Expect(file.Close()).To(Succeed())

				// Second pack build
				secondImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				imageIDs[secondImage.ID] = struct{}{}

				Expect(secondImage.Buildpacks).To(HaveLen(8))
				Expect(secondImage.Buildpacks[6].Key).To(Equal(settings.Buildpack.ID))
				Expect(secondImage.Buildpacks[6].Layers).To(HaveKey("assets"))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				))

				Expect(logs).To(ContainLines(
					"  Executing build process",
					"    Running 'bundle exec rails assets:precompile assets:clean'",
				))

				Expect(logs).To(ContainLines(
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				))

				Expect(logs).To(ContainLines(
					"  Configuring launch environment",
					`    RAILS_ENV                -> "production"`,
					`    RAILS_LOG_TO_STDOUT      -> "true"`,
					`    RAILS_SERVE_STATIC_FILES -> "true"`,
				))

				secondContainer, err = docker.Container.Run.
					WithEnv(map[string]string{
						"PORT":            "8080",
						"SECRET_KEY_BASE": "some-secret",
					}).
					WithPublish("8080").
					WithPublishAll().
					Execute(secondImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[secondContainer.ID] = struct{}{}

				Eventually(secondContainer).Should(BeAvailable())

				response, err = http.Get(fmt.Sprintf("http://localhost:%s", secondContainer.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))

				document, err := goquery.NewDocumentFromReader(response.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(response.Body.Close()).To(Succeed())

				var path string
				document.Find("script").Each(func(_ int, selection *goquery.Selection) {
					path, _ = selection.Attr("src")
				})

				Eventually(secondContainer).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

				Expect(secondImage.Buildpacks[6].Layers["assets"].SHA).NotTo(Equal(firstImage.Buildpacks[6].Layers["assets"].SHA))
			})
		})
	})

}
