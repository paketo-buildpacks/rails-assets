package railsassets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// NodeLockfileChecker checks for the existence of node package manager lock file
type NodeLockfileChecker struct{}

// NewNodeLockfileParser initializes a NodeLockfileParser instance.
func NewNodeLockfileChecker() NodeLockfileChecker {
	return NodeLockfileChecker{}
}

// Checks for the existence of either a yarn.lock or package-lock.json.
func (p NodeLockfileChecker) Check(path string) (bool, error) {
	var result = false
	var resultErr error

	for _, lockFile := range [2]string{"yarn.lock", "package-lock.json"} {
		var _, err = os.Stat(filepath.Join(path, lockFile))
		if err == nil {
			result = true
			break
		} else {
			if !errors.Is(err, os.ErrNotExist) {
				resultErr = fmt.Errorf("failed to stat %s: %w", lockFile, err)
				break
			}
		}
	}

	return result, resultErr
}
