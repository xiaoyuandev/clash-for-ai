package tooling

import (
	"os"
	"path/filepath"
)

func readOptionalText(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(content), nil
}

func ensureDir(filePath string) error {
	return os.MkdirAll(filepath.Dir(filePath), 0o755)
}
