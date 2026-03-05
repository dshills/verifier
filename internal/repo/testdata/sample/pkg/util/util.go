package util

import "os"

func ReadConfig(path string) ([]byte, error) {
	return os.ReadFile(path)
}
