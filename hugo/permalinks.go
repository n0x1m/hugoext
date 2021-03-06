package hugo

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PageMetadata struct {
	Title      string
	Slug       string
	Summary    string
	Categories []string
	Tags       []string
	Date       time.Time
	Draft      bool

	Filepath  string
	Subdir    string
	Permalink string
}

func init() {
	knownPermalinkAttributes = map[string]pageToPermaAttribute{
		"year":        pageToPermalinkDate,
		"month":       pageToPermalinkDate,
		"monthname":   pageToPermalinkDate,
		"day":         pageToPermalinkDate,
		"weekday":     pageToPermalinkDate,
		"weekdayname": pageToPermalinkDate,
		"yearday":     pageToPermalinkDate,
		"section":     pageToPermalinkSection,
		"title":       pageToPermalinkTitle,
		"slug":        pageToPermalinkSlugElseTitle,
		"filename":    pageToPermalinkFilename,
	}

	attributeRegexp = regexp.MustCompile(`:\w+`)
}

// pageToPermaAttribute is the type of a function which, given a page and a tag
// can return a string to go in that position in the page (or an error)
type pageToPermaAttribute func(*PageMetadata, string) (string, error)

// PathPattern represents a string which builds up a URL from attributes
type PathPattern string

// knownPermalinkAttributes maps :tags in a permalink specification to a
// function which, given a page and the tag, returns the resulting string
// to be used to replace that tag.
var knownPermalinkAttributes map[string]pageToPermaAttribute

var attributeRegexp *regexp.Regexp

// validate determines if a PathPattern is well-formed
func (pp PathPattern) validate() bool {
	fragments := strings.Split(string(pp[1:]), "/")
	var bail = false
	for i := range fragments {
		if bail {
			return false
		}
		if len(fragments[i]) == 0 {
			bail = true
			continue
		}

		matches := attributeRegexp.FindAllStringSubmatch(fragments[i], -1)
		if matches == nil {
			continue
		}

		for _, match := range matches {
			k := strings.ToLower(match[0][1:])
			if _, ok := knownPermalinkAttributes[k]; !ok {
				return false
			}
		}
	}
	return true
}

// Expand on a PathPattern takes a Content and returns the fully expanded Permalink
// or an error explaining the failure.
func (pp PathPattern) Expand(m *PageMetadata) (string, error) {
	if !pp.validate() {
		return "", fmt.Errorf("error")
	}
	sections := strings.Split(string(pp), "/")
	for i, field := range sections {
		if len(field) == 0 {
			continue
		}

		matches := attributeRegexp.FindAllStringSubmatch(field, -1)

		if matches == nil {
			continue
		}

		newField := field

		for _, match := range matches {
			attr := match[0][1:]
			callback, ok := knownPermalinkAttributes[attr]

			if !ok {
				return "", fmt.Errorf("err2")
			}

			newAttr, err := callback(m, attr)

			if err != nil {
				return "", fmt.Errorf("err3 %w", err)
			}

			newField = strings.Replace(newField, match[0], newAttr, 1)
		}

		sections[i] = newField
	}
	return strings.Join(sections, "/"), nil
}

func pageToPermalinkDate(m *PageMetadata, dateField string) (string, error) {
	// a Content contains a Node which provides a field Date, time.Time
	switch dateField {
	case "year":
		return strconv.Itoa(m.Date.Year()), nil
	case "month":
		return fmt.Sprintf("%02d", int(m.Date.Month())), nil
	case "monthname":
		return m.Date.Month().String(), nil
	case "day":
		return fmt.Sprintf("%02d", m.Date.Day()), nil
	case "weekday":
		return strconv.Itoa(int(m.Date.Weekday())), nil
	case "weekdayname":
		return m.Date.Weekday().String(), nil
	case "yearday":
		return strconv.Itoa(m.Date.YearDay()), nil
	}
	//TODO: support classic strftime escapes too
	// (and pass those through despite not being in the map)
	panic("coding error: should not be here")
}

// if the page has a slug, return the slug, else return the title
func pageToPermalinkSlugElseTitle(m *PageMetadata, a string) (string, error) {
	if m.Slug != "" {
		// Don't start or end with a -
		// TODO(bep) this doesn't look good... Set the Slug once.
		if strings.HasPrefix(m.Slug, "-") {
			m.Slug = m.Slug[1:len(m.Slug)]
		}

		if strings.HasSuffix(m.Slug, "-") {
			m.Slug = m.Slug[0 : len(m.Slug)-1]
		}
		return URLEscape(m.Slug)
	}
	return pageToPermalinkTitle(m, a)
}

// pageToPermalinkFilename returns the URL-safe form of the filename
func pageToPermalinkFilename(m *PageMetadata, _ string) (string, error) {
	return URLEscape(m.Filepath)
}

func pageToPermalinkTitle(m *PageMetadata, _ string) (string, error) {
	return URLEscape(m.Title)
}

func pageToPermalinkSection(m *PageMetadata, _ string) (string, error) {
	return URLEscape(m.Subdir)
}

func URLEscape(uri string) (string, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	return parsedURI.String(), nil
}
