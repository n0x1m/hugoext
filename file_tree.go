package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/n0x1m/hugoext/hugo"
)

type FileTree struct {
	Files []File
}

type File struct {
	Root        string
	Source      string
	Destination string
	Parent      string
	Name        string
	Extension   string
	Draft       bool

	Metadata hugo.PageMetadata
	Body     []byte
	NewBody  []byte
}

func parsePage(fullpath string) (hugo.Page, error) {
	file, err := os.Open(fullpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	page, err := hugo.ReadFrom(file)
	if err != nil {
		return nil, err
	}
	return page, nil
}

func parseMetadata(page hugo.Page) (*hugo.PageMetadata, error) {
	meta, err := page.Metadata()
	if err != nil {
		return nil, err
	}
	c := NewContentFromMeta(meta)

	return c, nil
}

func destinationPath(file *File, pattern string) error {
	p, err := parsePage(file.Source)
	if err != nil {
		return err
	}

	// create content
	c, err := parseMetadata(p)
	c.Filepath = file.Name

	if file.Parent != "." {
		link, err := hugo.PathPattern(pattern).Expand(c)
		if err != nil {
			return err
		}
		file.Destination = link
	} else {
		file.Destination = strings.TrimLeft(file.Name, "_")
	}

	file.Draft = c.Draft
	file.Metadata = *c
	file.Body = p.Body()

	return nil
}

func collectFiles(fullpath string, filechan chan File) error {
	defer close(filechan)
	return filepath.Walk(fullpath,
		func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(fullpath, p)
			if err != nil {
				return err
			}

			filename := info.Name()
			ext := path.Ext(filename)
			name := filename[0 : len(filename)-len(ext)]
			parent := filepath.Dir(rel)

			filechan <- File{
				Root:      fullpath,
				Source:    p,
				Name:      name,
				Extension: ext,
				Parent:    parent,
			}

			return nil
		})
}
