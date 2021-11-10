package railsassets_test

import (
	"os"
	"testing"

	railsassets "github.com/paketo-buildpacks/rails-assets"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGemfileParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path   string
		parser railsassets.GemfileParser
	)

	it.Before(func() {
		file, err := os.CreateTemp("", "Gemfile")
		Expect(err).NotTo(HaveOccurred())
		defer file.Close()

		path = file.Name()

		parser = railsassets.NewGemfileParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("Parse", func() {
		context("when using rails", func() {
			it("parses correctly", func() {
				Expect(os.WriteFile(path, []byte(`source 'https://rubygems.org' do
	gem 'rails'
end
`), 0600)).To(Succeed())

				hasRails, err := parser.Parse(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasRails).To(BeTrue())
			})
		})

		context("when not using rails", func() {
			it("parses correctly", func() {
				Expect(os.WriteFile(path, []byte(`source 'https://rubygems.org'`), 0600)).To(Succeed())

				hasRails, err := parser.Parse(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasRails).To(BeFalse())
			})
		})

		context("when the Gemfile file does not exist", func() {
			it.Before(func() {
				Expect(os.Remove(path)).To(Succeed())
			})

			it("returns all false", func() {
				hasRails, err := parser.Parse(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasRails).To(BeFalse())
			})
		})

		context("failure cases", func() {
			context("when the Gemfile cannot be opened", func() {
				it.Before(func() {
					Expect(os.Chmod(path, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.Parse(path)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("failed to parse Gemfile:")))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
