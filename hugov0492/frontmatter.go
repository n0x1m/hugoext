package hugov0492

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

// FrontmatterType represents a type of frontmatter.
type FrontmatterType struct {
	// Parse decodes content into a Go interface.
	Parse func([]byte) (map[string]interface{}, error)

	markstart, markend []byte // starting and ending delimiters
	includeMark        bool   // include start and end mark in output
}

// DetectFrontMatter detects the type of frontmatter analysing its first character.
func DetectFrontMatter(mark rune) (f *FrontmatterType) {
	switch mark {
	case '-':
		return &FrontmatterType{HandleYAMLMetaData, []byte(YAMLDelim), []byte(YAMLDelim), false}
	case '+':
		return &FrontmatterType{HandleTOMLMetaData, []byte(TOMLDelim), []byte(TOMLDelim), false}
	case '{':
		return &FrontmatterType{HandleJSONMetaData, []byte{'{'}, []byte{'}'}, true}
	default:
		return nil
	}
}

// HandleYAMLMetaData unmarshals YAML-encoded datum and returns a Go interface
// representing the encoded data structure.
func HandleYAMLMetaData(datum []byte) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	err := yaml.Unmarshal(datum, &m)

	// To support boolean keys, the `yaml` package unmarshals maps to
	// map[interface{}]interface{}. Here we recurse through the result
	// and change all maps to map[string]interface{} like we would've
	// gotten from `json`.
	if err == nil {
		for k, v := range m {
			if vv, changed := stringifyMapKeys(v); changed {
				m[k] = vv
			}
		}
	}

	return m, err
}

// HandleTOMLMetaData unmarshals TOML-encoded datum and returns a Go interface
// representing the encoded data structure.
func HandleTOMLMetaData(datum []byte) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	datum = removeTOMLIdentifier(datum)

	_, err := toml.Decode(string(datum), &m)

	return m, err

}

// removeTOMLIdentifier removes, if necessary, beginning and ending TOML
// frontmatter delimiters from a byte slice.
func removeTOMLIdentifier(datum []byte) []byte {
	ld := len(datum)
	if ld < 8 {
		return datum
	}

	b := bytes.TrimPrefix(datum, []byte(TOMLDelim))
	if ld-len(b) != 3 {
		// No TOML prefix trimmed, so bail out
		return datum
	}

	b = bytes.Trim(b, "\r\n")
	return bytes.TrimSuffix(b, []byte(TOMLDelim))
}

// HandleJSONMetaData unmarshals JSON-encoded datum and returns a Go interface
// representing the encoded data structure.
func HandleJSONMetaData(datum []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})

	if datum == nil {
		// Package json returns on error on nil input.
		// Return an empty map to be consistent with our other supported
		// formats.
		return m, nil
	}

	err := json.Unmarshal(datum, &m)
	return m, err
}

// stringifyMapKeys recurses into in and changes all instances of
// map[interface{}]interface{} to map[string]interface{}. This is useful to
// work around the impedence mismatch between JSON and YAML unmarshaling that's
// described here: https://github.com/go-yaml/yaml/issues/139
//
// Inspired by https://github.com/stripe/stripe-mock, MIT licensed
func stringifyMapKeys(in interface{}) (interface{}, bool) {
	switch in := in.(type) {
	case []interface{}:
		for i, v := range in {
			if vv, replaced := stringifyMapKeys(v); replaced {
				in[i] = vv
			}
		}
	case map[interface{}]interface{}:
		res := make(map[string]interface{})
		var (
			ok  bool
			err error
		)
		for k, v := range in {
			var ks string

			if ks, ok = k.(string); !ok {
				ks, err = cast.ToStringE(k)
				if err != nil {
					ks = fmt.Sprintf("%v", k)
				}
			}
			if vv, replaced := stringifyMapKeys(v); replaced {
				res[ks] = vv
			} else {
				res[ks] = v
			}
		}
		return res, true
	}

	return nil, false
}
