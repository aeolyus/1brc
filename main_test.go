package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	sampleInputDir  = "./test/samples"
	sampleInputExt  = ".txt"
	sampleOutputExt = ".out"
)

func TestEval(t *testing.T) {
	inputFiles, err := findFiles(sampleInputDir, sampleInputExt)
	if err != nil {
		t.Errorf("could not get input files: %v", err)
	}
	for _, file := range inputFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			actual, err := eval(file + sampleInputExt)
			if err != nil {
				t.Errorf("could not evaluate input: %v", err)
			}
			expected, err := readFile(file + sampleOutputExt)
			if err != nil {
				t.Errorf("could not read output file: %v", err)
			}
			assert.Equal(t, expected, actual)
		})
	}
}

func readFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}
	return string(content), nil
}

func findFiles(dir string, ext string) ([]string, error) {
	filePaths := []string{}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ext {
			f := filepath.Join(dir, file.Name())
			filePaths = append(filePaths, f[:len(f)-len(ext)])
		}
	}
	return filePaths, nil
}
