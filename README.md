# Runner
Zero config vercel like preview deployments using docker

Status: In early development. Deployment works kinda but WebUI is still very WIP

### Features
- [ ] Data persistence
  - Currently all deployments/apps are only stored in memory
- Web UI
  - [ ] Connect with git providers via OAuth
  - [ ] Configure deployments
  - [ ] Show build/runtime logs
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
- Access the Web UI

**Headless HTTP API Examples**:  
`curl http://127.0.0.1:1337/runner/api/app -H "Content-Type: application/json" -d '{"git_url": "https://github.com/fipso/nextjs-standalone-example.git", "name": "test", "template_id": "nextjs", "env": ""}'`  
  
`curl http://127.0.0.1:1337/runner/api/app/zIMkEZvZgeyw/deploy -H "Content-Type: application/json" -d '{"branch": "main", "commit": "ef2b6e795558cb29d89c87016e930c5a1c1974f2"}'`  

