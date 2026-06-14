package tvmaze_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/tvmaze-cli/tvmaze"
)

const fakeSearchJSON = `[
  {"score":0.9,"show":{"id":169,"name":"Breaking Bad","type":"Scripted","genres":["Drama","Crime"],"status":"Ended","premiered":"2008-01-20","rating":{"average":9.2},"network":{"name":"AMC"},"summary":"<p>A chemistry teacher.</p>","url":"https://www.tvmaze.com/shows/169/breaking-bad"}},
  {"score":0.7,"show":{"id":1371,"name":"Better Call Saul","type":"Scripted","genres":["Drama"],"status":"Ended","premiered":"2015-02-08","rating":{"average":8.9},"network":{"name":"AMC"},"summary":"<p>A prequel.</p>","url":"https://www.tvmaze.com/shows/1371/better-call-saul"}}
]`

const fakeScheduleJSON = `[
  {"id":100001,"name":"Ep1","season":1,"number":1,"show":{"id":169,"name":"Breaking Bad","type":"Scripted","genres":["Drama"],"status":"Running","premiered":"2026-01-01","rating":{"average":8.5},"network":{"name":"AMC"},"summary":"<p>A drama.</p>","url":"https://www.tvmaze.com/shows/169/test"}},
  {"id":100002,"name":"Ep2","season":1,"number":2,"show":{"id":999,"name":"Test Show","type":"Scripted","genres":[],"status":"Running","premiered":"2026-01-01","rating":{"average":7.0},"network":{"name":"NBC"},"summary":"","url":"https://www.tvmaze.com/shows/999/test"}}
]`

const fakeScheduleDupeJSON = `[
  {
    "id": 100001,
    "name": "Episode 1",
    "season": 5,
    "number": 1,
    "show": {
      "id": 169,
      "name": "Breaking Bad",
      "type": "Scripted",
      "genres": ["Drama"],
      "status": "Running",
      "premiered": "2026-01-01",
      "rating": {"average": 8.5},
      "network": {"name": "AMC"},
      "summary": "<p>A drama show.</p>",
      "url": "https://www.tvmaze.com/shows/169/test"
    }
  },
  {
    "id": 100002,
    "name": "Episode 2",
    "season": 5,
    "number": 2,
    "show": {
      "id": 169,
      "name": "Breaking Bad",
      "type": "Scripted",
      "genres": ["Drama"],
      "status": "Running",
      "premiered": "2026-01-01",
      "rating": {"average": 8.5},
      "network": {"name": "AMC"},
      "summary": "<p>A drama show.</p>",
      "url": "https://www.tvmaze.com/shows/169/test"
    }
  }
]`

func newTestClient(ts *httptest.Server) *tvmaze.Client {
	cfg := tvmaze.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return tvmaze.NewClient(cfg)
}

func TestSearchSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Search(context.Background(), "test", 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
	if !strings.Contains(gotUA, "tvmaze-cli") {
		t.Errorf("User-Agent %q does not contain tvmaze-cli", gotUA)
	}
}

func TestSearchParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Search(context.Background(), "breaking bad", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Rank != 1 {
		t.Errorf("items[0].Rank = %d, want 1", items[0].Rank)
	}
	if items[0].Name != "Breaking Bad" {
		t.Errorf("items[0].Name = %q, want Breaking Bad", items[0].Name)
	}
	if items[0].Rating != 9.2 {
		t.Errorf("items[0].Rating = %v, want 9.2", items[0].Rating)
	}
	if items[0].Network != "AMC" {
		t.Errorf("items[0].Network = %q, want AMC", items[0].Network)
	}
	if strings.Contains(items[0].Summary, "<p>") {
		t.Errorf("items[0].Summary contains HTML tags: %q", items[0].Summary)
	}
	const wantURLPrefix = "https://www.tvmaze.com/"
	if !strings.HasPrefix(items[0].URL, wantURLPrefix) {
		t.Errorf("items[0].URL = %q, want prefix %q", items[0].URL, wantURLPrefix)
	}
}

func TestSearchLimitRespected(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Search(context.Background(), "breaking bad", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestSearchRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	cfg := tvmaze.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := tvmaze.NewClient(cfg)

	_, err := c.Search(context.Background(), "test", 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestScheduleParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeScheduleJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Schedule(context.Background(), "US", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Rank != 1 {
		t.Errorf("items[0].Rank = %d, want 1", items[0].Rank)
	}
	if items[0].Name != "Breaking Bad" {
		t.Errorf("items[0].Name = %q, want Breaking Bad", items[0].Name)
	}
	if strings.Contains(items[0].Summary, "<p>") {
		t.Errorf("items[0].Summary contains HTML tags: %q", items[0].Summary)
	}
	if items[1].Name != "Test Show" {
		t.Errorf("items[1].Name = %q, want Test Show", items[1].Name)
	}
}

func TestScheduleDeduplicates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeScheduleDupeJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Schedule(context.Background(), "US", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1 (deduplication)", len(items))
	}
}
