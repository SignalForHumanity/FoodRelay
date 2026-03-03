You are operating inside the FoodRelay repository.

Read CLAUDE.md completely and follow it as the authoritative design and architecture guide.

Your job: bootstrap a COMPLETE, runnable, minimal v0 of the system while keeping the codebase extremely small and forkable.

Do NOT expand scope beyond CLAUDE.md.

--------------------------------
PRIMARY OBJECTIVE
--------------------------------

Create a fully working v0 of:

1) Local Node (Go, chi, SQLite, Twilio SMS)
2) Global Registry (Go, chi, SQLite)
3) Auto migrations on startup
4) SMS command parsing + storage + matching
5) Node heartbeat → registry
6) Minimal documentation + deploy instructions

The result must run locally with:
- go run ./cmd/registry
- go run ./cmd/node

No manual DB setup required.

--------------------------------
STRICT CONSTRAINTS
--------------------------------

Follow these exactly:

- Use Go 1.22+
- Use chi router
- Use SQLite for both node and registry
- Auto-run SQL migrations on boot
- Keep dependencies minimal
- No UI frameworks
- No heavy abstractions
- No microservices
- No Docker required for dev (but provide Dockerfiles)
- Code must be understandable by a single engineer

Privacy rules:
- Registry must NEVER store offers, needs, or private addresses
- Node stores local SMS data only
- No central logging of message contents

--------------------------------
PHASE 1 — CREATE/VERIFY CORE FILES
--------------------------------

Ensure these files exist and are complete:

go.mod  
.env.example  
README.md  
.gitignore  

cmd/node/main.go  
cmd/registry/main.go  

internal/common/*
internal/node/*
internal/registry/*

migrations/node_001.sql  
migrations/registry_001.sql  

docs/protocol.md  
docs/sms_grammar.md  
docs/privacy_security.md  
docs/ops_playbook.md  

deploy/docker/*  
deploy/systemd/*  

If any file is missing or incomplete:
→ create it

--------------------------------
PHASE 2 — IMPLEMENT NODE (LOCAL SMS ENGINE)
--------------------------------

Node must support:

Twilio webhook:
POST /twilio/sms

Commands:
OFFER
NEED
READY
HELP
UNKNOWN fallback

Flow:
OFFER → awaiting_ready
READY → ready → matching
match → create job → notify both parties

Matching rules:
- time windows overlap required
- prefer higher priority need
- deterministic simple scoring
- no ML

Scheduler loop:
- retry matching READY offers every 60s
- expire offers/needs after window_end
- escalate to admin if READY unmatched > X minutes

Must compile and run:
go run ./cmd/node

--------------------------------
PHASE 3 — IMPLEMENT REGISTRY
--------------------------------

Registry must support:

POST /v1/nodes/announce
POST /v1/nodes/heartbeat
GET  /v1/nodes

Security:
- ed25519 signed announce
- store pubkey
- verify heartbeat signature

Active node = heartbeat within 15 min

Must compile and run:
go run ./cmd/registry

--------------------------------
PHASE 4 — DOCUMENTATION
--------------------------------

Create or update:

README.md:
- project vision
- quick start (5 min setup)
- Twilio setup
- local dev instructions
- example SMS usage

docs/protocol.md:
- node ↔ registry heartbeat spec
- signing format
- JSON payloads

docs/sms_grammar.md:
- all commands
- examples
- parsing expectations

docs/privacy_security.md:
- data storage rules
- PII handling
- threat model

docs/ops_playbook.md:
- how to run a node in a city
- how to onboard donors/recipients
- how to monitor

--------------------------------
PHASE 5 — DEPLOYMENT FILES
--------------------------------

Create minimal but real:

Dockerfiles (node + registry)
docker-compose.yml
systemd service files

Must be simple and readable.

--------------------------------
PHASE 6 — TEST FLOWS
--------------------------------

After implementing, provide example curl tests:

Simulate SMS:
- OFFER
- READY
- NEED

Show expected responses.

--------------------------------
OUTPUT FORMAT
--------------------------------

When you respond:

1. List files created/updated
2. Provide full contents of new files
3. Provide run instructions
4. Provide test SMS examples
5. Keep explanations minimal and technical

Do NOT summarize architecture.
Do NOT expand scope.
Just build the working v0.

Begin now.