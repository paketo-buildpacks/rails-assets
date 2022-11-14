package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testRails70(t *testing.T, context spec.G, it spec.S) {
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

		source, err = occam.Source(filepath.Join("testdata", "7.0"))
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
		image, logs, err := settings.Pack.WithVerbose().Build.
			WithBuildpacks(buildpacks...).
			WithPullPolicy("never").
			WithEnv(map[string]string{
				"SECRET_KEY_BASE": "some-secret",
			}).
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
			if typ, ok := selection.Attr("type"); ok && typ == "importmap" {
				var importmap struct {
					Imports struct {
						GreetingControllerAsset string `json:"controllers/greeting_controller"`
					} `json:"imports"`
				}
				err := json.Unmarshal([]byte(selection.Text()), &importmap)
				Expect(err).NotTo(HaveOccurred())

				path = importmap.Imports.GreetingControllerAsset
			}
		})

		Eventually(container).Should(Serve(ContainSubstring("Hello from Javascript!")).OnPort(8080).WithEndpoint(path))

		Expect(logs).To(ContainLines(
			MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
			"  Executing build process",
			"    Running 'bundle exec rails assets:precompile assets:clean'",
		))
		Expect(logs).To(ContainLines(
			MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
			"",
			"  Configuring launch environment",
			`    RAILS_ENV                -> "production"`,
			`    RAILS_LOG_TO_STDOUT      -> "true"`,
			`    RAILS_SERVE_STATIC_FILES -> "true"`,
		))

		logs, err = settings.Docker.Container.Logs.Execute(container.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(logs).To(ContainLines(ContainSubstring("Processing by WelcomeController#index")))
	})
}
