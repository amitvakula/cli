package api

import (
	. "fmt"
	"os"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

type RawResolveResult struct {
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

// AddDynamicNode will take an untyped string map and add it to the ResolveResult.
func (r *ResolveResult) AddDynamicNode(x map[string]interface{}) {

	switch x["node_type"].(string) {
	case "group":
		var obj Group
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		r.Path = append(r.Path, &obj)

	case "project":
		var obj Project
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		r.Path = append(r.Path, &obj)

	case "session":
		var obj Session
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		r.Path = append(r.Path, &obj)

	case "acquisition":
		var obj Acquisition
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		r.Path = append(r.Path, &obj)

	case "file":
		var obj File
		config := newDecoderConfig()
		config.Result = &obj
		decode(config, x)
		r.Path = append(r.Path, &obj)

	default:
		Println("Unknown dynamic node type " + x["node_type"].(string))
	}
}
