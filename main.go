package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gohugoio/hugo/config"
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
	var noSectionList, withDrafts bool

	flag.StringVar(&ext, "ext", defaultExt, "ext to look for templates in ./layout")
	flag.StringVar(&pipe, "pipe", defaultProcessor, "pipe markdown to this program for content processing")
	flag.StringVar(&source, "source", defaultSource, "source directory")
	flag.StringVar(&destination, "destination", defaultDestination, "output directory")
	flag.StringVar(&cfgPath, "config", defaultConfigPath, "hugo config path")
	flag.BoolVar(&noSectionList, "no-section-list", false, "disable auto append of section content lists")
	flag.BoolVar(&withDrafts, "enable-withDrafts", false, "include withDrafts in processing and output")
	flag.Parse()

	// what are we doing
	fmt.Printf("converting hugo markdown to %v with %v\n", ext, pipe)

	// config
	cfg, err := config.FromFile(afero.NewOsFs(), "config.toml")
	if err != nil {
		log.Fatal("config from file", err)
	}

	uglyURLs := cfg.GetBool("uglyURLs")
	if !cfg.IsSet("uglyURLs") {
		fmt.Println("config: no uglyURLs set, using default: ", uglyURLs)
	}

	permalinks := cfg.GetStringMapString("permalinks")
	if permalinks == nil {
		fmt.Println("config: no permalinks set, using default: ", defaultPermalinkFormat)
	}

	linkpattern := func(section string) string {
		format, ok := permalinks[section]
		if ok {
			return format
		}
		return defaultPermalinkFormat
	}

	// process sources

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
