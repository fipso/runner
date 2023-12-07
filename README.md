# Runner

![image](https://github.com/fipso/runner/assets/8930842/f6bf3655-ebd4-4640-abcd-3b59b465f87b)

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

### TODO
- [x] Make 'update env' work

### Stuff we dont care about for now
- Scalability
  - This tool is only for preview deployments
  - Prod maybe soon
- Security
  - Only deploy code you trust
  - Docker containers provide no safety

### Usage
- Access the Web UI
