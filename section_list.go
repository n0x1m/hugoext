package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"time"
)

type Section struct {
	List []SectionEntry
	File string
}

type SectionEntry struct {
	Link    string
	Title   string
	Date    time.Time
	Summary string
}

func (section *Section) Write(file string) error {
	// sort section list
	sort.Slice(section.List, func(i, j int) bool {
		return section.List[i].Date.Before(section.List[j].Date)
	})

	var buf bytes.Buffer
	for _, file := range section.List {
		// TODO: this could be a template
		entry := "\n"
		entry += fmt.Sprintf("=> %s %s\n", file.Link, file.Title)
		entry += fmt.Sprintf("%v - %s\n", file.Date.Format("2006-01-02"), file.Summary)

		buf.Write([]byte(entry))
	}

	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(buf.Bytes())
	f.Close()
	return err
}
