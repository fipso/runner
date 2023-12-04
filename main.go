package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
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
	data, err := json.Marshal(apps)
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
			return c.Redirect("/runner#/deployments/" + deployment.Id + "/logs/build")
		}

		err := proxy.Do(c, fmt.Sprintf("http://127.0.0.1:%s", *deployment.Port))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		return c.Next()
	})

	// Serve Vue Frontend
	app.Static("/runner", "./www/dist")

	app.Get("/runner/api/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
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
			Id:          makeId(),
			Name:        body.Name,
			TemplateId:  &body.TemplateId,
			GitUrl:      body.GitUrl,
			GitUsername: body.GitUsername,
			GitPassword: body.GitPassword,
			Env:         ptr(body.Env),
		}

		apps = append(apps, &app)

		return c.JSON(app)
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

	app.Get("/runner/api/deployments/:id/logs/:logType", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid deployment id")
		}

		logType := c.Params("logType", "")
		if logType != "build" && logType != "running" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid log type")
		}

		deployment := getDeploymentById(id)
		if deployment == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Unkown deployment id")
		}

		if deployment.BuildJob == nil {
			return fiber.NewError(fiber.StatusBadRequest, "No build job found")
		}

		var err error
		var logs string

		if logType == "build" {
			logs, err = deployment.BuildJob.GetLogs()
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
		}
		if logType == "running" {
			logs, err = deployment.GetLogs()
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
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

	app.Get("/runner/*", func(ctx *fiber.Ctx) error {
		return ctx.SendFile("./www/dist/index.html", false)
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
