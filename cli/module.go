package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/itchyny/gojq"
)

type moduleLoader struct {
	paths []string
}

func (l *moduleLoader) LoadInitModules() ([]*gojq.Module, error) {
	var ms []*gojq.Module
	for _, path := range l.paths {
		if filepath.Base(path) != ".jq" {
			continue
		}
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if fi.IsDir() {
			continue
		}
		cnt, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		m, err := parseModule(path, string(cnt))
		if err != nil {
			return nil, &queryParseError{"query in module", path, string(cnt), err}
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func (l *moduleLoader) LoadModule(string) (*gojq.Module, error) {
	panic("moduleLoader#LoadModule: unreachable")
}

func (l *moduleLoader) LoadModuleWithMeta(name string, meta map[string]interface{}) (*gojq.Module, error) {
	path, err := l.lookupModule(name, ".jq", meta)
	if err != nil {
		return nil, err
	}
	cnt, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m, err := parseModule(path, string(cnt))
	if err != nil {
		return nil, &queryParseError{"query in module", path, string(cnt), err}
	}
	return m, nil
}

func (l *moduleLoader) LoadJSONWithMeta(name string, meta map[string]interface{}) (interface{}, error) {
	path, err := l.lookupModule(name, ".json", meta)
	if err != nil {
		return nil, err
	}
	return slurpFile(path)
}

func (l *moduleLoader) lookupModule(name, extension string, meta map[string]interface{}) (string, error) {
	paths := l.paths
	if path := searchPath(meta); path != "" {
		paths = append([]string{path}, paths...)
	}
	for _, base := range paths {
		path := filepath.Clean(filepath.Join(base, name+extension))
		if _, err := os.Stat(path); err == nil {
			return path, err
		}
		path = filepath.Clean(filepath.Join(base, name, filepath.Base(name)+extension))
		if _, err := os.Stat(path); err == nil {
			return path, err
		}
	}
	return "", fmt.Errorf("module not found: %q", name)
}

// This is a dirty hack to implement the "search" field.
// Note that gojq package should not depend on the filesystem.
func parseModule(path, cnt string) (*gojq.Module, error) {
	m, err := gojq.ParseModule(cnt)
	if err != nil {
		return nil, err
	}
	for _, i := range m.Imports {
		if i.Meta == nil {
			continue
		}
		i.Meta.KeyVals = append(
			i.Meta.KeyVals,
			gojq.ConstObjectKeyVal{
				Key: "$$path",
				Val: &gojq.ConstTerm{Str: strconv.Quote(path)},
			},
		)
	}
	return m, nil
}

func searchPath(meta map[string]interface{}) string {
	x, ok := meta["$$path"]
	if !ok {
		return ""
	}
	path, ok := x.(string)
	if !ok {
		return ""
	}
	x, ok = meta["search"]
	if !ok {
		return ""
	}
	s, ok := x.(string)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(path), s)
}
