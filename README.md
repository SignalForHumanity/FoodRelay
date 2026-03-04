# FoodRelay

A federated, forkable system for small-scale food rescue via SMS.

Each city runs its own **node** (Go + SQLite + Twilio). One or more **registries** maintain a minimal directory of active nodes for discovery. Registries form a mesh by gossiping with each other — no single point of failure. No central database of offers or needs; all sensitive data stays local to the node.

---

## Quick start (5 minutes)

### Prerequisites
- Go 1.22+
- A [Twilio](https://twilio.com) account with a phone number

```bash
git clone https://github.com/yourorg/foodrelay.git
cd foodrelay

go mod tidy
cp .env.example .env
# Edit .env — at minimum set TWILIO_* and NODE_PHONE
```

**Option A — single process (combined mode):**
```bash
# Node + embedded registry in one process
NODE_RUN_REGISTRY=true go run ./cmd/node
```

**Option B — separate processes:**
```bash
# Terminal 1: registry
go run ./cmd/registry

# Terminal 2: node
go run ./cmd/node
```

On first node startup, it prints a key — **copy it into `.env` as `NODE_PRIVATE_KEY_HEX`** before restarting.

---

## Twilio setup

1. Buy a Twilio phone number
2. Set `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_PHONE` in `.env`
3. In the Twilio Console → Phone Number → Messaging Webhook:
   - URL: `https://your-domain.com/twilio/sms`
   - Method: POST
4. For local dev, expose your port with [ngrok](https://ngrok.com): `ngrok http 8080`

---

## Example SMS usage

**Donor flow:**
```
→ OFFER 3 trays of pasta until 8pm 123 Main St
← Got it! Offer recorded: 3 trays of pasta at 123 Main St (until 8:00 PM).
  Reply READY when it's packed and ready for pickup.

→ READY
← Your offer is marked READY. We'll notify anyone looking for food.
  [match found — donor also receives:]
← MATCH! Someone is on their way to pick up at: 123 Main St.
```

**Recipient flow:**
```
→ NEED 55 Church St
  [READY offer exists — immediate reply:]
← Food available: 3 trays of pasta at 123 Main St. Pickup before 8:00 PM. Head there now!

  [no READY offer yet:]
← Got it! We'll text you when food is available nearby.
  [when donor goes READY later, recipient receives:]
← Food available: 3 trays of pasta at 123 Main St. Pickup before 8:00 PM. Head there now!
```

Time windows that have already passed when the SMS arrives are automatically
rolled forward 24 hours — texting `until 9pm` at 10 PM registers for tomorrow.

---

## Local dev test (no Twilio needed)

```bash
# Simulate OFFER
curl -X POST http://localhost:8080/twilio/sms \
  -d "From=%2B15551230001" \
  -d "Body=OFFER+3+trays+of+pasta+until+8pm+123+Main+St"

# Simulate READY
curl -X POST http://localhost:8080/twilio/sms \
  -d "From=%2B15551230001" \
  -d "Body=READY"

# Simulate NEED (with optional address)
curl -X POST http://localhost:8080/twilio/sms \
  -d "From=%2B15551230002" \
  -d "Body=NEED+55+Church+St"

# Simulate NEED (no address)
curl -X POST http://localhost:8080/twilio/sms \
  -d "From=%2B15551230003" \
  -d "Body=NEED"

# Check active nodes
curl http://localhost:8081/v1/nodes

# Health
curl http://localhost:8080/health
curl http://localhost:8081/health
```

---

## Architecture

```
SMS → Twilio → Node (POST /twilio/sms)
                 ├── parse command
                 ├── store in SQLite
                 ├── match engine (OFFER+NEED)
                 └── reply via Twilio SMS

Node ──announce/heartbeat──► Registry A ◄──gossip──► Registry B
                              Registry B ◄──gossip──► Registry C
                                    └── GET /v1/nodes (public discovery)
```

- **Node** — Go HTTP server, chi router, SQLite, Twilio webhook. One per city.
- **Registry** — Go HTTP server, chi router, SQLite, ed25519 verification. Can be shared across cities or embedded inside a node process (`NODE_RUN_REGISTRY=true`).
- **Federation** — Registries pull from each other every 5 minutes via `/v1/federation/nodes`. Each record is verified with an ed25519 signature before being stored. A new registry only needs one known peer to bootstrap the full node list.
- **No central offer/need data** — the registry stores only node metadata (phone, geo, capabilities).

---

## Federation (running multiple registries)

```bash
# Registry 1
REGISTRY_ADDR=:8081 REGISTRY_DB_PATH=./reg1.db \
  REGISTRY_PEERS=http://reg2.example.com:8082 \
  go run ./cmd/registry

# Registry 2
REGISTRY_ADDR=:8082 REGISTRY_DB_PATH=./reg2.db \
  REGISTRY_PEERS=http://reg1.example.com:8081 \
  go run ./cmd/registry

# Node — announces to both
REGISTRY_URLS=http://reg1.example.com:8081,http://reg2.example.com:8082 \
  go run ./cmd/node
```

If one registry goes down, the node continues heartbeating to the others and peer
registries continue syncing among themselves.

---

## Configuration

See `.env.example` for all options. Key variables:

| Variable | Description |
|----------|-------------|
| `NODE_ID` | Unique identifier for this node |
| `NODE_PHONE` | The Twilio number this node uses |
| `NODE_ADDR` | Node HTTP listen address (default `:8080`) |
| `TWILIO_ACCOUNT_SID` | Twilio credentials |
| `TWILIO_AUTH_TOKEN` | Twilio credentials |
| `TWILIO_FROM_PHONE` | Outbound SMS number |
| `REGISTRY_URLS` | Comma-separated registry URLs to announce to |
| `NODE_RUN_REGISTRY` | `true` to start an embedded registry in the node process |
| `NODE_PRIVATE_KEY_HEX` | ed25519 seed — generated and printed on first run |
| `ADMIN_PHONES` | Comma-separated phones for escalation alerts |
| `READY_ESCALATE_MINUTES` | Minutes before alerting admin (default: 30) |
| `REGISTRY_ADDR` | Registry HTTP listen address (default `:8081`) |
| `REGISTRY_PEERS` | Comma-separated peer registry URLs for gossip federation |

---

## Docs

- [docs/sms_grammar.md](docs/sms_grammar.md) — all SMS commands and examples
- [docs/protocol.md](docs/protocol.md) — node↔registry API spec and federation protocol
- [docs/privacy_security.md](docs/privacy_security.md) — data rules and threat model
- [docs/ops_playbook.md](docs/ops_playbook.md) — deployment, operations, and key management
- [SECURITY.md](SECURITY.md) — security findings log

---

## Deployment

**Docker:**
```bash
cd deploy/docker
docker compose up --build
```

**Systemd:** see [docs/ops_playbook.md](docs/ops_playbook.md).

---

## License

MIT
