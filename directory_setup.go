package railsassets

import (
	"os"
	"path/filepath"
)

const (
	publicDir = "public"
	assetsDir = "assets"
	tmpDir    = "tmp"
	cacheDir  = "cache"

	publicAssetsDir   = "public-assets"
	tmpCacheAssetsDir = "tmp-cache-assets"
)

type DirectorySetup struct{}

func NewDirectorySetup() DirectorySetup {
	return DirectorySetup{}
}

func (DirectorySetup) ResetLocal(workingDir string) error {
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

	return nil
}

func (DirectorySetup) ResetLayer(layerPath string) error {
	err := os.MkdirAll(filepath.Join(layerPath, publicAssetsDir), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(layerPath, tmpCacheAssetsDir), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (DirectorySetup) Link(layerPath, workingDir string) error {
	err := os.Symlink(filepath.Join(layerPath, publicAssetsDir), filepath.Join(workingDir, publicDir, assetsDir))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, tmpCacheAssetsDir), filepath.Join(workingDir, tmpDir, cacheDir, assetsDir))
	if err != nil {
		return err
	}

	return nil
}
