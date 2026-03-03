# FoodRelay Operations Playbook

## Deployment modes

FoodRelay can be run in three configurations depending on scale and redundancy needs.

### Minimal (single process, combined mode)
One process runs both the node and an embedded registry. Good for a first deployment
or a single-city pilot.

```bash
NODE_RUN_REGISTRY=true REGISTRY_ADDR=:8081 go run ./cmd/node
```

The node automatically announces to its own embedded registry. No separate registry
process needed.

### Standard (separate node + registry)
One registry process and one or more node processes on the same or different servers.

```bash
go run ./cmd/registry   # listens on :8081
go run ./cmd/node       # announces to :8081 via REGISTRY_URLS
```

### Federated (multiple registries)
Multiple registries gossip with each other. Nodes announce to all of them. Any
single registry failure does not interrupt the system.

```bash
# Registry 1
REGISTRY_ADDR=:8081 REGISTRY_DB_PATH=./reg1.db \
  REGISTRY_PEERS=http://reg2.example.com:8082 \
  go run ./cmd/registry

# Registry 2
REGISTRY_ADDR=:8082 REGISTRY_DB_PATH=./reg2.db \
  REGISTRY_PEERS=http://reg1.example.com:8081 \
  go run ./cmd/registry

# Node — announces to both; if one is down the other still receives heartbeats
REGISTRY_URLS=http://reg1.example.com:8081,http://reg2.example.com:8082 \
  go run ./cmd/node
```

Registries sync with peers every 5 minutes. A new registry only needs one peer in
`REGISTRY_PEERS`; it will pull all known nodes from that peer on first sync.

---

## Running a node in a city

### Prerequisites

1. A server (VPS, Raspberry Pi, etc.) with Go 1.22+ installed
2. A Twilio account with a phone number
3. The FoodRelay repo

### Setup steps

```bash
# 1. Clone repo
git clone https://github.com/yourorg/foodrelay.git
cd foodrelay

# 2. Create .env
cp .env.example .env
# Edit .env with your values (see key variables below)

# 3. Run dependencies
go mod tidy

# 4. Start the registry (or point REGISTRY_URLS to an existing one)
go run ./cmd/registry

# 5. Start the node (in another terminal or process)
go run ./cmd/node
```

On first run, the node prints a `NODE_PRIVATE_KEY_HEX` value.
**Copy it into your `.env` immediately** — this is the node's permanent identity.
If lost, the node will generate a new keypair on next start and must re-announce.

### Key environment variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `NODE_ID` | Unique node identifier | `node-nyc-01` |
| `NODE_PHONE` | Public SMS number for this node | `+15551234567` |
| `NODE_ADDR` | Node HTTP listen address | `:8080` |
| `REGISTRY_URLS` | Comma-separated registry URLs to announce to | `http://reg1:8081,http://reg2:8082` |
| `NODE_RUN_REGISTRY` | Start embedded registry in same process | `false` |
| `NODE_PRIVATE_KEY_HEX` | ed25519 seed (generated on first run) | *(32-byte hex)* |
| `REGISTRY_ADDR` | Registry HTTP listen address | `:8081` |
| `REGISTRY_PEERS` | Comma-separated peer registry URLs for gossip | `http://reg2:8082` |
| `ADMIN_PHONES` | Comma-separated phones for escalation alerts | `+15559876543` |
| `READY_ESCALATE_MINUTES` | Minutes before alerting admins about unmatched READY | `30` |

`REGISTRY_URL` (singular) is still accepted for backwards compatibility but
`REGISTRY_URLS` (plural, comma-separated) takes precedence.

### Twilio webhook setup

1. Go to [Twilio Console](https://console.twilio.com) → Phone Numbers → Manage → Active Numbers
2. Select your number
3. Under "Messaging", set Webhook URL to:
   `https://your-domain.com/twilio/sms`
4. Method: HTTP POST

For local dev, use [ngrok](https://ngrok.com):
```bash
ngrok http 8080
# Use the HTTPS URL as your Twilio webhook
```

---

## Onboarding donors

Send them the node's SMS number and these instructions:

> "Text OFFER to [number] to donate food. Example:
> OFFER 12 meals 6-7pm 123 Main St
> Then reply READY when food is packed."

---

## Onboarding recipients / hubs

> "Text NEED to [number] to request food. Example:
> NEED 20 meals by 7pm 55 Church St
> We'll text you when a match is found."

---

## Monitoring

### Health check
```bash
curl http://localhost:8080/health   # node
curl http://localhost:8081/health   # registry
```

### Check active nodes (registry)
```bash
curl http://localhost:8081/v1/nodes
```

### Check federation state (peer-to-peer sync)
```bash
curl http://localhost:8081/v1/federation/nodes
```

### Node map (browser)
```
http://localhost:8081/map
```

### Check logs
The node and registry log to stdout. Use journald (`journalctl -u foodrelay-node -f`)
with systemd.

### Key log lines

| Line | Meaning |
|------|---------|
| `[sms] from=+1... body="OFFER ..."` | Inbound SMS received |
| `[engine] matched offer=N need=M score=...` | Successful match |
| `[scheduler] expired N offers` | Expiration pass ran |
| `[scheduler] escalating offer=N` | Unmatched READY offer alerted to admins |
| `[admin-alert]` | Escalation SMS sent |
| `[heartbeat] /v1/nodes/announce -> http://...: ...` | Registry unreachable on announce |
| `[federation] synced http://...: N merged, M skipped` | Peer gossip completed |
| `[federation] invalid announce_sig for node ... skipping` | Peer served a bad record |

---

## Systemd deployment

```bash
# Build binaries
go build -o /opt/foodrelay/node ./cmd/node
go build -o /opt/foodrelay-registry/registry ./cmd/registry

# Copy migration files
cp -r migrations/ /opt/foodrelay/
cp -r migrations/ /opt/foodrelay-registry/

# Copy env files
cp .env /opt/foodrelay/.env
cp .env /opt/foodrelay-registry/.env

# Create user
useradd -r -s /bin/false foodrelay

# Install service files
cp deploy/systemd/foodrelay-node.service /etc/systemd/system/
cp deploy/systemd/foodrelay-registry.service /etc/systemd/system/

# Enable and start
systemctl daemon-reload
systemctl enable --now foodrelay-registry foodrelay-node
```

---

## Key management

### Rotating the admin phone number
Update `ADMIN_PHONES` in `.env` and restart the node. No DB change needed.

### Rotating the node keypair
**Avoid unless absolutely necessary.** After the first announce, the registry locks
in the public key — re-announces are verified against the stored key. To rotate:

1. Delete `NODE_PRIVATE_KEY_HEX` from `.env`
2. **Also delete the node record from every registry's DB**, or the new keypair will
   fail signature verification on re-announce:
   ```sql
   DELETE FROM nodes WHERE node_id = 'your-node-id';
   ```
3. Restart node — it generates a new keypair, prints the new seed, and re-announces.
4. Copy the printed `NODE_PRIVATE_KEY_HEX` into `.env` immediately.

### Adding a new registry to an existing federation
1. Start the new registry with at least one existing peer in `REGISTRY_PEERS`.
2. On first sync (within 5 minutes of startup), the new registry pulls all known
   nodes from its peer and verifies their announce signatures before inserting them.
3. Add the new registry's URL to `REGISTRY_URLS` in each node's `.env` and restart
   the nodes. (Nodes do not auto-discover new registries.)

---

## Backup

The node and registry each store a single SQLite file.
```bash
# Node backup
cp /opt/foodrelay/node.db /backups/node-$(date +%Y%m%d).db

# Registry backup
cp /opt/foodrelay-registry/registry.db /backups/registry-$(date +%Y%m%d).db
```

Schedule with cron:
```
0 2 * * * cp /opt/foodrelay/node.db /backups/node-$(date +\%Y\%m\%d).db
```
