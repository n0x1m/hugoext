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

	Draft bool

	Body    []byte
	NewBody []byte
}

func parse(fullpath string) ([]byte, *hugo.Content, error) {
	file, err := os.Open(fullpath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	page, err := hugo.ReadFrom(file)
	if err != nil {
		return nil, nil, err
	}
	meta, err := page.Metadata()
	if err != nil {
		return nil, nil, err
	}
	c := NewContentFromMeta(meta)
	body := page.FrontMatter()
	body = append(body, '\n')
	body = append(body, page.Content()...)

	return body, c, nil
}

func destinationPath(file *File, pattern string) error {
	body, c, err := parse(file.Source)
	if err != nil {
		return err
	}
	c.Filepath = file.Name
	file.Body = body
	file.Draft = c.Draft

	if file.Parent != "." {
		link, err := hugo.PathPattern(pattern).Expand(c)
		if err != nil {
			return err
		}
		file.Destination = link
	} else {
		file.Destination = strings.TrimLeft(file.Name, "_")
	}

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
