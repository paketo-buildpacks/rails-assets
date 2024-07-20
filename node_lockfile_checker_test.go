package railsassets_test

import (
	"os"
	"path/filepath"
	"testing"

	railsassets "github.com/paketo-buildpacks/rails-assets"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testNodeLockfileChecker(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		dir     string
		checker railsassets.NodeLockfileChecker
	)

	it.Before(func() {
		tmpDir, err := os.MkdirTemp("", "")
		dir = tmpDir
		Expect(err).NotTo(HaveOccurred())
		checker = railsassets.NewNodeLockfileChecker()
	})

	it.After(func() {
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	context("Check", func() {
		context("when lock file exists", func() {
			var path string
			var setupTest = func(lockFile string) func() {
				path = filepath.Join(dir, lockFile)
				err := os.WriteFile(path, []byte(""), 0600)
				Expect(err).NotTo(HaveOccurred())

				return func() {
					Expect(os.RemoveAll(path)).To(Succeed())
				}
			}

			context("for yarn", func() {
				it("returns true", func() {
					defer setupTest("yarn.lock")()
					hasLock, err := checker.Check(dir)
					Expect(err).NotTo(HaveOccurred())
					Expect(hasLock).To(BeTrue())
				})
			})

			context("for npm", func() {
				it("returns true", func() {
					defer setupTest("package-lock.json")()
					hasLock, err := checker.Check(dir)
					Expect(err).NotTo(HaveOccurred())
					Expect(hasLock).To(BeTrue())
				})
			})
		})

		context("when no lock file", func() {
			it("returns false", func() {
				hasLock, err := checker.Check(dir)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasLock).To(BeFalse())
			})
		})
	})
}
