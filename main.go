package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	cp "github.com/otiai10/copy"
)

type TemplateConfig struct {
	Name              string    `toml:"name"`
	MatchDependencies []string  `toml:"match_dependencies"`
	Build             StepBuild `toml:"build"`
	Run               StepRun   `toml:"run"`
}

type BuildStep struct {
	Image        string  `toml:"image"`
	Cmd          string  `toml:"cmd"`
	BeforeScript *string `toml:"before_script,omitempty"`
}

type StepBuild struct {
	BuildStep
	artifact string `toml:"artifact"`
}

type StepRun struct {
	BuildStep
}

var deploymentTemplates map[string]TemplateConfig

func main() {
	loadTemplates()

	connectDocker()

	log.Fatal(runApp("example", "./example/", ""))
}

func runApp(name, srcPath, env string) error {
	pkgJson, err := loadPackageJSON(srcPath)
	if err != nil {
		return err
	}

	deps, err := parseDependencies(pkgJson)
	if err != nil {
		return err
	}

	templateKey, err := findTemplateByDependencies(deps)
	if err != nil {
		return err
	}

	template := deploymentTemplates[templateKey]
	spew.Dump(template)

	port, err := getFreePort()
	if err != nil {
		return err
	}

	buildDir, err := os.MkdirTemp("./mounts", "")
	if err != nil {
		return err
	}
	// defer os.RemoveAll(tmpDir)

	log.Println("Mount:", buildDir)

	// Copy source code into container
	err = cp.Copy(srcPath, buildDir)
	if err != nil {
		return err
	}

	// Write build script into container
	beforeScript := ""
	if template.Build.BeforeScript != nil {
		beforeScript = *template.Build.BeforeScript
	}
	buildScript := fmt.Sprintf(
		"#!/bin/bash -v\ncd /runner/\n#Before Script:\n%s\n#Run Command:\n%s",
		beforeScript,
		template.Build.Cmd,
	)
	// err = os.Mkdir(buildDir, 0755)
	// if err != nil {
	// 	return err
	// }
	err = os.WriteFile(path.Join(buildDir, "r_build.sh"), []byte(buildScript), 0755)

	_, err = dockerRun(
		name,
		template.Build.Image,
		"/bin/ash /runner/r_build.sh",
		nil,
		strconv.Itoa(port),
		nil,
		buildDir,
	)
	if err != nil {
		return err
	}

	return nil
}
