# FoodRelay Node ↔ Registry Protocol

## Overview

Nodes announce themselves on boot and send a heartbeat every 5 minutes to every
registry listed in `REGISTRY_URLS`. Registries verify ed25519 signatures to prevent
spoofing. Multiple registries form a federation mesh by pulling records from each other
every 5 minutes via a gossip endpoint.

---

## Node → Registry endpoints

### POST /v1/nodes/announce

Registers or updates a node. Called once on startup; re-called after a key rotation.

On **first announce** (TOFU): the registry stores the provided public key.
On **re-announce**: the registry verifies the signature against the *already-stored*
public key. A different keypair cannot overwrite an existing registration.

**Request body (JSON):**
```json
{
  "node_id":            "node-nyc-01",
  "public_phone":       "+15551234567",
  "lat":                40.7128,
  "lon":               -74.0060,
  "coverage_radius_km": 10,
  "capabilities":       ["sms"],
  "version":            "0.1.0",
  "public_key":         "<hex-encoded ed25519 pubkey>",
  "timestamp":          1709400000,
  "signature":          "<hex-encoded ed25519 signature>"
}
```

**Announce signature:**
```
message   = node_id + "|" + public_phone + "|" + timestamp
signature = ed25519.Sign(privateKey, []byte(message))
```

`public_phone` is included in the signed message so peer registries can verify the
phone number has not been substituted during federation propagation.

**Response 200:**
```json
{"status": "ok"}
```

**Response 401:** signature invalid, timestamp out of range (±5 min), or key mismatch
on re-announce.

---

### POST /v1/nodes/heartbeat

Updates `last_seen` for an existing node. Called every 5 minutes.
Verified against the stored public key.

**Request body (JSON):**
```json
{
  "node_id":   "node-nyc-01",
  "timestamp": 1709400300,
  "signature": "<hex-encoded ed25519 signature>"
}
```

**Heartbeat signature:**
```
message   = node_id + "|" + timestamp
signature = ed25519.Sign(privateKey, []byte(message))
```

**Response 200:**
```json
{"status": "ok"}
```

**Response 404:** node not found — call `/announce` first.
**Response 401:** signature invalid or timestamp out of range (±5 min).

---

### GET /v1/nodes

Returns all nodes active within the last 15 minutes. Public; no authentication.

**Response 200:**
```json
[
  {
    "node_id":            "node-nyc-01",
    "public_phone":       "+15551234567",
    "lat":                40.7128,
    "lon":               -74.0060,
    "coverage_radius_km": 10,
    "capabilities":       ["sms"],
    "version":            "0.1.0",
    "last_seen":          "2024-03-02T14:05:00Z"
  }
]
```

---

## Registry ↔ Registry (federation) endpoint

### GET /v1/federation/nodes

Returns all nodes seen in the last 24 hours, including the original announce signature.
Used exclusively by peer registries for gossip synchronisation.

The response window is 24 hours (vs. 15 minutes for `/v1/nodes`) so that a peer which
was briefly offline can catch up without missing records.

**Response 200:**
```json
[
  {
    "node_id":            "node-nyc-01",
    "public_phone":       "+15551234567",
    "lat":                40.7128,
    "lon":               -74.0060,
    "coverage_radius_km": 10,
    "capabilities":       ["sms"],
    "version":            "0.1.0",
    "public_key":         "<hex-encoded ed25519 pubkey>",
    "last_seen":          1709400000,
    "source":             "local",
    "announce_sig":       "<original hex-encoded ed25519 announce signature>",
    "announce_ts":        1709400000
  }
]
```

**Federation merge rules (receiving registry):**
1. Verify `announce_sig` over `node_id|public_phone|announce_ts` using `public_key`.
   Skip the record if the signature is missing or invalid.
2. If the `node_id` does not exist locally: INSERT with all fields from the peer.
3. If the `node_id` already exists: UPDATE only mutable fields (`capabilities`,
   `version`, `last_seen`, `announce_sig`, `announce_ts`, `source`) using
   "newest `last_seen` wins" logic. Identity fields (`public_phone`, `lat`, `lon`,
   `coverage_radius_km`, `public_key`) are **never** overwritten from federation —
   only a direct announce can change those.

---

## Federation mesh topology

```
Node → [Registry A, Registry B, ...]   (fan-out announce + heartbeat)

Registry A  ←pull—  Registry B
Registry B  ←pull—  Registry A
Registry C  ←pull—  Registry B         (C only needs to know 1 peer to bootstrap)
```

**Pull-based gossip** runs every 5 minutes. Each registry fetches
`/v1/federation/nodes` from each of its configured `REGISTRY_PEERS` and merges
records after signature verification.

A new registry only needs one known peer in `REGISTRY_PEERS`; it will pull all
verified records from that peer's 24-hour window on first sync.

If a registry goes down:
- Nodes that listed it in `REGISTRY_URLS` log the failure and continue heartbeating
  to all other registries.
- Peer registries stop pulling from it (timeout after 20 s) and continue syncing
  with their remaining peers.

---

## Security model

- Each node generates an ed25519 keypair on first run. The 32-byte seed is stored
  in `NODE_PRIVATE_KEY_HEX`.
- Announce signatures cover `nodeID|publicPhone|timestamp`, binding the phone number
  to the keypair. This prevents a peer registry from substituting a different phone.
- Heartbeat signatures cover `nodeID|timestamp` only (phone is not re-transmitted).
- Timestamps prevent replay attacks (±5 minute tolerance).
- On re-announce the registry verifies against the **stored** public key, preventing
  a third party from rotating the key for an existing node.
- Federation only propagates records whose announce signature verifies correctly.
- The registry never stores offer, need, or private address data.

---

## Active node definition

Nodes heartbeat every **24 hours**. A node is considered active for the public
`/v1/nodes` endpoint if its `last_seen` is within the past **48 hours** (2×
heartbeat interval — allows one missed heartbeat before being considered stale).

The federation `/v1/federation/nodes` endpoint uses a **72-hour** window so peers
can recover after a multi-day outage without losing node records.
