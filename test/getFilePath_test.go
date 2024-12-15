package test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFileName(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	dir = filepath.Base(dir)
	print(dir)
}
