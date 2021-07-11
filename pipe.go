package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func pipe(cmd string, input io.Reader) ([]byte, error) {
	extpipe := exec.Command(cmd)
	extpipe.Stdin = input

	var pipeout bytes.Buffer
	extpipe.Stdout = &pipeout

	if err := extpipe.Start(); err != nil {
		return nil, fmt.Errorf("pipe start: %w", err)
	}

	if err := extpipe.Wait(); err != nil {
		return nil, fmt.Errorf("pipe wait: %w", err)
	}

	return pipeout.Bytes(), nil
}

func targetPath(dest, newext string, uglyURLs bool) (dir string, filename string) {
	filename = "index." + newext
	dir = dest

	if uglyURLs && dest != "index" {
		// make the last element in destination the file
		filename = filepath.Base(dest) + "." + newext
		// set the parent directory of that file to be the dir to create
		dir = filepath.Dir(dest)
	}

	if dest == "index" {
		dir = "."
	}

	return
}

func mkdir(dir string) (bool, error) {
	// skip if this exists
	if _, err := os.Stat(dir); err == nil {
		return false, nil
	}

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return false, fmt.Errorf("mkdir: %w", err)
	}

	return true, nil
}
