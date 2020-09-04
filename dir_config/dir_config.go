package dir_config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/go-errors/errors"
	"github.com/moisespsena-go/maps"
)

var ErrSiteDisabled = errors.New("site disabled")

func KeyName(fileName string) (v string) {
	if pos := strings.LastIndexByte(fileName, '.'); pos >= 0 {
		v = fileName[0:pos]
	} else {
		v = fileName
	}
	if v[0] == '/' || v[0] == '\\' || v[0] == '_' {
		v = v[1:]
	}
	return
}

func ValidConfigFile(fileName string) bool {
	return fileName[0] != '.' && (strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml"))
}

func LoadMainConfig(dir string, keyNamer ...func(dir, name string, isdir bool) string) (mainConfig maps.MapSI, err error) {
	var fileInfos []os.FileInfo

	if fileInfos, err = ioutil.ReadDir(dir); err != nil {
		return
	}

	var (
		main     string
		mainKeys [][2]string
		subs []string
		keyNameOf = func(name string, isdir bool) (keyName string) {
			for _, knr := range keyNamer {
				if keyName = knr(dir, name, isdir); keyName != "" {
					break
				}
			}
			if keyName == "" {
				keyName = KeyName(name)
			}
			return
		}
	)

	for _, f := range fileInfos {
		if f.IsDir() {
			subs = append(subs, f.Name())
		}
		if ValidConfigFile(f.Name()) {
			var keyName = keyNameOf(f.Name(), false)
			if keyName == "config" {
				main = f.Name()
				continue
			}
			mainKeys = append(mainKeys, [2]string{f.Name(), keyName})
		}
	}

	if main != "" {
		if mainConfig, err = load(filepath.Join(dir, main)); err != nil {
			return nil, fmt.Errorf("load main config %q failed: %v", main, err)
		}
	}

	if mainConfig == nil {
		mainConfig = make(maps.MapSI)
	}

	for _, name := range mainKeys {
		pth := filepath.Join(dir, name[0])
		if cfg, err := load(pth); err != nil {
			return nil, fmt.Errorf("load main %q config failed: %v", pth, err)
		} else {
			mainConfig.Set(name[1], cfg)
		}
	}

	for _, name := range subs {
		var (
			pth  = filepath.Join(dir, name)
			sub  maps.MapSI
		)
		if sub, err = LoadMainConfig(pth, keyNamer...); err != nil {
			return nil, errors.WrapPrefix(err, "dir `"+pth+"`", 1)
		} else if sub != nil && len(sub) > 0 {
			name = keyNameOf(name, true)
			if _, ok := mainConfig[name]; ok {
				if err = sub.CopyTo(mainConfig[name]); err != nil {
					return nil, errors.WrapPrefix(err, "cfg `"+pth+" copy to parent failed`", 1)
				}
			} else {
				mainConfig[name] = sub
			}
		}
	}

	return
}

func load(path string) (data maps.MapSI, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var cfg map[string]interface{}

	if err = yaml.NewDecoder(f).Decode(&cfg); err != nil && err != io.EOF {
		return nil, err
	}
	return cfg, nil
}
