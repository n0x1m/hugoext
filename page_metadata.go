package main

import (
	"time"

	"github.com/n0x1m/hugoext/hugo"
)

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
