package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	cp "github.com/otiai10/copy"
)

type Deployment struct {
	Id              string      `json:"id"`
	Time            time.Time   `json:"time"`
	ContainerId     *string     `json:"container_id"`
	GitBranch       string      `json:"git_branch"`
	GitCommit       string      `json:"git_commit"`
	Status          string      `json:"status"`
	Port            *string     `json:"port"`
	BuildJob        *BuildJob   `json:"build_job"`
	RequestsLog     []string    `json:"-"`
	RequestsLogLock *sync.Mutex `json:"-"`
	App             *App        `json:"-"`
}

func (d Deployment) GetSlug() string {
	return fmt.Sprintf("%s-%s-%s", d.App.GetSlug(), d.GitBranch, d.GitCommit[:7])
}

func (d Deployment) GetDomain() string {
	return fmt.Sprintf("%s.%s", d.GetSlug(), domain)
}

func (d Deployment) GetUrl() string {
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
	return fmt.Sprintf("http%s://%s%s", s, d.GetDomain(), p)
}

func (d Deployment) GetName() string {
	short := d.GitCommit[:7]
	return fmt.Sprintf("%s/%s", d.GitBranch, short)
}

func (d *Deployment) MarshalJSON() ([]byte, error) {
	type Alias Deployment

	return json.Marshal(struct {
		*Alias
		Name string `json:"name"`
		Url  string `json:"url"`
	}{
		Alias: (*Alias)(d),
		Name:  d.GetName(),
		Url:   d.GetUrl(),
	})
}

func (d *Deployment) Run() (err error) {
	// Update status
	defer func() {
		if err != nil {
			d.Status = "Failed"
		} else {
			d.Status = "Running"
		}

		writeConfig()
	}()

	template := deploymentTemplates[*d.App.TemplateId]

	// Select random host port for container
	port, err := getFreePort()
	if err != nil {
		return
	}
	d.Port = ptr(strconv.Itoa(port))

	// Create tmp mount dir
	workDir, err := os.MkdirTemp("./mounts/running", "")
	if err != nil {
		return
	}

	// Copy artifacts into workDir
	err = cp.Copy(d.BuildJob.ArtifactsPath, workDir)
	if err != nil {
		return
	}

	// Write run script into container
	runScript := fmt.Sprintf(
		"#!/bin/sh\n\ncd /runner/\n\n%s",
		template.Run.Script,
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
	d.ContainerId = ptr(containerId)

	return
}

func (d *Deployment) GetLogs() (logs string, err error) {
	if d.ContainerId == nil {
		return "", fmt.Errorf("No container found yet")
	}
	logs, err = dockerLogs(*d.ContainerId)
	return
}
