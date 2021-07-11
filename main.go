package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gohugoio/hugo/config"
	"github.com/n0x1m/hugoext/hugo"
	"github.com/spf13/afero"
)

const (
	defaultExt         = "md"
	defaultProcessor   = ""
	defaultSource      = "content"
	defaultDestination = "public"
	defaultConfigPath  = "config.toml"
	defaultUglyURLs    = false

	defaultPermalinkFormat = "/:year/:month/:title/"
)

func main() {
	var ext, pipe, source, destination, cfgPath string
	var uglyURLs, noSectionList, withDrafts bool

	flag.StringVar(&ext, "ext", defaultExt, "ext to look for templates in ./layout")
	flag.StringVar(&pipe, "pipe", defaultProcessor, "pipe markdown to this program for content processing")
	flag.StringVar(&source, "source", defaultSource, "source directory")
	flag.StringVar(&destination, "destination", defaultDestination, "output directory")
	flag.StringVar(&cfgPath, "config", defaultConfigPath, "hugo config path")
	flag.BoolVar(&uglyURLs, "ugly-urls", false, "use directories with index or .ext files")
	flag.BoolVar(&noSectionList, "no-section-list", false, "disable auto append of section content lists")
	flag.BoolVar(&withDrafts, "enable-withDrafts", false, "include withDrafts in processing and output")
	flag.Parse()

	fmt.Printf("converting markdown to %v with %v\n", ext, pipe)

	osfs := afero.NewOsFs()
	cfg, err := config.FromFile(osfs, "config.toml")
	if err != nil {
		log.Fatal("config from file", err)
	}

	permalinks := cfg.GetStringMapString("permalinks")
	if permalinks == nil {
		log.Println("no permalinks from config loaded, using default: ", defaultPermalinkFormat)
	}

	linkpattern := func(subdir string) string {
		format, ok := permalinks[subdir]
		if ok {
			return format
		}
		return defaultPermalinkFormat
	}

	// iterate through file tree source
	files := make(chan File)
	go collectFiles(source, files)

	// for each file, get destination path, switch file extension, remove underscore for index
	var tree FileTree
	for file := range files {
		pattern := linkpattern(file.Parent)
		err := destinationPath(&file, pattern)
		if err != nil {
			fmt.Println(err, file)
			continue
		}
		//fmt.Printf("%s -> %s (%d)\n", file.Source, file.Destination, len(file.Body))
		if file.Draft && !withDrafts {
			fmt.Printf("skipping draft %s (%dbytes)\n", file.Source, len(file.Body))
			continue
		}
		tree.Files = append(tree.Files, file)
	}

	// call proc and pipe content through it, catch output of proc
	for i, file := range tree.Files {
		// fmt.Printf("%s -> %s (%d)\n", file.Source, file.Destination, len(file.Body))
		extpipe := exec.Command(pipe)
		buf := bytes.NewReader(file.Body)
		extpipe.Stdin = buf

		var procout bytes.Buffer
		extpipe.Stdout = &procout

		extpipe.Start()
		extpipe.Wait()

		// write to source
		tree.Files[i].NewBody = procout.Bytes()
		fmt.Printf("processed %s (%dbytes)\n", file.Source, len(tree.Files[i].Body))
	}

	newpath := filepath.Join(".", "public")
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	// write to destination
	for _, file := range tree.Files {
		//fmt.Printf("%s -> %s (%d)\n", file.Source, file.Destination, len(file.Body))

		outfile := "index." + ext
		outdir := file.Destination

		if uglyURLs && file.Destination != "index" {
			// make the last element in destination the file
			outfile = filepath.Base(file.Destination) + "." + ext
			// set the parent directory of that file to be the dir to create
			outdir = filepath.Dir(file.Destination)
		}

		if file.Destination == "index" {
			outdir = "."
		}

		// ensure directory exists
		newpath := filepath.Join(destination, outdir)
		fmt.Printf("mkdir %s\n", newpath)
		if err = os.MkdirAll(newpath, os.ModePerm); err != nil {
			log.Fatalf("cannot directory file %s, error: %v", newpath, err)
		}

		// create file based on directory and filename
		fullpath := filepath.Join(newpath, outfile)
		newfile, err := os.Create(fullpath)
		if err != nil {
			log.Fatalf("cannot create file %s, error: %v", fullpath, err)
		}

		n, err := newfile.Write(file.NewBody)
		if err != nil {
			log.Fatalf("cannot write file %s, error: %v", fullpath, err)
		}
		newfile.Close()

		fmt.Printf("written %s (%dbytes)\n", fullpath, n)
	}

	// TODO
	//
	// append section listings
	// => link title line
	// date - summary block

	// TODO
	// in watch mode, compare timestamps of files before replacement, keep index?
	// check/replace links
	// write rss?
	// write listings from template?
}

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

func NewContentFromMeta(meta map[string]interface{}) *hugo.Content {
	return &hugo.Content{
		Title:      stringFromInterface(meta["title"]),
		Slug:       stringFromInterface(meta["slug"]),
		Summary:    stringFromInterface(meta["summary"]),
		Categories: stringArrayFromInterface(meta["categories"]),
		Tags:       stringArrayFromInterface(meta["tags"]),
		Date:       dateFromInterface(meta["date"]),
		Draft:      boolFromInterface(meta["draft"]),
	}
}

func stringFromInterface(input interface{}) string {
	str, _ := input.(string)
	return str
}

func boolFromInterface(input interface{}) bool {
	v, _ := input.(bool)
	return v
}

func dateFromInterface(input interface{}) time.Time {
	str, ok := input.(string)
	if !ok {
		return time.Now()

	}
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		// try just date, or give up
		t, err := time.Parse("2006-01-02", str)
		if err != nil {
			return time.Now()
		}
		return t
	}
	return t
}

func stringArrayFromInterface(input interface{}) []string {
	strarr, ok := input.([]interface{})
	if ok {
		var out []string
		for _, str := range strarr {
			out = append(out, stringFromInterface(str))
		}
		return out
	}
	return nil
}
