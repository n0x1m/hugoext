package main

import (
	"fmt"
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

func (file *File) Write(dest, newext string, uglyURLs bool) (string, error) {
	outdir, outfile := targetPath(file.Destination, newext, uglyURLs)

	// ensure directory exists
	newdir := filepath.Join(dest, outdir)
	if made, err := mkdir(newdir); err != nil {
		return "", err
	} else if made {
		fmt.Printf("mkdir %s\n", newdir)
	}

	fullpath := filepath.Join(newdir, outfile)

	// create file based on directory and filename
	newfile, err := os.Create(fullpath)
	if err != nil {
		return fullpath, err
	}

	if _, err = newfile.Write(file.NewBody); err != nil {
		return fullpath, err
	}

	newfile.Close()
	return fullpath, nil
}

func parsePage(fullpath string) (hugo.Page, error) {
	file, err := os.Open(fullpath)
	if err != nil {
		return nil, fmt.Errorf("file open: %w", err)
	}
	defer file.Close()

	page, err := hugo.ReadFrom(file)
	if err != nil {
		return nil, fmt.Errorf("hugo read from: %w", err)
	}

	return page, nil
}

func parseMetadata(page hugo.Page) (*hugo.PageMetadata, error) {
	meta, err := page.Metadata()
	if err != nil {
		return nil, fmt.Errorf("page metadata: %w", err)
	}

	c := NewContentFromMeta(meta)

	return c, nil
}

func destinationPath(file *File, pattern string) error {
	p, err := parsePage(file.Source)
	if err != nil {
		return fmt.Errorf("parse page: %w", err)
	}

	// create content
	c, err := parseMetadata(p)
	if err != nil {
		return fmt.Errorf("parse metadata: %w", err)
	}

	c.Filepath = file.Name

	if file.Parent != "." {
		link, err := hugo.PathPattern(pattern).Expand(c)
		if err != nil {
			return fmt.Errorf("hugo pathpattern: %w", err)
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

	err := filepath.Walk(fullpath,
		func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(fullpath, p)
			if err != nil {
				return fmt.Errorf("rel path: %w", err)
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
	if err != nil {
		return fmt.Errorf("filetree walk: %w", err)
	}

	return nil
}
