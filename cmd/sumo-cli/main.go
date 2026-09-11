// sumo-cli lists upcoming Makuuchi bouts from sumo-api.com.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/atillalab/sumo-cli/internal/sumoapi"
)

const (
	exitOK    = 0
	exitData  = 1
	exitUsage = 2
)

var Version = "v0.0.0-dev"

const usage = `sumo-cli - upcoming Makuuchi bout schedule

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

The data comes from https://www.sumo-api.com/api/.
Exit codes: 0 success, 1 data/runtime error, 2 usage error.
`

type CLI struct {
	Out        io.Writer
	Err        io.Writer
	Now        func() time.Time
	HTTPClient *http.Client
}

func main() {
	cli := &CLI{Out: os.Stdout, Err: os.Stderr, Now: time.Now, HTTPClient: http.DefaultClient}
	os.Exit(cli.Run(os.Args[1:]))
}

func (c *CLI) Run(args []string) int {
	if c.Out == nil {
		c.Out = os.Stdout
	}
	if c.Err == nil {
		c.Err = os.Stderr
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}

	if len(args) == 0 {
		fmt.Fprint(c.Out, usage)
		return exitOK
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(c.Out, usage)
		return exitOK
	case "-V", "--version", "version":
		fmt.Fprintln(c.Out, "sumo-cli "+Version)
		return exitOK
	case "matches":
		return c.runMatches(args[1:])
	case "basho":
		return c.runBasho(args[1:])
	default:
		fmt.Fprintf(c.Err, "sumo-cli: unknown command %q\n\n%s", args[0], usage)
		return exitUsage
	}
}

func (c *CLI) runBasho(args []string) int {
	if len(args) == 0 || args[0] != "next" {
		fmt.Fprintf(c.Err, "sumo-cli: usage: sumo-cli basho next [--timezone TZ] [--json]\n")
		return exitUsage
	}
	fs := flag.NewFlagSet("basho next", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	timezone := fs.String("timezone", "Asia/Tokyo", "IANA timezone")
	jsonOutput := fs.Bool("json", false, "machine-readable JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(c.Err, "sumo-cli: basho next accepts no positional arguments")
		return exitUsage
	}
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		fmt.Fprintf(c.Err, "sumo-cli: invalid timezone %q: %v\n", *timezone, err)
		return exitUsage
	}
	basho, err := sumoapi.New(c.HTTPClient).NextBasho(context.Background(), c.Now(), loc)
	if err != nil {
		if *jsonOutput {
			return writeJSONError(c.Out, err)
		}
		fmt.Fprintf(c.Err, "sumo-cli: %v\n", err)
		return exitData
	}
	if *jsonOutput {
		return writeJSON(c.Out, map[string]any{
			"schemaVersion": 1,
			"command":       "basho next",
			"basho":         basho,
			"timezone":      *timezone,
			"generatedAt":   c.Now().UTC(),
		})
	}
	fmt.Fprintf(c.Out, "%s Grand Tournament\n%s – %s\n", bashoName(basho.StartDate.In(loc)), basho.StartDate.In(loc).Format("2006-01-02"), basho.EndDate.In(loc).Format("2006-01-02"))
	return exitOK
}

func bashoName(start time.Time) string {
	return start.Format("January 2006")
}

func (c *CLI) runMatches(args []string) int {
	fs := flag.NewFlagSet("matches", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	days := fs.Int("days", 15, "number of calendar days to show")
	timezone := fs.String("timezone", "Asia/Tokyo", "IANA timezone")
	jsonOutput := fs.Bool("json", false, "machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 || *days < 1 {
		fmt.Fprintln(c.Err, "sumo-cli: --days must be a positive number and matches accepts no positional arguments")
		return exitUsage
	}
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		fmt.Fprintf(c.Err, "sumo-cli: invalid timezone %q: %v\n", *timezone, err)
		return exitUsage
	}

	result, err := sumoapi.New(c.HTTPClient).Upcoming(context.Background(), c.Now(), *days, loc)
	if err != nil {
		if *jsonOutput {
			return writeJSONError(c.Out, err)
		}
		fmt.Fprintf(c.Err, "sumo-cli: %v\n", err)
		return exitData
	}
	if *jsonOutput {
		return writeJSON(c.Out, map[string]any{
			"schemaVersion": 1,
			"command":       "matches",
			"division":      "Makuuchi",
			"basho":         result.Basho,
			"matches":       result.Matches,
			"timezone":      *timezone,
			"generatedAt":   c.Now().UTC(),
		})
	}
	fmt.Fprint(c.Out, renderTable(result.Matches, loc))
	return exitOK
}

func writeJSON(w io.Writer, value any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return exitData
	}
	return exitOK
}

func writeJSONError(w io.Writer, err error) int {
	return writeJSON(w, map[string]any{"error": err.Error()})
}

func renderTable(matches []sumoapi.Match, loc *time.Location) string {
	if len(matches) == 0 {
		return "No upcoming matches found.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%-12s  %-4s  %-24s  %-24s\n", "DATE", "NO", "EAST", "WEST")
	for _, m := range matches {
		fmt.Fprintf(&b, "%-12s  %-4d  %-24s  %-24s\n", m.Date.In(loc).Format("2006-01-02"), m.MatchNumber, m.East, m.West)
	}
	return b.String()
}
