package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/gohugoio/hugo/config"
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
	fmt.Println("permalinks", permalinks)

	fpath := "content/posts/first-post.md"
	file, err := os.Open(fpath)
	if err != nil {
		log.Fatal("open", err)
	}

	page, err := ReadFrom(file)
	if err != nil {
		log.Fatal("read page", err)
	}

	fmt.Println(string(page.FrontMatter()))
	fmt.Println(string(page.Content()))
	meta, err := page.Metadata()
	if err != nil {
		log.Fatal("read meta", err)
	}
	c := NewContentFromMeta(meta)
	fmt.Println(c)
	link, err := pathPattern(permalinks["posts"]).Expand(c)
	if err != nil {
		log.Fatal("permalink expand", err)
	}
	fmt.Println(link)

	// test
	linkcfg := func(subdir string) string {
		format, ok := permalinks[subdir]
		if ok {
			return format
		}
		return defaultPermalinkFormat
	}
	listDirectory(source, linkcfg)
}

func parse(fullpath string) (*Content, error) {
	file, err := os.Open(fullpath)
	if err != nil {
		return nil, err
	}
	page, err := ReadFrom(file)
	if err != nil {
		return nil, err
	}
	meta, err := page.Metadata()
	if err != nil {
		return nil, err
	}
	c := NewContentFromMeta(meta)
	fmt.Println(c)

	return c, nil
}

func listDirectory(fullpath string, linkcfg func(string) string) error {
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
			fmt.Println(p, info.Name(), parent, rel, ext)
			c, err := parse(p)
			if err != nil {
				return err
			}
			c.Filepath = name

			if parent != "." {
				pattern := linkcfg(parent)
				link, err := pathPattern(pattern).Expand(c)
				if err != nil {
					return err
				}
				c.Permalink = link
			} else {
				c.Permalink = filename
			}
			fmt.Println(c)

			return nil
		})
}

func NewContentFromMeta(meta map[string]interface{}) *Content {
	return &Content{
		Title:      istring(meta["title"]),
		Slug:       istring(meta["slug"]),
		Categories: istringArr(meta["categories"]),
		Date:       idate(meta["date"]),
	}
}

func istring(input interface{}) string {
	str, ok := input.(string)
	if ok {
		return str
	}
	return ""
}

func idate(input interface{}) time.Time {
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

func istringArr(input interface{}) []string {
	str, ok := input.([]string)
	if ok {
		return str
	}
	return nil
}
