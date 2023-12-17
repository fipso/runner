package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type App struct {
	Id            string        `json:"id"`
	Name          string        `json:"name"`
	Port          *string       `json:"port"`
	Env           *string       `json:"env"`
	GitUrl        string        `json:"git_url"`
	GitUsername   *string       `json:"git_username"`
	GitPassword   *string       `json:"git_password"`
	TemplateId    *string       `json:"template_id"`
	Deployments   []*Deployment `json:"deployments"`
	WebhookSecret string        `json:"webhook_secret"`
}

func (a *App) Deploy(gitBranch, gitCommit string) (deployment *Deployment, err error) {

	log.Println(
		"[Deployment] Deploying branch:",
		gitBranch,
		"commit:",
		gitCommit,
		"for app:",
		a.Name,
		"with id:",
		a.Id,
	)

	// templateId, err := a.suggestBuildTemplate(*a.RepoPath)
	// if err != nil {
	// 	return err
	// }
	// a.TemplateId = ptr(templateId)
	deployment = &Deployment{
		Id:              makeId(),
		Time:            time.Now(),
		App:             a,
		GitBranch:       gitBranch,
		GitCommit:       gitCommit,
		Status:          "Initializing Build",
		RequestsLogLock: &sync.Mutex{},
	}

	buildJob := &BuildJob{
		Id:         makeId(),
		Deployment: deployment,
		Status:     "Building",
	}
	deployment.BuildJob = buildJob

	a.Deployments = append(a.Deployments, deployment)
	writeConfig()

	// Build and deploy in background
	go func(buildJob *BuildJob, deployment *Deployment) {
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
	}(buildJob, deployment)

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

func (a *App) GetWebhookUrl() string {
	s := ""
	p := ""
	if ssl {
		s = "s"
		if sslPort != "443" {
			p = fmt.Sprintf(":%s", sslPort)
		}
	} else {
		if port != "80" {
			p = fmt.Sprintf(":%s", port)
		}

	}
	return fmt.Sprintf("http%s://%s%s/runner/api/app/%s/webhook/", s, domain, p, a.Id)
}

func (a *App) MarshalJSON() ([]byte, error) {
	type Alias App

	return json.Marshal(struct {
		*Alias
		WebhookUrl string `json:"webhook_url"`
	}{
		Alias:      (*Alias)(a),
		WebhookUrl: a.GetWebhookUrl(),
	})
}

func (a *App) GetSlug() string {
	slug := a.Name
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")

	return slug
}
