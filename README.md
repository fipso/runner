# Runner

![image](https://github.com/fipso/runner/assets/8930842/3846e95e-3bc6-4a11-8959-59a72ad2ef13)


Zero config vercel like preview deployments using docker

Status: In early development

### Features
- Web UI
  - [ ] Connect with git providers via OAuth
  - [x] Configure deployments
  - [x] Show build/runtime logs
- Github and Gitlab Webhook integration 
- Fast builds using docker
- Comes with ready to use build templates:
  - [x] NextJS
  - [ ] Vite
  - [ ] React
  - [ ] Static
- [x] Templates are easy to modify using .toml files
- [x] Automatic SSL using Let's Encrypt ACME
- [ ] SSH directly into container

### Stuff we dont care about for now
- Scalability
  - This tool is only for preview deployments
  - Prod maybe soon
- Security
  - Only deploy code you trust
  - Docker containers provide no safety

### Usage
- TODO: Add docker file

### Dev Usage
- Start dev backend:
  - `go mod tidy`
  - `go build`
  - `sudo ./runner -domain site1.local -port 1337`
- Start dev frontend
 - `cd www`
 - `bun install`
 - `bun run dev`
- Access the Web UI on :3000
