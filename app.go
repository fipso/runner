package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	cp "github.com/otiai10/copy"
)

type App struct {
	Id          string       `json:"id"`
	Name        string       `json:"name"`
	Port        *string      `json:"port"`
	Env         *string      `json:"env"`
	GitUrl      string       `json:"git_url"`
	GitUsername *string      `json:"git_username"`
	GitPassword *string      `json:"git_password"`
	TemplateId  *string      `json:"template_id"`
	Deployments []Deployment `json:"deployments"`
}

func (a *App) Deploy(gitBranch, gitCommit string) (deployment *Deployment, err error) {
	// templateId, err := a.suggestBuildTemplate(*a.RepoPath)
	// if err != nil {
	// 	return err
	// }
	// a.TemplateId = ptr(templateId)
	deployment = &Deployment{
		Id:        makeId(),
		App:       a,
		GitBranch: gitBranch,
		GitCommit: gitCommit,
		Status:    "Intializing Build",
	}

	buildJob := BuildJob{
		Id:         makeId(),
		Deployment: deployment,
		Status:     "Building",
	}
	deployment.BuildJob = &buildJob

	err = buildJob.Run()
	if err != nil {
		return nil, err
	}

	deployment.Status = fmt.Sprintf("Build: %s", buildJob.Status)

	err = deployment.Run()
	if err != nil {
		return nil, err
	}

	a.Deployments = append(a.Deployments, *deployment)

	return
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

func (b *BuildJob) Run() (err error) {
	// Update build job status
	defer func() {
		if err != nil {
			b.Status = "Failed"
		} else {
			b.Status = "Success"
		}
	}()

	// Create tmp mount dir
	buildDir, err := os.MkdirTemp("./mounts/build", "")
	if err != nil {
		return
	}
	defer os.RemoveAll(buildDir)

	// Clone src into buildDir
	err = b.cloneRepo(buildDir)
	if err != nil {
		return
	}

	template := deploymentTemplates[*b.Deployment.App.TemplateId]

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
	b.ContainerId = ptr(containerId)

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
	artifactDir, err := os.MkdirTemp("./artifacts", "")
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

	b.ArtifactsPath = artifactDir

	return
}

func (b *BuildJob) GetLogs() (logs string, err error) {
	logs, err = dockerLogs(*b.ContainerId)
	return
}

func (b *BuildJob) cloneRepo(path string) error {
	branchRef := plumbing.ReferenceName(
		fmt.Sprintf("refs/heads/%s", b.Deployment.GitBranch),
	)

	options := git.CloneOptions{
		URL:           b.Deployment.App.GitUrl,
		SingleBranch:  true,
		ReferenceName: branchRef,
	}
	if b.Deployment.App.GitUsername != nil && b.Deployment.App.GitPassword != nil {
		options.Auth = &http.BasicAuth{
			Username: *b.Deployment.App.GitUsername,
			Password: *b.Deployment.App.GitPassword,
		}
	}
	repo, err := git.PlainClone(path, false, &options)
	if err != nil {
		return err
	}

	head, err := repo.Head()
	if err != nil {
		return err
	}

	if head.Hash().String() != b.Deployment.GitCommit {
		w, err := repo.Worktree()
		if err != nil {
			return err
		}

		err = repo.Fetch(&git.FetchOptions{
			RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
		})
		if err != nil {
			return err
		}

		err = w.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
			Hash:   plumbing.NewHash(b.Deployment.GitCommit),
			Force:  true,
		})
		if err != nil {
			return err
		}

	}

	return nil
}
