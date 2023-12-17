package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/go-playground/webhooks/v6/gitlab"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/samber/lo"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"golang.org/x/crypto/acme/autocert"
)

type TemplateConfig struct {
	Name              string    `toml:"name"               json:"name"`
	MatchDependencies []string  `toml:"match_dependencies" json:"match_dependencies"`
	Info              string    `toml:"info"               json:"info"`
	Build             StepBuild `toml:"build"              json:"build"`
	Run               StepRun   `toml:"run"                json:"run"`
}

type DeployStep struct {
	Image  string `toml:"image"  json:"image"`
	Script string `toml:"script" json:"script"`
}

type StepBuild struct {
	DeployStep
	Artifact string `toml:"artifact"`
}

type StepRun struct {
	DeployStep
	Port string `toml:"port"`
}

// Globals
var deploymentTemplates map[string]TemplateConfig
var apps []*App

// CLI Flags
var domain string
var ssl bool
var debug bool
var port string
var sslPort string

func writeConfig() {
	data, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("./apps.json", data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Save config on exit
	defer func() {
		writeConfig()
	}()

	// Handle CLI
	flag.StringVar(&domain, "domain", "", "Base domain for all deployments and UI")
	flag.BoolVar(&ssl, "ssl", false, "Enable SSL")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.StringVar(&port, "port", "80", "Port for HTTP")
	flag.StringVar(&sslPort, "ssl-port", "443", "Port for HTTPS")

	flag.Parse()

	if domain == "" {
		log.Fatal("-domain is required")
	}

	// Load config
	if _, err := os.Stat("./apps.json"); !os.IsNotExist(err) {
		data, err := os.ReadFile("./apps.json")
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(data, &apps)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Recreate pointer references for app and deployment
	for _, app := range apps {
		for _, deployment := range app.Deployments {
			deployment.App = app
			deployment.RequestsLogLock = &sync.Mutex{}
			if deployment.BuildJob != nil {
				deployment.BuildJob.Deployment = deployment
			}
		}
	}

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
	if err := createDirIfNotExists("./repos"); err != nil {
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

	app := fiber.New(
		fiber.Config{
			DisableStartupMessage: true,
		},
	)

	app.Use(func(c *fiber.Ctx) error {
		// Skip Frontend and API routes
		if bytes.HasPrefix(c.Request().URI().Path(), []byte("/runner")) {
			return c.Next()
		}

		// Find deployment by domain
		deployment := getDeploymentByDomain(strings.Split(c.Hostname(), ":")[0]) // Remove port
		if deployment == nil {
			return c.Redirect("/runner")
			//return fiber.NewError(fiber.StatusNotFound, "Deployment not found")
		}

		if deployment.Status != "Running" {
			return c.Redirect("/runner/deployment/" + deployment.Id + "/logs/build")
		}

		// Rewrite request host
		url := c.Request().URI()
		url.SetHost(fmt.Sprintf("127.0.0.1:%s", *deployment.Port))

		err := proxy.Do(c, url.String())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		resp := c.Response()
		deployment.RequestsLogLock.Lock()
		deployment.RequestsLog = append(
			deployment.RequestsLog,
			fmt.Sprintf("%s %s %d", c.Method(), c.Path(), resp.StatusCode()),
		)
		deployment.RequestsLogLock.Unlock()
		//return c.Next()

		return nil
	})

	app.Get("/runner/api/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"templates": deploymentTemplates,
			// Unused for now:
			"domain":  domain,
			"ssl":     ssl,
			"debug":   debug,
			"port":    port,
			"sslPort": sslPort,
		})
	})

	app.Get("/runner/api/app", func(c *fiber.Ctx) error {
		return c.JSON(apps)
	})

	app.Get("/runner/api/app/:id", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid app id")
		}

		app := getAppById(id)
		if app == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown app id")
		}

		return c.JSON(app)
	})

	app.Post("/runner/api/app", func(c *fiber.Ctx) error {
		var body struct {
			Name        string  `json:"name"`
			TemplateId  string  `json:"template_id"`
			GitUrl      string  `json:"git_url"`
			GitUsername *string `json:"git_username,omitempty"`
			GitPassword *string `json:"git_password,omitempty"`
			Env         string  `json:"env"`
		}

		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		if body.Name == "" || body.TemplateId == "" || body.GitUrl == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Missing required fields")
		}

		app := App{
			Id:            makeId(),
			Name:          body.Name,
			TemplateId:    &body.TemplateId,
			GitUrl:        body.GitUrl,
			GitUsername:   body.GitUsername,
			GitPassword:   body.GitPassword,
			Env:           ptr(body.Env),
			WebhookSecret: makeId(),
		}

		apps = append(apps, &app)

		writeConfig()

		return c.JSON(app)
	})

	app.Post("/runner/api/app/:id/env", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid app id")
		}

		app := getAppById(id)
		if app == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown app id")
		}

		var body struct {
			Env string `json:"env"`
		}

		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		if body.Env == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Missing required fields")
		}

		app.Env = ptr(body.Env)

		writeConfig()

		return c.JSON(fiber.Map{
			"success": true,
		})
	})

	app.Delete("/runner/api/app/:id", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid app id")
		}

		app := getAppById(id)
		if app == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown app id")
		}

		// Delete app
		apps = lo.Filter(apps, func(a *App, i int) bool {
			return a.Id != id
		})

		// Delete deployments
		for _, deployment := range app.Deployments {
			if deployment.ContainerId != nil {
				dockerStop(*deployment.ContainerId)
				dockerRemove(*deployment.ContainerId)
			}
		}

		writeConfig()

		return c.JSON(fiber.Map{
			"success": true,
		})

	})

	app.Post("/runner/api/app/:id/deploy", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid app id")
		}

		var body struct {
			Branch string `json:"branch"`
			Commit string `json:"commit"`
		}

		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		if body.Branch == "" || body.Commit == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Missing required fields")
		}

		app := getAppById(id)
		if app == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown app id")
		}

		deployment, err := app.Deploy(body.Branch, body.Commit)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		return c.JSON(deployment)
	})

	app.Delete("/runner/api/deployment/:id", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid deployment id")
		}

		deployment := getDeploymentById(id)
		if deployment == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown deployment id")
		}

		// Delete deployment
		deployment.App.Deployments = lo.Filter(
			deployment.App.Deployments,
			func(d *Deployment, i int) bool {
				return d.Id != id
			},
		)

		if deployment.ContainerId != nil {
			dockerStop(*deployment.ContainerId)
			dockerRemove(*deployment.ContainerId)
		}

		writeConfig()

		return c.JSON(fiber.Map{
			"success": true,
		})

	})

	app.Get("/runner/api/deployment/:id/logs/:logType", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid deployment id")
		}

		logType := c.Params("logType", "")

		deployment := getDeploymentById(id)
		if deployment == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown deployment id")
		}

		if deployment.BuildJob == nil {
			return fiber.NewError(fiber.StatusBadRequest, "No build job found")
		}

		var err error
		var logs string

		switch logType {
		case "build":
			logs, err = deployment.BuildJob.GetLogs()
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
		case "running":
			logs, err = deployment.GetLogs()
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
		case "requests":
			logs = strings.Join(deployment.RequestsLog, "\n")
		default:
			return fiber.NewError(fiber.StatusBadRequest, "Invalid log type")
		}

		var deploymentUrl string
		if deployment.Status == "Running" {
			deploymentUrl = deployment.GetUrl()
		}
		return c.JSON(fiber.Map{
			"logs":         logs,
			"build_status": deployment.BuildJob.Status,
			"url":          deploymentUrl,
		})
	})

	app.Post("/runner/api/app/:id/webhook/:provider", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid app id")
		}

		app := getAppById(id)
		if app == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown app id")
		}

		provider := c.Params("provider", "")

		var r http.Request
		err := fasthttpadaptor.ConvertRequest(c.Context(), &r, true)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		var commit, branch string

		switch provider {
		case "github":
			githubHook, _ := github.New(github.Options.Secret(app.WebhookSecret))
			payload, err := githubHook.Parse(&r, github.PushEvent)
			if err != nil {
				if err == github.ErrEventNotFound {
					return fiber.NewError(fiber.StatusBadRequest, "Invalid event")
				} else {
					return fiber.NewError(fiber.StatusBadRequest, err.Error())
				}
			}
			switch payload.(type) {

			case github.PushPayload:
				push := payload.(github.PushPayload)
				commit = push.After
				branch = plumbing.ReferenceName(push.Ref).Short()
			}

		case "gitlab":
			gitlabHook, _ := gitlab.New(gitlab.Options.Secret(app.WebhookSecret))
			payload, err := gitlabHook.Parse(&r, gitlab.PushEvents)
			if err != nil {
				if err == gitlab.ErrEventNotFound {
					return fiber.NewError(fiber.StatusBadRequest, "Invalid event")
				} else {
					return fiber.NewError(fiber.StatusBadRequest, err.Error())
				}
			}
			switch payload.(type) {
			case gitlab.PushEventPayload:
				push := payload.(gitlab.PushEventPayload)
				commit = push.After
				branch = plumbing.ReferenceName(push.Ref).Short()
			}

		default:
			return fiber.NewError(fiber.StatusBadRequest, "Invalid provider type")
		}

		go func() {
			_, err := app.Deploy(branch, commit)
			if err != nil {
				log.Println(err)
			}
		}()

		return c.JSON(fiber.Map{
			"success": true,
		})

	})

	// Serve Vue Frontend
	app.Static("/runner", "./www/dist/")

	app.Get("/runner/*", func(ctx *fiber.Ctx) error {
		return ctx.SendFile("./www/dist/index.html", true)
	})

	if ssl {
		go func() {
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
			log.Println("HTTPS Listening on port:", sslPort)
			log.Fatal(app.Listener(ln))
		}()
	}

	log.Println("HTTP Listening on port:", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}
