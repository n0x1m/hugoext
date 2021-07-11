package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gohugoio/hugo/config"
	"github.com/spf13/afero"
)

const (
	defaultExt           = "md"
	defaultProcessor     = ""
	defaultSource        = "content"
	defaultDestination   = "public"
	defaultConfigPath    = "config.toml"
	defaultSectionOnRoot = "posts"

	defaultPermalinkFormat = "/:year/:month/:title/"
)

func main() {
	var ext, pipecmd, source, destination, cfgPath, seconOnRoot string
	var noSectionList bool

	flag.StringVar(&ext, "ext", defaultExt, "ext to look for templates in ./layout")
	flag.StringVar(&pipecmd, "pipe", defaultProcessor, "pipe markdown to this program for content processing")
	flag.StringVar(&source, "source", defaultSource, "source directory")
	flag.StringVar(&destination, "destination", defaultDestination, "output directory")
	flag.StringVar(&cfgPath, "config", defaultConfigPath, "hugo config path")
	flag.BoolVar(&noSectionList, "no-section-list", false, "disable auto append of section content lists")
	flag.StringVar(&seconOnRoot, "section-on-root", defaultSectionOnRoot, "if append sections, add this one on the root")
	flag.Parse()

	// what are we doing
	fmt.Printf("hugoext: converting hugo markdown to %v with %v\n", ext, pipecmd)

	// config
	cfg, err := config.FromFile(afero.NewOsFs(), "config.toml")
	if err != nil {
		log.Fatal("config from file", err)
	}

	uglyURLs := cfg.GetBool("uglyURLs")
	if !cfg.IsSet("uglyURLs") {
		fmt.Println("config: no uglyURLs set, using default: ", uglyURLs)
	}

	buildDrafts := cfg.GetBool("buildDrafts")
	if !cfg.IsSet("buildDrafts") {
		fmt.Println("config: no buildDrafts set, using default: ", buildDrafts)
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
			log.Fatalf("failed to derive destination for %v error: %v", file.Source, err)
		}

		if file.Draft && !buildDrafts {
			fmt.Printf("skipping draft %s (%dbytes)\n", file.Source, len(file.Body))
			continue
		}
		tree.Files = append(tree.Files, file)
	}

	// call proc and pipe content through it, catch output of proc
	for i, file := range tree.Files {
		buf := bytes.NewReader(file.Body)
		out := pipe(pipecmd, buf)

		// write to source
		tree.Files[i].NewBody = out
		fmt.Printf("processed %s (%dbytes)\n", file.Source, len(tree.Files[i].Body))
	}

	newdir := filepath.Join(".", "public")
	if made, err := mkdir(newdir); err != nil {
		log.Fatal(err)
	} else if made {
		fmt.Printf("mkdir %s\n", newdir)
	}

	// write to destination
	for _, file := range tree.Files {
		outdir, outfile := targetPath(file.Destination, ext, uglyURLs)

		// ensure directory exists
		newdir := filepath.Join(destination, outdir)
		if made, err := mkdir(newdir); err != nil {
			log.Fatal(err)
		} else if made {
			fmt.Printf("mkdir %s\n", newdir)
		}

		fullpath := filepath.Join(newdir, outfile)

		// create file based on directory and filename
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

	// we're done if we
	if noSectionList {
		return
	}

	// aggregate sections and section entries
	sections := make(map[string]*Section)
	for _, file := range tree.Files {
		// not a section
		if file.Parent == "." {
			continue
		}
		name := file.Parent
		sectionFile := filepath.Join(destination, file.Parent, "index."+ext)

		//fmt.Printf("section %v\n", sectionFile)
		link := file.Destination
		if uglyURLs {
			link += "." + ext
		}

		//fmt.Printf("link %v\n", link)
		if _, ok := sections[name]; !ok {
			sections[name] = &Section{File: sectionFile}
		}

		sections[name].List = append(sections[name].List, SectionEntry{
			Date:    file.Metadata.Date,
			Title:   file.Metadata.Title,
			Summary: file.Metadata.Summary,
			Link:    link,
		})
	}

	for name, section := range sections {
		section.Write(section.File)
		fmt.Printf("written section listing %s to %s\n", name, section.File)
	}

	section, ok := sections[seconOnRoot]
	if ok {
		sectionFile := filepath.Join(destination, "index."+ext)
		// TODO: come up with sth better as one might have content there.
		os.Remove(sectionFile)
		section.Write(sectionFile)
		fmt.Printf("written section listing for root to %s\n", section.File)
	}
}

func pipe(cmd string, input io.Reader) []byte {
	extpipe := exec.Command(cmd)
	extpipe.Stdin = input

	var pipeout bytes.Buffer
	extpipe.Stdout = &pipeout

	extpipe.Start()
	extpipe.Wait()

	return pipeout.Bytes()
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
		return false, err
	}

	return true, nil
}
