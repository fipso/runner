package main

import (
	"fmt"
	"log"
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
		Status:    "Initializing Build",
	}

	buildJob := &BuildJob{
		Id:         makeId(),
		Deployment: deployment,
		Status:     "Building",
	}
	deployment.BuildJob = buildJob

	a.Deployments = append(a.Deployments, *deployment)
	writeConfig()

	// Build and deploy in background
	go func() {
		err = buildJob.Run()
		if err != nil {
			log.Println("[Build Job]", err)
		}

		deployment.Status = fmt.Sprintf("Build: %s", buildJob.Status)

		err = deployment.Run()
		if err != nil {
			log.Println("[Build Job]", err)
		}

		writeConfig()
	}()

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
