package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	cp "github.com/otiai10/copy"
)

type App struct {
	Id               string  `json:"id"`
	Name             string  `json:"name"`
	ProjectName      string  `json:"project_name"`
	ContainerId      *string `json:"container_id"`
	BuildContainerId *string `json:"build_container_id"`
	Port             *string `json:"port"`
	Env              *string `json:"env"`
	Status           string  `json:"status"`
	GitUrl           string  `json:"git_url"`
	GitUsername      *string `json:"git_username"`
	GitPassword      *string `json:"git_password"`
	RepoPath         *string `json:"src_path"`
	TemplateId       *string `json:"template_id"`
}

func (a App) GetSlug() string {
	return fmt.Sprintf("%s-%s", a.ProjectName, a.Name)
}

func (a App) GetDomain() string {
	return fmt.Sprintf("%s.%s", a.GetSlug(), domain)
}

func (a App) GetUrl() string {
	s := ""
	if ssl {
		s = "s"
	}
	return fmt.Sprintf("http%s://%s", s, a.GetDomain())
}

func (a *App) Deploy() error {
	templateId, err := a.suggestBuildTemplate(*a.RepoPath)
	if err != nil {
		return err
	}
	a.TemplateId = ptr(templateId)

	err = a.cloneRepo()
	if err != nil {
		return err
	}

	artifactDir, err := a.Build()
	if err != nil {
		return err
	}

	err = a.Run(artifactDir)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) Build() (artifactDir string, err error) {
	// Create tmp mount dir
	buildDir, err := os.MkdirTemp("./mounts/build", "")
	if err != nil {
		return
	}
	defer os.RemoveAll(buildDir)

	// Copy source code into container
	err = cp.Copy(*a.RepoPath, buildDir)
	if err != nil {
		return
	}

	template := deploymentTemplates[*a.TemplateId]

	// Write build script into container
	beforeScript := ""
	if template.Build.BeforeScript != nil {
		beforeScript = *template.Build.BeforeScript
	}
	afterScript := ""
	if template.Build.AfterScript != nil {
		afterScript = *template.Build.AfterScript
	}

	buildScript := fmt.Sprintf(
		"#!/bin/sh\n\ncd /runner/\n\n#Before Script:\n%s\n#Run Command:\n%s\n#After Script:\n%s",
		beforeScript,
		template.Build.Cmd,
		afterScript,
	)
	err = os.WriteFile(path.Join(buildDir, "r_build.sh"), []byte(buildScript), 0755)

	// Start container
	containerId, err := dockerRun(
		template.Build.Image,
		"/runner/r_build.sh",
		nil,
		nil,
		nil,
		buildDir,
	)
	if err != nil {
		return
	}
	a.BuildContainerId = ptr(containerId)

	// Watch container
	eChan, errChan := docker.Events(context.Background(), types.EventsOptions{})

LOOP:
	for {
		select {
		case err := <-errChan:
			log.Fatal(err)
		case msg := <-eChan:
			if msg.Type == "container" {
				if msg.Action == "die" {
					log.Println("Container died:", msg.Actor.ID)
					break LOOP
				}
			}
		}
	}

	// Save artifact
	artifactDir, err = os.MkdirTemp("./artifacts", "")
	if err != nil {
		return
	}

	err = cp.Copy(
		path.Join(buildDir, template.Build.Artifact),
		path.Join(artifactDir, template.Build.Artifact),
	)
	if err != nil {
		return
	}

	return
}

func (a *App) Run(artifactDir string) (err error) {
	template := deploymentTemplates[*a.TemplateId]

	// Select random host port for container
	port, err := getFreePort()
	if err != nil {
		return
	}
	a.Port = ptr(strconv.Itoa(port))

	// Create tmp mount dir
	workDir, err := os.MkdirTemp("./mounts/running", "")
	if err != nil {
		return
	}

	// Copy artifacts into workDir
	err = cp.Copy(artifactDir, workDir)
	if err != nil {
		return
	}

	// Write run script into container
	beforeScript := ""
	if template.Run.BeforeScript != nil {
		beforeScript = *template.Run.BeforeScript
	}
	afterScript := ""
	if template.Run.AfterScript != nil {
		afterScript = *template.Run.AfterScript
	}

	runScript := fmt.Sprintf(
		"#!/bin/sh\n\ncd /runner/\n\n#Before Script:\n%s\n#Run Command:\n%s\n#After Script:\n%s",
		beforeScript,
		template.Run.Cmd,
		afterScript,
	)
	err = os.WriteFile(path.Join(workDir, "r_run.sh"), []byte(runScript), 0755)

	// Start container
	containerId, err := dockerRun(
		template.Run.Image,
		"/runner/r_run.sh",
		nil,
		ptr(template.Run.Port),
		ptr(strconv.Itoa(port)),
		workDir,
	)
	if err != nil {
		return
	}
	a.ContainerId = ptr(containerId)

	return
}

func (a *App) cloneRepo() error {
	tmpDir, err := os.MkdirTemp("./repos", "")
	if err != nil {
		return err
	}
	a.RepoPath = ptr(tmpDir)

	options := git.CloneOptions{
		URL: a.GitUrl,
	}
	if a.GitUsername != nil && a.GitPassword != nil {
		options.Auth = &http.BasicAuth{
			Username: *a.GitUsername,
			Password: *a.GitPassword,
		}
	}
	_, err = git.PlainClone(*a.RepoPath, false, &options)
	if err != nil {
		return err
	}

	return nil
}

func (a App) suggestBuildTemplate(path string) (templateId string, err error) {
	// Select deployment template based on project.json
	var pkgJson map[string]interface{}
	pkgJson, err = loadPackageJSON(path)
	if err != nil {
		return
	}
	deps, err := parseDependencies(pkgJson)
	if err != nil {
		return
	}
	templateId, err = findTemplateByDependencies(deps)
	if err != nil {
		return
	}

	return templateId, nil
}
