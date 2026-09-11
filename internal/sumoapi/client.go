package sumoapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://www.sumo-api.com/api"

type Client struct{ HTTPClient *http.Client }

type Basho struct {
	Date      string    `json:"date"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type Match struct {
	ID          string    `json:"id"`
	BashoID     string    `json:"bashoId"`
	Division    string    `json:"division"`
	Day         int       `json:"day"`
	MatchNumber int       `json:"matchNo"`
	East        string    `json:"eastShikona"`
	EastRank    string    `json:"eastRank"`
	West        string    `json:"westShikona"`
	WestRank    string    `json:"westRank"`
	Date        time.Time `json:"date"`
}

type dayResponse struct {
	Basho
	Torikumi []Match `json:"torikumi"`
}

type Schedule struct {
	Basho   Basho
	Matches []Match
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{HTTPClient: httpClient}
}

func (c *Client) Upcoming(ctx context.Context, now time.Time, days int, loc *time.Location) (Schedule, error) {
	if days < 1 {
		return Schedule{}, fmt.Errorf("days must be positive")
	}
	start := localMidnight(now, loc)
	var basho Basho
	var err error
	for _, candidate := range candidateBashoDates(start) {
		basho, err = c.getBasho(ctx, candidate)
		if err == nil && !basho.EndDate.Before(start.UTC()) {
			break
		}
	}
	if err != nil {
		return Schedule{}, err
	}

	end := start.AddDate(0, 0, days)
	firstDay := 1
	if start.After(basho.StartDate.In(loc)) {
		firstDay = int(start.Sub(localMidnight(basho.StartDate, loc))/24/time.Hour) + 1
		if firstDay < 1 {
			firstDay = 1
		}
	}
	lastDay := int(end.Sub(localMidnight(basho.StartDate, loc))/24/time.Hour) + 1
	if lastDay > 15 {
		lastDay = 15
	}
	if firstDay > lastDay {
		return Schedule{Basho: basho}, nil
	}

	var matches []Match
	for day := firstDay; day <= lastDay; day++ {
		response, fetchErr := c.getDay(ctx, basho.Date, day)
		if fetchErr != nil {
			return Schedule{}, fetchErr
		}
		date := localMidnight(basho.StartDate, loc).AddDate(0, 0, day-1)
		for _, match := range response.Torikumi {
			match.Date = date
			matches = append(matches, match)
		}
	}
	return Schedule{Basho: basho, Matches: matches}, nil
}

func candidateBashoDates(start time.Time) []string {
	// Grand tournaments are held in odd months. Include the current month
	// and the next five tournament months so a pre-basho query can advance.
	var out []string
	month := time.Month(start.Month())
	year := start.Year()
	for len(out) < 6 {
		if month%2 == 1 {
			out = append(out, fmt.Sprintf("%04d%02d", year, month))
		}
		month++
		if month == 13 {
			month = 1
			year++
		}
	}
	return out
}

func (c *Client) getBasho(ctx context.Context, date string) (Basho, error) {
	var result Basho
	err := c.get(ctx, "/basho/"+date, nil, &result)
	return result, err
}

func (c *Client) getDay(ctx context.Context, basho string, day int) (dayResponse, error) {
	var result dayResponse
	err := c.get(ctx, fmt.Sprintf("/basho/%s/torikumi/Makuuchi/%d", basho, day), nil, &result)
	return result, err
}

func (c *Client) get(ctx context.Context, path string, query url.Values, target any) error {
	u := baseURL + path
	if len(query) != 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned HTTP %s for %s", resp.Status, path)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func localMidnight(now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}
