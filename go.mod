module github.com/paketo-buildpacks/rails-assets

go 1.16

require (
	github.com/BurntSushi/toml v1.2.1
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/onsi/gomega v1.27.6
	github.com/opencontainers/runc v1.1.4 // indirect
	github.com/paketo-buildpacks/occam v0.16.0
	github.com/paketo-buildpacks/packit/v2 v2.9.0
	github.com/sclevine/spec v1.4.0
	golang.org/x/net v0.9.0 // indirect
	gotest.tools/v3 v3.4.0 // indirect
)

replace github.com/CycloneDX/cyclonedx-go => github.com/CycloneDX/cyclonedx-go v0.6.0
