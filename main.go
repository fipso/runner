package main

import (
	"bytes"
	"crypto/tls"
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
		app := getAppByDomain(c.Hostname())
		if app == nil {
			return fiber.NewError(fiber.StatusNotFound, "App not found")
		}

		err := proxy.Do(c, fmt.Sprintf("http://127.0.0.1:%s", *app.Port))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		return c.Next()
	})

	app.Post("/api/deploy", func(c *fiber.Ctx) error {
		var body struct {
			ProjectName string `json:"project_name"`
			GitUrl      string `json:"git_url"`
			GitUsername string `json:"git_username"`
			GitPassword string `json:"git_password"`
			Env         string `json:"env"`
		}

		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		app := App{
			ProjectName: body.ProjectName,
			GitUrl:      body.GitUrl,
			GitUsername: body.GitUsername,
			GitPassword: body.GitPassword,
			Env:         ptr(body.Env),
		}

		return app.Deploy()
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
