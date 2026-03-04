# FoodRelay SMS Grammar

## Commands

All commands are case-insensitive. Parser is permissive: extra whitespace is ignored.

---

### OFFER

Register a food donation offer.

```
OFFER <description> <timewindow> <address>
```

**Examples:**
```
OFFER 3 trays of pasta until 8pm 123 Main St
OFFER hot soup 7-9pm First Baptist Church
OFFER sealed sandwiches until 6pm 400 Oak Ave
OFFER leftover catering 5-7pm 22 Church St
```

**Fields:**
- `description` — free text describing the food (required; everything before the time token)
- `timewindow` — see Time Window formats below (required)
- `address` — pickup address (required; everything after the time token)

**State after:** offer is `awaiting_ready`. Reply READY when food is packed and ready.

---

### READY

Marks your latest active offer as ready for pickup.

```
READY
```

State transitions: `awaiting_ready` → `ready` → matching attempted immediately.

---

### NEED

Register a standing food request. The system immediately replies with any
currently-available READY offer, or queues the request for the next match.

```
NEED [address]
```

**Examples:**
```
NEED
NEED 55 Church St
NEED downtown shelter
```

**Fields:**
- `address` — optional area hint for future proximity matching

---

### CANCEL

Cancels your latest active offer (if `awaiting_ready` or `ready`) or latest open need.

```
CANCEL
```

---

### FOOD

Displays a summary of all commands.

```
FOOD
```

---

## Time Window Formats

| Input | Interpretation |
|-------|---------------|
| `until 8pm` | now – 8:00 PM |
| `by 8pm` | now – 8:00 PM |
| `9-10pm` | 9:00 PM – 10:00 PM |
| `9am-10am` | 9:00 AM – 10:00 AM |
| `9:30-10:30pm` | 9:30 PM – 10:30 PM |
| `9pm` | 9:00 PM – 10:00 PM (1-hour window) |

### Next-day roll-forward

Time windows are always interpreted relative to when the message is sent. If the
window end has already passed, the entire window is automatically rolled forward
by 24 hours.

| Sent at | Message | Interpreted as |
|---------|---------|---------------|
| 8:00 PM | `OFFER soup until 10pm 123 Main` | Tonight until 10 PM |
| 11:00 PM | `OFFER soup until 10pm 123 Main` | **Tomorrow** until 10 PM |

---

## Matching

**When a donor replies READY:**
1. Engine looks for the oldest open NEED (FIFO).
2. Creates a job, updates statuses, and texts both parties.

**When a recipient texts NEED:**
1. Standing request is registered immediately.
2. Engine looks for the READY offer expiring soonest.
3. If found: recipient gets food details inline; donor is notified by SMS.
4. If not found: recipient gets "We'll notify you when food is available."

If no match is found immediately, the scheduler retries every 60 seconds.
Unmatched READY offers escalate to admins after `READY_ESCALATE_MINUTES`.

**Note:** one offer matches one need (1:1). Quantity is not tracked — the
description is free text and coordination happens human-to-human after match.

---

## Error handling

If a message cannot be parsed, the sender receives:

```
Sorry, I didn't understand "...". Reply FOOD for commands.
```

Common OFFER parse failures:
- No time window found → "could not find time window; use e.g. 'until 8pm' or '7-9pm'"
- No description before the time → missing description
- No address after the time → missing location
