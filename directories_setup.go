package railsassets

import (
	"os"
	"path/filepath"
)

type DirectoriesSetup struct{}

func NewDirectoriesSetup() DirectoriesSetup {
	return DirectoriesSetup{}
}

const (
	publicDir = "public"
	assetsDir = "assets"
	tmpDir    = "tmp"
	cacheDir  = "cache"

	publicAssetsDir   = "public-assets"
	tmpCacheAssetsDir = "tmp-cache-assets"
)

func (DirectoriesSetup) Run(layerPath, workingDir string) error {
	err := os.RemoveAll(filepath.Join(workingDir, publicDir, assetsDir))
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(workingDir, publicDir), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.RemoveAll(filepath.Join(workingDir, tmpDir, cacheDir, assetsDir))
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(workingDir, tmpDir, cacheDir), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(layerPath, publicAssetsDir), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, publicAssetsDir), filepath.Join(workingDir, publicDir, assetsDir))
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(layerPath, tmpCacheAssetsDir), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, tmpCacheAssetsDir), filepath.Join(workingDir, tmpDir, cacheDir, assetsDir))
	if err != nil {
		return err
	}

	return nil
}
