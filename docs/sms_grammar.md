# FoodRelay SMS Grammar

## Commands

All commands are case-insensitive. Parser is permissive: extra whitespace is ignored.

---

### OFFER

Register a food donation offer.

```
OFFER <qty> [unit] <timewindow> <location> [notes: <text>] [allergens: <text>]
```

**Examples:**
```
OFFER 12 meals 9-10pm 123 Main St
OFFER 30 9am-11am 400 Oak Ave notes: sealed containers allergens: nuts dairy
OFFER 5 boxes by 8pm 22 Church St
```

**Fields:**
- `qty` — integer quantity (required)
- `unit` — one of: meals, items, boxes, bags, portions, servings, units, packages, lbs, kg (default: meals)
- `timewindow` — see Time Window formats below
- `location` — free text address (required)
- `notes:` — optional free text
- `allergens:` — optional free text

**State after:** offer is `awaiting_ready`. Reply READY when food is packed.

---

### READY

Marks your latest active offer as ready for pickup.

```
READY
```

State transitions: `awaiting_ready` → `ready` → matching attempted immediately.

---

### NEED

Register a food request.

```
NEED <qty> [unit] <timewindow> <location> [priority: <N>]
```

**Examples:**
```
NEED 20 meals by 8pm 55 Church St
NEED 10 by 7pm 100 Broad St priority: 3
NEED 50 meals 6-8pm 200 West Ave priority: 1
```

**Fields:**
- `qty` — integer quantity (required)
- `unit` — same options as OFFER (default: meals)
- `timewindow` — see Time Window formats below
- `location` — free text address (required)
- `priority:` — integer 1–10, higher = more urgent (default: 1)

---

### CANCEL

Cancels your latest active offer (if `awaiting_ready` or `ready`) or latest open need.

```
CANCEL
```

---

### HELP

Displays a summary of all commands.

```
HELP
```

---

## Time Window Formats

| Input | Interpretation |
|-------|---------------|
| `9-10pm` | 9:00 PM – 10:00 PM |
| `9am-10am` | 9:00 AM – 10:00 AM |
| `9:30-10:30pm` | 9:30 PM – 10:30 PM |
| `by 8pm` | now – 8:00 PM |
| `9pm` | 9:00 PM – 10:00 PM (1-hour window) |

### Next-day roll-forward

Time windows are always interpreted relative to when the message is sent. If the
window end has already passed by the time the SMS is received, the entire window is
automatically rolled forward by 24 hours.

| Sent at | Message | Interpreted as |
|---------|---------|---------------|
| 8:00 PM | `OFFER 10 meals 9-10pm 123 Main` | Tonight 9–10 PM |
| 10:30 PM | `OFFER 10 meals 9-10pm 123 Main` | **Tomorrow** 9–10 PM |
| 7:00 PM | `NEED 20 meals by 8pm 55 Church` | Tonight, by 8 PM |
| 9:00 PM | `NEED 20 meals by 8pm 55 Church` | **Tomorrow** by 8 PM |

This prevents late-evening texts from being immediately expired by the scheduler.
Windows whose end is still in the future are never rolled forward.

---

## Matching

When a donor replies READY, the engine:
1. Finds all open needs whose time window overlaps with the offer.
2. Scores each candidate: `priority × 10 + overlap_hours`.
3. Selects the highest-scoring need.
4. Creates a job, updates statuses, and texts both parties.

If no match is found immediately, the scheduler retries every 60 seconds.
Unmatched READY offers escalate to admins after `READY_ESCALATE_MINUTES`.

---

## Error handling

If a message cannot be parsed, the sender receives:

```
Sorry, I didn't understand "...". Reply HELP for commands.
```
