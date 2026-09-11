package sumoapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpcomingFetchesMakuuchiDays(t *testing.T) {
	var paths []string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		var body string
		switch r.URL.Path {
		case "/api/basho/202609":
			body = `{"date":"202609","startDate":"2026-09-13T00:00:00Z","endDate":"2026-09-27T00:00:00Z"}`
		case "/api/basho/202609/torikumi/Makuuchi/1":
			body = `{"date":"202609","torikumi":[{"id":"202609-1-1-1-2","bashoId":"202609","division":"Makuuchi","day":1,"matchNo":1,"eastShikona":"Hoshoryu","westShikona":"Onosato"}]}`
		case "/api/basho/202609/torikumi/Makuuchi/2":
			body = `{"date":"202609","torikumi":[]}`
		default:
			return nil, &urlError{path: r.URL.Path}
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	result, err := New(httpClient).Upcoming(context.Background(), now, 4, time.FixedZone("JST", 9*60*60))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("got %d matches, want 1", len(result.Matches))
	}
	if got := result.Matches[0].East; got != "Hoshoryu" {
		t.Errorf("east = %q", got)
	}
	if got := result.Matches[0].Date.Format("2006-01-02"); got != "2026-09-13" {
		t.Errorf("date = %s", got)
	}
	if len(paths) != 3 {
		t.Fatalf("got paths %v", paths)
	}
}

type urlError struct{ path string }

func (e *urlError) Error() string { return "unexpected path: " + e.path }
