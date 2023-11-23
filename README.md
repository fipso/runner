# Runner
Zero config vercel like preview deployments using docker

Status: In early development. Only HTTP API works yet.

### Features
- Web UI
  - Connect with git providers via OAuth
  - Configure deployments
- Fast builds using docker
- Comes with ready to use build templates:
  - [x] NextJS
  - [ ] Vite
  - [ ] React
  - [ ] Static
- Templates are easy to modify using .toml files
- [ ] Automatic SSL using Let's Encrypt ACME

### Usage

HTTP API Examples:

`curl http://127.0.0.1:1337/api/app -H "Content-Type: application/json" -d '{"git_url": "https://github.com/fipso/nextjs-standalone-example.git", "name": "test", "template_id": "nextjs", "env": ""}'`

`curl http://127.0.0.1:1337/api/app/zIMkEZvZgeyw/deploy -H "Content-Type: application/json" -d '{"branch": "main", "commit": "ef2b6e795558cb29d89c87016e930c5a1c1974f2"}'`
