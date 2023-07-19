package railsassets

import (
	"os"
	"path/filepath"
	"strings"
)

// DirectorySetup performs the operations necessary to setup a valid working
// directory and link it to the layers created by the buildpack.
type DirectorySetup struct{}

// NewDirectorySetup initializes a DirectorySetup instance.
func NewDirectorySetup() DirectorySetup {
	return DirectorySetup{}
}

// ResetLocal deletes public/assets, public/packs, tmp/cache/assets,
// and all custom assets directories. These directories will be replaced
// by links to directories internal to the "assets" layer that is created
// by this buildpack.
//
// Additionally, ResetLocal ensures that the working directory at least
// contains a public and tmp/cache directory so that these links have a
// location to be placed into.
func (DirectorySetup) ResetLocal(workingDir string) error {
	paths := []string{
		filepath.Join("public", "assets"),
		filepath.Join("public", "packs"),
		filepath.Join("tmp", "cache", "assets"),
	}
	paths = append(paths, customAssetsPrecompilePaths()...)

	for _, path := range paths {
		err := os.RemoveAll(filepath.Join(workingDir, path))
		if err != nil {
			return err
		}
	}

	for _, path := range paths {
		err := os.MkdirAll(filepath.Join(workingDir, filepath.Dir(path)), os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// ResetLayer ensures that the "assets" layer contains public-assets,
// public-packs, and tmp-cache-assets and the custom assets directories defined by the user.
// These directories will hold the results of running the "rails assets:precompile" build process.
func (DirectorySetup) ResetLayer(layerPath string) error {
	err := os.MkdirAll(filepath.Join(layerPath, "public-assets"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(layerPath, "public-packs"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(layerPath, "tmp-cache-assets"), os.ModePerm)
	if err != nil {
		return err
	}

	for _, path := range customAssetsPrecompilePaths() {
		err := os.MkdirAll(filepath.Join(layerPath, slugifyPath(path)), os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// Link creates symlinks between the working directory and the directories in
// the "assets" layer that contain the results of the "rails assets:precompile"
// build process. This makes those contents appear as if they are part of the
// application source code while still being located in a layer that can be
// cached and reused on subsequent builds.
func (DirectorySetup) Link(layerPath, workingDir string) error {
	err := os.Symlink(filepath.Join(layerPath, "public-assets"), filepath.Join(workingDir, "public", "assets"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, "public-packs"), filepath.Join(workingDir, "public", "packs"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, "tmp-cache-assets"), filepath.Join(workingDir, "tmp", "cache", "assets"))
	if err != nil {
		return err
	}

	for _, path := range customAssetsPrecompilePaths() {
		err := os.Symlink(filepath.Join(layerPath, slugifyPath(path)), filepath.Join(workingDir, path))
		if err != nil {
			return err
		}
	}

	return nil
}

func customAssetsPrecompilePaths() []string {
	assetsPaths := []string{}
	for _, customPath := range filepath.SplitList(os.Getenv("BP_RAILS_ASSETS_EXTRA_DESTINATION_PATHS")) {
		assetsPaths = append(assetsPaths, filepath.Clean(customPath))
	}
	return assetsPaths
}

func slugifyPath(path string) string {
	return strings.ReplaceAll(path, string(filepath.Separator), "-")
}
