# sumo-cli

A small Go CLI that lists upcoming Makuuchi bouts and the next Grand Tournament (`basho`) using data from [sumo-api.com](https://www.sumo-api.com/api/).

## Features

- `matches` — list upcoming top-division (Makuuchi) bouts for the next N calendar days
- `basho next` — show the next Grand Tournament (name + start/end dates)
- Timezone-aware dates (default: `Asia/Tokyo`)
- Human-readable table output and machine-readable `--json` output
- Explicit exit codes for scripting: `0` success, `1` data/runtime error, `2` usage error

## Requirements

- Go 1.22+

## Install

```sh
# Run directly
go run ./cmd/sumo-cli --help

# Build a binary
go build -o sumo-cli ./cmd/sumo-cli

# Install into $GOPATH/bin
go install ./cmd/sumo-cli
```

The module is `github.com/atillalab/sumo-cli`.

## Usage

```text
sumo-cli - upcoming Makuuchi bout schedule

Usage:
  sumo-cli matches [options]
  sumo-cli --help | --version

Commands:
  matches    list upcoming bouts from the top division (Makuuchi)
  basho next show the next Grand Tournament

Options:
  --days N       number of calendar days to show (default 15)
  --timezone TZ  IANA timezone for dates (default Asia/Tokyo)
  --json         machine-readable JSON output
```

### List upcoming matches

```sh
# Next 15 days (default), Tokyo time
sumo-cli matches

# Next 3 days
sumo-cli matches --days 3

# Different timezone
sumo-cli matches --days 7 --timezone Europe/Berlin

# JSON output
sumo-cli matches --days 3 --json
```

Table output:

```text
DATE          NO    EAST                      WEST
2026-09-13    1     Hoshoryu                  Onosato
```

JSON output includes `schemaVersion`, `command`, `division`, `basho`, `matches`, `timezone`, and `generatedAt`.

### Show the next basho

```sh
sumo-cli basho next
sumo-cli basho next --timezone Europe/Berlin
sumo-cli basho next --json
```

Example output:

```text
September 2026 Grand Tournament
2026-09-13 – 2026-09-27
```

Note: `basho next` returns the first tournament whose start date is today or later in the requested timezone, so an ongoing tournament is skipped.

### Other commands

```sh
sumo-cli --help
sumo-cli --version
sumo-cli help
sumo-cli version
```

Running with no arguments prints usage and exits `0`.

## Exit codes

- `0` — success
- `1` — data / runtime error (e.g. API failure)
- `2` — usage error (unknown command, bad flag, invalid `--days` / `--timezone`)

In `--json` mode, data errors are returned as `{"error": "..."}` on stdout.

## How it works

- Grand tournaments are held in odd months. The client probes `GET /basho/YYYYMM` for the current and next five tournament months.
- For `matches`, it finds the basho covering the requested window, clamps the day range to days 1–15, then fetches `GET /basho/{bashoId}/torikumi/Makuuchi/{day}` for each day.
- Dates are normalized to local midnight in the requested IANA timezone; `--days N` covers today + N-1 following days (end-exclusive).

API base URL: `https://www.sumo-api.com/api`

## Project layout

```text
cmd/sumo-cli/main.go          CLI entrypoint, flags, table/JSON rendering
internal/sumoapi/client.go    sumo-api.com client (NextBasho, Upcoming)
internal/sumoapi/client_test.go  unit tests with stubbed HTTP transport
go.mod                        module github.com/atillalab/sumo-cli (go 1.22.0)
```

## Development

```sh
go test ./...
go vet ./...
go run ./cmd/sumo-cli matches --days 3
```
