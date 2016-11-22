package api

import (
	. "fmt"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

// rawResolveResult represents the json structure of the resolver's results.
// This sturcture is consumed to produce a ResolveResult with proper typing.
type rawResolveResult struct {
	Path     []map[string]interface{} `json:"path"`
	Children []map[string]interface{} `json:"children"`
}

type ResolveResult struct {
	Path     []interface{} `json:"path"`
	Children []interface{} `json:"children"`
}

// stringToDateTimeHook adds mapstructure support for time.Time reflection via string.
// Ref: https://github.com/mitchellh/mapstructure/issues/41
func stringToDateTimeHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(time.Time{}) && f == reflect.TypeOf("") {
		return time.Parse(time.RFC3339, data.(string))
	}

	return data, nil
}

func newDecoderConfig() *mapstructure.DecoderConfig {
	return &mapstructure.DecoderConfig{
		DecodeHook: stringToDateTimeHook,
		TagName:    "json",
	}
}

// this should be removed later
func check(err error) {
	if err != nil {
		Println(err)
		os.Exit(1)
	}
}

func decode(config *mapstructure.DecoderConfig, src interface{}) {
	decoder, err := mapstructure.NewDecoder(config)
	check(err)

	err = decoder.Decode(src)
	check(err)
}

// addDynamicNode will take an untyped string map and add it to a slice.
func (r *ResolveResult) addDynamicNode(x map[string]interface{}, slice *[]interface{}) {

	switch x["node_type"].(string) {
	case "group":
		var obj Group
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		*slice = append(*slice, &obj)

	case "project":
		var obj Project
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		*slice = append(*slice, &obj)

	case "session":
		var obj Session
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		*slice = append(*slice, &obj)

	case "acquisition":
		var obj Acquisition
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		*slice = append(*slice, &obj)

	case "file":
		var obj File
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		*slice = append(*slice, &obj)

	default:
		Println("Unknown dynamic node type " + x["node_type"].(string))
	}
}

type resolvePath struct {
	Path []string `json:"path"`
}

func (c *Client) ResolvePath(path []string) (*ResolveResult, *http.Response, error) {
	var aerr *ApiError
	var raw rawResolveResult
	var result ResolveResult

	if path[0] == "" {
		path = []string{}
	}

	request := resolvePath{
		Path: path,
	}

	resp, err := c.S.New().Post("resolve").BodyJSON(&request).Receive(&raw, &aerr)

	err = coalesce(err, aerr)
	if err != nil {
		return nil, resp, err
	}

	for _, x := range raw.Path {
		result.addDynamicNode(x, &result.Path)
	}
	for _, x := range raw.Children {
		result.addDynamicNode(x, &result.Children)
	}

	return &result, resp, coalesce(err, aerr)
}
