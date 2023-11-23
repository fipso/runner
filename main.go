package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"

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
var apps []App

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
		// Skip API routes
		if bytes.HasPrefix(c.Request().URI().Path(), []byte("/api")) {
			return c.Next()
		}

		// Forward to docker container
		deployment := getDeploymentByDomain(c.Hostname())
		if deployment == nil {
			return fiber.NewError(fiber.StatusNotFound, "Deployment not found")
		}

		err := proxy.Do(c, fmt.Sprintf("http://127.0.0.1:%s", *deployment.Port))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		return c.Next()
	})

	app.Post("/api/app", func(c *fiber.Ctx) error {
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

		app := App{
			Name:        body.Name,
			TemplateId:  &body.TemplateId,
			GitUrl:      body.GitUrl,
			GitUsername: body.GitUsername,
			GitPassword: body.GitPassword,
			Env:         ptr(body.Env),
		}

		return c.JSON(app)
	})

	app.Post("/api/app/:id/deploy", func(c *fiber.Ctx) error {
		id := c.Params("id", "")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid app id")
		}

		var body struct {
			Branch string `json:"branch"`
			Commit string `json:"commit"`
		}

		err := json.Unmarshal(c.Body(), &body)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
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
