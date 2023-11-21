package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	cp "github.com/otiai10/copy"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/acme/autocert"
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
	AfterScript  *string `toml:"after_script,omitempty"`
}

type StepBuild struct {
	BuildStep
	Artifact string `toml:"artifact"`
}

type StepRun struct {
	BuildStep
	Port string `toml:"port"`
}

type App struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	ContainerId string `json:"container_id"`
	Port        string `json:"port"`
}

func (a App) GetDomain() string {
	return fmt.Sprintf("%s.%s", a.Name, domain)
}

func (a App) GetUrl() string {
	s := ""
	if ssl {
		s = "s"
	}
	return fmt.Sprintf("http%s://%s.%s", s, a.Name, domain)
}

// Globals
var deploymentTemplates map[string]TemplateConfig
var deployments []App

// CLI Flags
var domain string
var ssl bool
var debug bool
var port string
var sslPort string

func main() {
	// Handle CLI
	flag.StringVar(&domain, "domain", "", "Base domain for all deployments and UI")
	flag.BoolVar(&ssl, "ssl", false, "Enable SSL")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.StringVar(&port, "port", "80", "Port for HTTP")
	flag.StringVar(&sslPort, "ssl-port", "443", "Port for HTTPS")

	flag.Parse()

	// Init
	loadTemplates()
	connectDocker()

	// Create required tmp folder structure
	if err := createDirIfNotExists("./mounts"); err != nil {
		log.Fatal(err)
	}
	if err := createDirIfNotExists("./mounts/build"); err != nil {
		log.Fatal(err)
	}
	if err := createDirIfNotExists("./mounts/running"); err != nil {
		log.Fatal(err)
	}
	if err := createDirIfNotExists("./artifacts"); err != nil {
		log.Fatal(err)
	}

	// Initialize web server
	proxy.WithClient(&fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		DisablePathNormalizing:   true,
	})

	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		app := getAppByDomain(c.Hostname())
		if app == nil {
			return fiber.NewError(fiber.StatusNotFound, "App not found")
		}

		return proxy.Do(c, fmt.Sprintf("http://127.0.0.1:%s", app.Port))
	})

	if ssl {
		// Certificate manager
		m := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			// Replace with your domain
			HostPolicy: autocert.HostWhitelist(fmt.Sprintf("*.%s", domain)),
			// Folder to store the certificates
			Cache: autocert.DirCache("./certs"),
		}

		// TLS Config
		cfg := &tls.Config{
			// Get Certificate from Let's Encrypt
			GetCertificate: m.GetCertificate,
			// By default NextProtos contains the "h2"
			// This has to be removed since Fasthttp does not support HTTP/2
			// Or it will cause a flood of PRI method logs
			// http://webconcepts.info/concepts/http-method/PRI
			NextProtos: []string{
				"http/1.1", "acme-tls/1",
			},
		}
		ln, err := tls.Listen("tcp", fmt.Sprintf(":%s", sslPort), cfg)
		if err != nil {
			panic(err)
		}

		// Start server
		log.Fatal(app.Listener(ln))
	} else {
		log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
	}

}

func deployApp(path string, env string) {
	_, templateId, artifactDir, err := buildApp(path, env)
	if err != nil {
		log.Fatal(err)
	}

	contaierId, port, err := runApp(templateId, artifactDir)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(contaierId, port)
}

func buildApp(
	srcPath, env string,
) (containerId string, templateId string, artifactDir string, err error) {
	// Select deployment template based on project.json
	var pkgJson map[string]interface{}
	pkgJson, err = loadPackageJSON(srcPath)
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
	template := deploymentTemplates[templateId]

	// Create tmp mount dir
	buildDir, err := os.MkdirTemp("./mounts/build", "")
	if err != nil {
		return
	}
	defer os.RemoveAll(buildDir)

	log.Println("Build Dir Mount:", buildDir)

	// Copy source code into container
	err = cp.Copy(srcPath, buildDir)
	if err != nil {
		return
	}

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
	containerId, err = dockerRun(
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

	// Log
	log.Println(dockerLogs(containerId))

	// Save artifact
	artifactDir, err = os.MkdirTemp("./artifacts", "")
	if err != nil {
		return
	}

	log.Println("Artifact Dir:", artifactDir)

	err = cp.Copy(
		path.Join(buildDir, template.Build.Artifact),
		path.Join(artifactDir, template.Build.Artifact),
	)
	if err != nil {
		return
	}

	return
}

func runApp(templateId, artifactDir string) (containerId string, port int, err error) {
	template := deploymentTemplates[templateId]

	// Select random host port for container
	port, err = getFreePort()
	if err != nil {
		return
	}
	log.Println("Random port", port)

	// Create tmp mount dir
	workDir, err := os.MkdirTemp("./mounts/running", "")
	if err != nil {
		return
	}

	log.Println("Run Dir Mount:", workDir)

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
	containerId, err = dockerRun(
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

	return
}
