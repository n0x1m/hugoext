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
	"github.com/n0x1m/hugoext/hugov0492"
	"github.com/spf13/afero"
)

const (
	defaultExt         = "gmi"
	defaultProcessor   = "md2gmi"
	defaultSource      = "content"
	defaultDestination = "public"
	defaultConfigPath  = "config.toml"
	defaultUglyURLs    = false

	defaultPermalinkFormat = "/:year/:month/:title/"
)

func main() {
	var ext, processor, source, destination, cfgpath string
	var uglyurls bool

	flag.StringVar(&ext, "ext", defaultExt, "ext to look for templates in ./layout")
	flag.StringVar(&processor, "proc", defaultProcessor, "processor to pipe markdown content through")
	flag.StringVar(&source, "source", defaultSource, "source directory")
	flag.StringVar(&destination, "destination", defaultDestination, "output directory")
	flag.StringVar(&cfgpath, "config", defaultConfigPath, "hugo config path")
	flag.BoolVar(&uglyurls, "ugly-urls", defaultUglyURLs, "use directories with index or .ext files")
	flag.Parse()

	osfs := afero.NewOsFs()
	cfg, err := config.FromFile(osfs, "config.toml")
	if err != nil {
		log.Fatal("config from file", err)
	}

	permalinks := cfg.GetStringMapString("permalinks")
	if permalinks == nil {
		log.Println("no permalinks from config loaded, using default: ", defaultPermalinkFormat)
	}

	// test
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
		tree.Files = append(tree.Files, file)
	}

	// call proc and pipe content through it, catch output of proc
	for i, file := range tree.Files {
		// fmt.Printf("%s -> %s (%d)\n", file.Source, file.Destination, len(file.Body))
		extpipe := exec.Command(processor)
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
		outdir := filepath.Join(destination, file.Destination)
		if file.Destination != "index" {
			newpath := filepath.Join(destination, file.Destination)
			if err = os.MkdirAll(newpath, os.ModePerm); err != nil {
				log.Println(err)
				continue
			}
			// TODO: change dir^
			// TODO: consider links
			if uglyurls {
				outdir += ".gmi"
				outfile = ""
			}
		} else {
			outdir = destination
		}

		fullpath := filepath.Join(outdir, outfile)
		newfile, err := os.Create(fullpath)
		if err != nil {
			log.Println(err)
			continue
		}

		n, _ := newfile.Write(file.NewBody)
		newfile.Close()

		fmt.Printf("written %s (%dbytes)\n", fullpath, n)
	}

	// TODO
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

	Body    []byte
	NewBody []byte
}

func parse(fullpath string) ([]byte, *hugov0492.Content, error) {
	file, err := os.Open(fullpath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	page, err := hugov0492.ReadFrom(file)
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

	if file.Parent != "." {
		link, err := hugov0492.PathPattern(pattern).Expand(c)
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

func NewContentFromMeta(meta map[string]interface{}) *hugov0492.Content {
	return &hugov0492.Content{
		Title:      stringFromInterface(meta["title"]),
		Slug:       stringFromInterface(meta["slug"]),
		Summary:    stringFromInterface(meta["summary"]),
		Categories: stringArrayFromInterface(meta["categories"]),
		Tags:       stringArrayFromInterface(meta["tags"]),
		Date:       dateFromInterface(meta["date"]),
	}
}

func stringFromInterface(input interface{}) string {
	str, ok := input.(string)
	if ok {
		return str
	}
	return ""
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
