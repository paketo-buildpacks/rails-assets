package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var settings struct {
	Buildpacks struct {
		RailsAssets struct {
			Online string
		}
		MRI struct {
			Online string
		}
		Bundler struct {
			Online string
		}
		BundleInstall struct {
			Online string
		}
		Yarn struct {
			Online string
		}
		Puma struct {
			Online string
		}
		NodeEngine struct {
			Online string
		}
	}

	Buildpack struct {
		ID   string
		Name string
	}

	Config struct {
		MRI           string `json:"mri"`
		Bundler       string `json:"bundler"`
		BundleInstall string `json:"bundle-install"`
		Puma          string `json:"puma"`
		NodeEngine    string `json:"node-engine"`
		Yarn          string `json:"yarn"`
	}
}

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.DecodeReader(file, &settings)
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())

	buildpackStore := occam.NewBuildpackStore()

	settings.Buildpacks.RailsAssets.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.MRI.Online, err = buildpackStore.Get.
		Execute(settings.Config.MRI)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.Bundler.Online, err = buildpackStore.Get.
		Execute(settings.Config.Bundler)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.BundleInstall.Online, err = buildpackStore.Get.
		Execute(settings.Config.BundleInstall)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.Puma.Online, err = buildpackStore.Get.
		Execute(settings.Config.Puma)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.NodeEngine.Online, err = buildpackStore.Get.
		Execute(settings.Config.NodeEngine)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.Yarn.Online, err = buildpackStore.Get.
		Execute(settings.Config.Yarn)
	Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Parallel(), spec.Report(report.Terminal{}))
	suite("DefaultApp", testDefaultApp)
	suite("Caching", testCaching)
	suite.Run(t)
}
