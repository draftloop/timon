# Timon

Timon is a self-hosted monitoring daemon for servers and scripts. It tracks the health of **probes** (recurring checks) and **jobs** (scripts with a start and an end), opens incidents automatically when things go wrong, and sends notifications via configurable webhooks.

## How it works

Timon runs as a background daemon on your machine. Your scripts and cron jobs push their health status to the daemon over a Unix socket. The daemon applies your rules, manages incidents, and dispatches webhook notifications.

```
your scripts  ──push──►  timon daemon  ──►  SQLite database
                                       ──►  webhook calls
```

## Installation

```sh
curl -fsSL https://raw.githubusercontent.com/draftloop/timon/master/install.sh | sh
```

Or build from source:

```sh
go build -o timon .
```

## Getting started

**1. Start the daemon**

```sh
timon daemon
```

The daemon reads its configuration from `~/.config/timon/timon.toml` or `/etc/timon/timon.toml`. It starts with sensible defaults if no config file is found.

**2. Push your first probe**

```sh
timon push probe myapp.health healthy --comment "All good"
```

**3. Check the status**

```sh
timon status
```

## Commands

### `timon daemon`

Start the daemon. Reads config, opens the Unix socket, and starts background tasks.

```sh
timon daemon
```

> For quick testing, running `timon daemon` in a terminal is enough. For production use, set it up as a system service via the install script.

---

### `timon status`

Show active incidents and the health of all known probes and jobs.

```
ACTIVE INCIDENTS (1)
  INC-3  open  "db.backup is critical"  1h ago

PROBES & JOBS (3)
  myapp.health   ✓ healthy — 4m ago   "All good"
  db.backup      ✗ critical — 1h ago              INC-3
  nightly.sync   ✓ healthy — 8h ago
```

Use `watch` to get a live-updating view refreshed every 2 seconds:

```sh
watch -n2 timon status
```

---

### `timon summary`

Print a one-line health summary. Useful for shell prompts or status bars.

```sh
timon summary
# Timon — 1 active incidents · 1 critical (db.backup) · 0 stale · 0 warning · 2 healthy · 0 running jobs

timon summary --short
# Timon — 1 active incidents · 1 critical · 0 stale · 0 warning · 2 healthy · 0 running jobs
```

---

### `timon push probe <code> <health>`

Push a health report for a probe. Creates the probe automatically on first push.

```sh
timon push probe <code> <healthy|warning|critical> [flags]
```

| Flag | Description |
|------|-------------|
| `--comment <text>` | Optional comment attached to this sample |
| `--stale-after <duration>` | Flag the probe as stale if no push arrives within this delay |
| `--stale-incident-after <duration>` | Same as `--stale-after`, and also opens an incident |

```sh
timon push probe myapp.health healthy \
    --comment "All good" \
    --stale-after 5m \
    --stale-incident-after 15m
```

---

### `timon push job start <code>`

Start a new job run. Prints the run UID, which must be passed to subsequent `step` and `end` calls.

```sh
timon push job start <code> [flags]
```

| Flag | Description |
|------|-------------|
| `--comment <text>` | Optional start comment |
| `--stale-after <duration>` | Flag the job as stale if no push arrives within this delay after it ends |
| `--stale-incident-after <duration>` | Same as `--stale-after`, and also opens an incident |
| `--overtime-incident-after <duration>` | Open an incident if the job runs longer than this delay |
| `--overlap-incident` | Open an incident if a new run starts while one is already running (default: true) |

```sh
TIMON_JOB_RUN=$(timon push job start nightly.sync \
    --comment "Starting nightly sync" \
    --overtime-incident-after 30m \
    --stale-after 25h \
    --stale-incident-after 26h)
```

---

### `timon push job step <code:run-uid> <label> <health>`

Push a step to an ongoing job run. The run UID is passed directly in the code argument using the `code:run-uid` notation.

```sh
timon push job step <code:run-uid> <label> <healthy|warning|critical> [flags]
```

| Flag | Description |
|------|-------------|
| `--end` | End the run after this step; the step label is used as end comment if `--end-comment` is not set |
| `--end-comment <text>` | Override the end comment (only with `--end`) |

```sh
timon push job step nightly.sync:$TIMON_JOB_RUN "Exported data" healthy
timon push job step nightly.sync:$TIMON_JOB_RUN "Uploaded to S3" healthy
timon push job step nightly.sync:$TIMON_JOB_RUN "Sent report" healthy --end
```

---

### `timon push job end <code:run-uid>`

End a job run. This is an alternative to passing `--end` to the last `job step` when you want to end the run in a separate call.

```sh
timon push job end <code:run-uid> [--comment <text>]
```

---

### `timon push incident <title> [description]`

Manually open an incident. Prints the incident code (`INC-<id>`).

```sh
timon push incident "Payment gateway down" "Stripe returning 503 since 14:32"
# INC-7
```

---

### `timon show <code>`

Show details of a probe, job, specific run/sample, or incident.

```sh
timon show myapp.health          # probe overview + sample history
timon show myapp.health:<uid>    # specific probe sample
timon show nightly.sync          # job overview + run history
timon show nightly.sync:<uid>    # specific job run with steps
timon show INC-7                 # incident details + timeline
```

---

### `timon annotate <INC-id> <note>`

Add a note to an incident's timeline.

```sh
timon annotate INC-7 "Contacted Stripe support, ticket #8821"
```

---

### `timon resolve <INC-id>`

Mark an incident as resolved.

```sh
timon resolve INC-7
timon resolve INC-7 --note "Fixed by rolling back the payment service to v2.4.1"
```

---

### `timon delete <code>`

Permanently delete a probe or job (and all its history), a specific sample or run, or an incident. Prompts for confirmation unless `--yes` is passed.

By default, deletion is refused if the target is linked to an active incident, or if an incident is not yet resolved. Use `--force` to override.

```sh
timon delete myapp.health           # delete probe and all its samples
timon delete nightly.sync           # delete job and all its runs
timon delete nightly.sync:abc123    # delete a specific run
timon delete INC-7                  # delete a resolved incident
timon delete myapp.health --force   # delete even if linked to an active incident
timon delete myapp.health --yes     # skip confirmation prompt
```

---

### `timon truncate`

Bulk-delete old samples, runs, and resolved incidents based on retention durations. Items linked to active incidents are silently skipped — no error is returned, making it safe to use in batch scripts.

```sh
timon truncate [<code>] [--keep <duration>] [--keep-healthy <d>] [--keep-warning <d>] [--keep-critical <d>] [--keep-incidents <d>]
```

| Flag | Description |
|------|-------------|
| `<code>` | Optional probe or job code — restrict the truncation to this probe or job |
| `--keep <duration>` | Delete samples and runs older than this duration (shorthand for all three health flags) |
| `--keep-healthy <duration>` | Retention duration for healthy samples and runs |
| `--keep-warning <duration>` | Retention duration for warning samples and runs |
| `--keep-critical <duration>` | Retention duration for critical samples and runs |
| `--keep-incidents <duration>` | Delete resolved incidents older than this duration |

`--keep` is mutually exclusive with `--keep-healthy`, `--keep-warning`, and `--keep-critical`. At least one flag is required.

When health flags are used, omitted flags inherit from the next higher severity: `--keep-healthy` defaults to `--keep-warning`, which defaults to `--keep-critical`. Samples and runs of a given health are kept indefinitely if no applicable flag is set.

Unfinished runs that never received a warning or critical step (health unknown) are treated as critical for truncation purposes.

```sh
timon truncate --keep 30d                                  # all samples and runs older than 30 days
timon truncate myapp.health --keep 30d                     # samples of a single probe
timon truncate --keep-healthy 7d --keep-critical 90d       # warning inherits critical: 90d
timon truncate --keep 30d --keep-incidents 90d             # samples, runs, and resolved incidents
timon truncate --keep-incidents 180d                       # resolved incidents only
```

---

## Incidents

Incidents are opened automatically based on the rules you set, or manually with `timon push incident`.

| Trigger | Cause | Auto-generated title |
|---------|-------|----------------------|
| `critical` | A probe push with health `critical`, or a job run that ended with at least one `critical` step | `<code> is critical` |
| `stale` | No push received before `--stale-incident-after` expires | `<code> is stale` |
| `job_overtime` | A job run exceeds `--overtime-incident-after` | `<code> is overtime` |
| `job_overlap` | A new run starts while one is already running | `<code> is overlapping` |
| `manual` | Created explicitly with `timon push incident` | *(user-supplied)* |

An incident transitions through the following states:

```
open  ──(probe recovers)──►  recovered  ──(degrades again)──►  relapsed
  │                              │                                  │
  └──────────────────────────────┴──────────(timon resolve)────────►  resolved
```

Resolving an incident (`timon resolve`) is permanent and can be done from any state. Recovered/relapsed transitions happen automatically as health reports come in.

---

## Configuration

Config is loaded from the first file found among:
- `~/.config/timon/timon.toml`
- `/etc/timon/timon.toml`

All settings are optional. Durations accept `ns`, `us`, `ms`, `s`, `m`, `h`, `d`, `w`, `mo`, `y`. Units above `h` are approximate (`d` = 24h, `w` = 168h, `mo` = 720h, `y` = 8760h); use `h` or smaller when precision matters.

For local development, a minimal config is enough:

```toml
[daemon]
data_dir  = "/tmp/timon/"
log_level = "debug"
```

Full reference:

```toml
[daemon]
hostname      = "prod-server-1"   # used in webhooks; defaults to machine hostname
data_dir      = "/etc/timon/"     # SQLite database location; defaults to /etc/timon/
log_dir       = "/var/log/timon/" # log file location when installed as a service; logs go to stdout otherwise
log_level     = "info"            # silent | fatal | error | warn | info | debug
ping_interval = "5m"              # send a timon.ping webhook event on this interval

[[webhook]]
on      = ["incident.open", "incident.relapsed"]
url     = "https://gotify.internal/message?token=CHANGE_ME"
cert    = "/usr/local/share/ca-certificates/extra/myca.crt"  # optional custom CA certificate to trust
headers = { "X-My-Header" = "yes" }
body    = """
{ "message": {{ if .incident.description }}{{ json .incident.description }}{{ else }}{{ json .incident.title }}{{ end }}, "title": {{ json .incident.title }}, "priority": 8 }
"""

[webhook.retry]
attempts = 3     # retries after the initial attempt (0 = no retry); defaults to 5
timeout  = "10s" # per-request timeout
delay    = "5s"  # delay between attempts
```

### Webhook events

| Event | Description |
|-------|-------------|
| `incident.open` | An incident was opened |
| `incident.recovered` | An incident recovered |
| `incident.relapsed` | A recovered incident relapsed |
| `incident.resolved` | An incident was manually resolved |
| `incident.annotated` | An annotation was added to an incident |
| `timon.ping` | Periodic heartbeat (requires `ping_interval`) |
| `timon.started` | The daemon started |

### Webhook template

The body is a [Go template](https://pkg.go.dev/text/template). The following variables are always available, plus additional ones depending on the event:

| Variable / Function      | Description |
|--------------------------|-------------|
| `._hostname`             | Daemon hostname |
| `._event`                | Event name (e.g. `incident.open`) |
| `._timestamp`            | RFC3339 timestamp of the event |
| `{{ json .value }}`      | Encode a value as a JSON string |
| `{{ urlencode .value }}` | URL-encode a string |

---

## Example: cron job monitoring

Store the `code:run-uid` pair in a variable to avoid repeating the job code on every call. The `[ -n "$RUN" ] &&` guard ensures timon calls are silently skipped if the daemon is unreachable — the actual job always runs.

```sh
#!/bin/sh

_UID=$(timon push job start db.backup \
    --overtime-incident-after 1h \
    --stale-after 25h \
    --stale-incident-after 26h)
RUN="${_UID:+db.backup:$_UID}"

pg_dump mydb | gzip > /backups/mydb.gz
[ -n "$RUN" ] && timon push job step "$RUN" "Dump completed" healthy

aws s3 cp /backups/mydb.gz s3://mybucket/
[ -n "$RUN" ] && timon push job step "$RUN" "Uploaded to S3" healthy

[ -n "$RUN" ] && timon push job end "$RUN" --comment "Backup OK"
```

## Example: probe from a cron job

```sh
# /etc/cron.d/timon-probe
*/5 * * * * root /usr/local/bin/check-myapp.sh
```

```sh
#!/bin/sh
# check-myapp.sh

if curl -sf http://localhost:8080/health > /dev/null; then
  timon push probe myapp.health healthy --stale-incident-after 10m
else
  timon push probe myapp.health critical --stale-incident-after 10m
fi
```

---

## License

[MIT](LICENSE)
