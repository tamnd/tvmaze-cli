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
  {"score":0.9,"show":{"id":169,"name":"Breaking Bad","type":"Scripted","language":"English","genres":["Drama","Crime"],"status":"Ended","premiered":"2008-01-20","rating":{"average":9.2},"network":{"name":"AMC"},"summary":"<p>A chemistry teacher.</p>"}},
  {"score":0.7,"show":{"id":1371,"name":"Better Call Saul","type":"Scripted","language":"English","genres":["Drama"],"status":"Ended","premiered":"2015-02-08","rating":{"average":8.9},"network":{"name":"AMC"},"summary":"<p>A prequel.</p>"}}
]`

const fakeScheduleJSON = `[
  {"id":100001,"name":"Ep1","season":1,"number":1,"airdate":"2024-01-15","airtime":"21:00","show":{"id":169,"name":"Breaking Bad","network":{"name":"AMC"}}},
  {"id":100002,"name":"Ep2","season":1,"number":2,"airdate":"2024-01-15","airtime":"22:00","show":{"id":999,"name":"Test Show","network":{"name":"NBC"}}}
]`

const fakeShowJSON = `{"id":169,"name":"Breaking Bad","type":"Scripted","language":"English","genres":["Drama","Crime","Thriller"],"status":"Ended","premiered":"2008-01-20","ended":"2013-09-29","rating":{"average":9.2},"network":{"name":"AMC"},"summary":"<p>A high school chemistry teacher.</p>"}`

const fakeEpisodesJSON = `[
  {"id":1,"name":"Pilot","season":1,"number":1,"airdate":"2008-01-20","summary":"<p>Walter White.</p>","runtime":58,"rating":{"average":8.0}},
  {"id":2,"name":"Cat's in the Bag","season":1,"number":2,"airdate":"2008-01-27","summary":"<p>Walt and Jesse.</p>","runtime":48,"rating":{"average":7.5}},
  {"id":10,"name":"Seven Thirty-Seven","season":2,"number":1,"airdate":"2009-03-08","summary":"<p>Season 2 premiere.</p>","runtime":47,"rating":{"average":8.3}}
]`

const fakeCastJSON = `[
  {"person":{"id":1,"name":"Bryan Cranston","birthday":"1956-03-07","country":{"name":"United States"}},"character":{"name":"Walter White"},"self":false,"voice":false},
  {"person":{"id":2,"name":"Aaron Paul","birthday":"1979-08-27","country":{"name":"United States"}},"character":{"name":"Jesse Pinkman"},"self":false,"voice":false}
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
	if items[0].ID != 169 {
		t.Errorf("items[0].ID = %d, want 169", items[0].ID)
	}
	if items[0].Name != "Breaking Bad" {
		t.Errorf("items[0].Name = %q, want Breaking Bad", items[0].Name)
	}
	if items[0].Rating != "9.2" {
		t.Errorf("items[0].Rating = %q, want 9.2", items[0].Rating)
	}
	if items[0].Network != "AMC" {
		t.Errorf("items[0].Network = %q, want AMC", items[0].Network)
	}
	if items[0].Genres != "Drama, Crime" {
		t.Errorf("items[0].Genres = %q, want Drama, Crime", items[0].Genres)
	}
	if strings.Contains(items[0].Summary, "<p>") {
		t.Errorf("items[0].Summary contains HTML tags: %q", items[0].Summary)
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

func TestGetShowParses(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/shows/") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeShowJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	show, err := c.GetShow(context.Background(), 169)
	if err != nil {
		t.Fatal(err)
	}
	if show.ID != 169 {
		t.Errorf("ID = %d, want 169", show.ID)
	}
	if show.Name != "Breaking Bad" {
		t.Errorf("Name = %q, want Breaking Bad", show.Name)
	}
	if show.Rating != "9.2" {
		t.Errorf("Rating = %q, want 9.2", show.Rating)
	}
	if strings.Contains(show.Summary, "<p>") {
		t.Errorf("Summary contains HTML: %q", show.Summary)
	}
	if show.Network != "AMC" {
		t.Errorf("Network = %q, want AMC", show.Network)
	}
	if show.Ended != "2013-09-29" {
		t.Errorf("Ended = %q, want 2013-09-29", show.Ended)
	}
	if show.Genres != "Drama, Crime, Thriller" {
		t.Errorf("Genres = %q, want Drama, Crime, Thriller", show.Genres)
	}
}

func TestEpisodesParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/episodes") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeEpisodesJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	eps, err := c.Episodes(context.Background(), 169)
	if err != nil {
		t.Fatal(err)
	}
	if len(eps) != 3 {
		t.Fatalf("len(eps) = %d, want 3", len(eps))
	}
	ep := eps[0]
	if ep.ID != 1 {
		t.Errorf("ID = %d, want 1", ep.ID)
	}
	if ep.Name != "Pilot" {
		t.Errorf("Name = %q, want Pilot", ep.Name)
	}
	if ep.Season != 1 {
		t.Errorf("Season = %d, want 1", ep.Season)
	}
	if ep.Number != 1 {
		t.Errorf("Number = %d, want 1", ep.Number)
	}
	if ep.Airdate != "2008-01-20" {
		t.Errorf("Airdate = %q, want 2008-01-20", ep.Airdate)
	}
	if ep.Runtime != 58 {
		t.Errorf("Runtime = %d, want 58", ep.Runtime)
	}
	if ep.Rating != "8.0" {
		t.Errorf("Rating = %q, want 8.0", ep.Rating)
	}
	if strings.Contains(ep.Summary, "<p>") {
		t.Errorf("Summary contains HTML: %q", ep.Summary)
	}
}

func TestCastParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/cast") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeCastJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	cast, err := c.Cast(context.Background(), 169)
	if err != nil {
		t.Fatal(err)
	}
	if len(cast) != 2 {
		t.Fatalf("len(cast) = %d, want 2", len(cast))
	}
	m := cast[0]
	if m.PersonID != 1 {
		t.Errorf("PersonID = %d, want 1", m.PersonID)
	}
	if m.PersonName != "Bryan Cranston" {
		t.Errorf("PersonName = %q, want Bryan Cranston", m.PersonName)
	}
	if m.Character != "Walter White" {
		t.Errorf("Character = %q, want Walter White", m.Character)
	}
	if m.Birthday != "1956-03-07" {
		t.Errorf("Birthday = %q, want 1956-03-07", m.Birthday)
	}
	if m.Country != "United States" {
		t.Errorf("Country = %q, want United States", m.Country)
	}
	if cast[1].PersonName != "Aaron Paul" {
		t.Errorf("cast[1].PersonName = %q, want Aaron Paul", cast[1].PersonName)
	}
}

func TestScheduleParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/schedule") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, fakeScheduleJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.GetSchedule(context.Background(), "US", "2024-01-15", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	it := items[0]
	if it.ShowName != "Breaking Bad" {
		t.Errorf("ShowName = %q, want Breaking Bad", it.ShowName)
	}
	if it.Episode != "Ep1" {
		t.Errorf("Episode = %q, want Ep1", it.Episode)
	}
	if it.Network != "AMC" {
		t.Errorf("Network = %q, want AMC", it.Network)
	}
	if it.Airdate != "2024-01-15" {
		t.Errorf("Airdate = %q, want 2024-01-15", it.Airdate)
	}
	if it.Airtime != "21:00" {
		t.Errorf("Airtime = %q, want 21:00", it.Airtime)
	}
	if items[1].ShowName != "Test Show" {
		t.Errorf("items[1].ShowName = %q, want Test Show", items[1].ShowName)
	}
}

func TestScheduleLimitRespected(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeScheduleJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.GetSchedule(context.Background(), "US", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestScheduleDateParamIncluded(t *testing.T) {
	var gotURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.RawQuery
		_, _ = fmt.Fprint(w, "[]")
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, _ = c.GetSchedule(context.Background(), "GB", "2024-06-01", 0)
	if !strings.Contains(gotURL, "date=2024-06-01") {
		t.Errorf("query %q does not contain date=2024-06-01", gotURL)
	}
	if !strings.Contains(gotURL, "country=GB") {
		t.Errorf("query %q does not contain country=GB", gotURL)
	}
}

func TestScheduleNoDateParam(t *testing.T) {
	var gotURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.RawQuery
		_, _ = fmt.Fprint(w, "[]")
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, _ = c.GetSchedule(context.Background(), "US", "", 0)
	if strings.Contains(gotURL, "date=") {
		t.Errorf("query %q should not contain date= when date is empty", gotURL)
	}
}
