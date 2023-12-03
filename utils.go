package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/samber/lo"
)

func ptr[T any](v T) *T {
	return &v
}

func loadTemplates() {
	// Load deployment templates
	deploymentTemplates = make(map[string]TemplateConfig)

	matches, err := filepath.Glob("./templates/*.toml")
	if err != nil {
		log.Fatal(err)
	}

	var config TemplateConfig
	for _, template := range matches {
		_, err := toml.DecodeFile(template, &config)
		if err != nil {
			log.Fatal(err)
		}
	}

	configKey := strings.TrimSuffix(filepath.Base(matches[0]), ".toml")
	deploymentTemplates[configKey] = config
}

func findTemplateByDependencies(deps map[string]string) (string, error) {
	for key, value := range deploymentTemplates {
		for _, dep := range value.MatchDependencies {
			if _, ok := deps[dep]; ok {
				return key, nil
			}
		}
	}

	return "", errors.New("No template found")
}

func loadPackageJSON(srcPath string) (map[string]interface{}, error) {
	pkgB, err := os.ReadFile(filepath.Join(srcPath, "package.json"))
	if err != nil {
		return nil, err
	}

	var pkgJson map[string]interface{}
	err = json.Unmarshal(pkgB, &pkgJson)
	if err != nil {
		return nil, err
	}

	return pkgJson, nil
}

func parseDependencies(pkgJson map[string]interface{}) (map[string]string, error) {
	deps := make(map[string]string)
	switch v := pkgJson["dependencies"].(type) {
	case map[string]interface{}:
		for key, value := range v {
			switch v2 := value.(type) {
			case string:
				deps[key] = v2
				break
			default:
				return nil, errors.New(fmt.Sprintf("Package.json -> Dependencies -> Value: Must be a string. Got: %s", reflect.TypeOf(v2)))
			}
		}
		break
	default:
		return nil, errors.New(fmt.Sprintf("Package.json -> Dependencies: Must be a map[string]string. Got: %s", reflect.TypeOf(v)))
	}

	return deps, nil
}

func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, 755)
	}

	return nil
}

// Trying to not copy structs here

// TODO: check if we break references here
func getAllDeployments() []Deployment {
	return lo.FlatMap(apps, func(app App, index int) []Deployment {
		return apps[index].Deployments
	})
}

func getDeploymentByDomain(domain string) *Deployment {
	allDeployments := getAllDeployments()
	_, i, found := lo.FindIndexOf(allDeployments, func(deployment Deployment) bool {
		return deployment.GetDomain() == domain
	})
	if !found {
		return nil
	}
	return &allDeployments[i]
}

func getAppById(id string) *App {
	_, i, found := lo.FindIndexOf(apps, func(app App) bool {
		return app.Id == id
	})
	if !found {
		return nil
	}
	return &apps[i]
}

func getDeploymentById(id string) *Deployment {
	allDeployments := getAllDeployments()
	_, i, found := lo.FindIndexOf(allDeployments, func(deployment Deployment) bool {
		return deployment.Id == id
	})
	if !found {
		return nil
	}
	return &allDeployments[i]
}

func makeId() string {
	return lo.RandomString(12, lo.LettersCharset)
}
