package auth_test

import (
	"io/fs"
	"os"
)

func statKey(path string) (fs.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}
