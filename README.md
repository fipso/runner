# Runner

![image](https://github.com/fipso/runner/assets/8930842/3846e95e-3bc6-4a11-8959-59a72ad2ef13)

Zero config vercel like preview deployments using docker

Status: Alpha ~ Should work

### Features
- Web UI
  - [ ] Connect with git providers via OAuth
  - [x] Add and configure deployments
  - [x] Show build/runtime/request logs
- Github and Gitlab Webhook integration 
  - [x] Handle push event
- Fast builds using docker
- Comes with ready to use build templates:
  - [x] NextJS
  - [ ] Vite
  - [ ] React
  - [ ] Static
- [x] Templates are easy to modify using .toml files
- [x] Automatic SSL using Let's Encrypt ACME
- [ ] SSH directly into container
- [ ] Detect package manager from package.json
- [ ] Authentication
- [ ] HTTP API

### Stuff we dont care about for now
- Scalability
  - This tool is only for preview deployments
  - Prod maybe soon
- Security
  - Only deploy code you trust as docker containers provide no safety

### Usage
- Using docker:
    - `docker run -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/fipso/runner:main`
- Using a binary:
    - Download a runner build artifact
    - ./runner -d mydomain.com

### Dev Usage
- Start dev backend:
    - `go mod tidy`
    - `go build`
    - `sudo ./runner -domain site1.local -port 1337`
    - For air users (scuffed run as root workarround):
        - `SUDO_PW=<your sudo pw> air`
- Start dev frontend
    - `cd www`
    - `bun install`
    - `bun run dev`
- Access the Web UI on :3000
