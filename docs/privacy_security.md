# FoodRelay Privacy & Security

## Data storage rules

### Node (local)
The node stores:
- Donor/recipient phone numbers (required to send SMS replies)
- Offer and need records (qty, location, time window, notes, allergens)
- Job records (matched offer+need pairs)
- All data is local to the node's SQLite database

The node does **not** send offer/need content or phone numbers to the registry.

### Registry
The registry stores **only**:
- Node ID, public phone number, geo coordinates, coverage radius
- Capabilities, version
- ed25519 public key
- Last heartbeat timestamp
- Original announce signature and timestamp (for federation verification)

The registry **never** stores:
- Donor or recipient phone numbers
- Offer or need details
- Addresses beyond a node's public location
- Message contents of any kind
- Global lists of offers or needs

---

## PII handling

| Data | Stored where | Retention |
|------|-------------|-----------|
| Donor phone | Node SQLite | Plan: purge after 90 days |
| Recipient phone | Node SQLite | Plan: purge after 90 days |
| Offer/need content | Node SQLite | Plan: purge after 90 days |
| Node public phone | Registry | Until node is deregistered |
| Node geo | Registry | Until node is deregistered |

**Future work:** Auto-purge PII older than 30–90 days via a scheduled job on the node.

---

## Signature scheme

### Announce
```
message   = node_id + "|" + public_phone + "|" + timestamp
signature = ed25519.Sign(privateKey, []byte(message))
```
The phone number is included in the signed message. This means that when the record
propagates through federation, any peer registry can verify that the phone number
was explicitly authorised by the node's private key — a compromised peer cannot
substitute a different number.

### Heartbeat
```
message   = node_id + "|" + timestamp
signature = ed25519.Sign(privateKey, []byte(message))
```
Heartbeats do not re-transmit the phone number; they are verified against the
public key already stored from the announce.

### Re-announce (key lock-in)
On first announce (TOFU), the registry accepts whatever public key is provided.
On subsequent announces, the registry verifies the signature against the **stored**
public key. A third party who knows only the node ID cannot rotate the key.

---

## Threat model

| Threat | Mitigation |
|--------|-----------|
| Spoofed first announce | TOFU — first announce wins; key is then locked |
| Key rotation by a third party | Re-announce verified against stored public key |
| Replay attack on heartbeat/announce | Timestamp validated within ±5 minutes |
| Phantom node injection via federation | Announce signature verified before INSERT; missing/invalid sig → record skipped |
| Phone number substitution in federation | Phone is part of the signed announce message |
| XSS via malicious node_id in /map | All node fields HTML-escaped before rendering |
| Oversized request bodies (DoS) | Request bodies capped at 64 KB |
| Slow or oversized federation peer response | 20-second HTTP timeout; 1 MB response limit |
| Mass SMS spam | Twilio webhook validates `From`; rate limiting is future work |
| Registry data leak | Registry stores no sensitive data by design |
| Node data leak | Node DB is local; operator is responsible for host security |
| Unauthorized offer/need | Phone-based access only; no authentication beyond phone ownership |

### Known limitations
- The announce signature covers `nodeID|publicPhone|timestamp` but not `lat`, `lon`,
  or `coverage_radius_km`. A compromised peer registry could serve incorrect
  coordinates for a federated node. Impact: wrong map placement; SMS routing is
  unaffected since that uses the verified phone number.
- Replay within the 5-minute timestamp window is possible (no nonce). For heartbeats
  the worst case is a spurious `last_seen` bump. Acceptable for MVP.
- Federation uses TOFU for the public key of a brand-new node: the first registry to
  receive a direct announce sets the authoritative key. Subsequent federation
  propagation of that same node's key is verified against the stored value.

---

## Transport security

- All Twilio webhooks arrive over HTTPS via Twilio's infrastructure.
- Node-to-registry and registry-to-registry communication should use HTTPS in
  production (reverse proxy: nginx/caddy).
- When running multiple registries, peer traffic should be on a private network
  or authenticated via a reverse proxy if exposed publicly.

---

## Responsible disclosure

Report security issues to the project maintainers privately before public disclosure.
See `SECURITY.md` for the full findings log.
