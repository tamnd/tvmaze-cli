// Package tvmaze is the library behind the tvmaze command line:
// the HTTP client, request shaping, and the typed data models for the TVMaze
// show search and schedule endpoints.
//
// The TVMaze public API at api.tvmaze.com requires no authentication. The
// Client sets a real User-Agent, paces requests at 200 ms intervals, and
// retries transient 429/5xx failures with exponential backoff.
package tvmaze

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "api.tvmaze.com"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.tvmaze.com",
		UserAgent: "Mozilla/5.0 (compatible; tvmaze-cli/dev; +https://github.com/tamnd/tvmaze-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to TVMaze over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Search fetches shows matching query from /search/shows.
// It returns at most limit results (pass 0 for all).
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Show, error) {
	u := fmt.Sprintf("%s/search/shows?q=%s", c.cfg.BaseURL, neturl.QueryEscape(query))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var results []searchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}
	items := make([]Show, 0, len(results))
	for _, r := range results {
		items = append(items, normalizeShow(r.Show))
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items, nil
}

// GetShow fetches a single show by its TVMaze ID.
func (c *Client) GetShow(ctx context.Context, id int) (*Show, error) {
	u := fmt.Sprintf("%s/shows/%d", c.cfg.BaseURL, id)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var r rawShow
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("decode show: %w", err)
	}
	s := normalizeShow(r)
	return &s, nil
}

// Episodes fetches all episodes for a show by its TVMaze ID.
func (c *Client) Episodes(ctx context.Context, showID int) ([]Episode, error) {
	u := fmt.Sprintf("%s/shows/%d/episodes", c.cfg.BaseURL, showID)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []rawEpisode
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode episodes: %w", err)
	}
	out := make([]Episode, len(raw))
	for i, e := range raw {
		var rating string
		if e.Rating.Average != nil {
			rating = fmt.Sprintf("%.1f", *e.Rating.Average)
		}
		summary := stripHTML(e.Summary)
		if len(summary) > 150 {
			summary = summary[:150]
		}
		out[i] = Episode{
			ID:      e.ID,
			Name:    e.Name,
			Season:  e.Season,
			Number:  e.Number,
			Airdate: e.Airdate,
			Summary: summary,
			Runtime: e.Runtime,
			Rating:  rating,
		}
	}
	return out, nil
}

// Cast fetches the cast for a show by its TVMaze ID.
func (c *Client) Cast(ctx context.Context, showID int) ([]CastMember, error) {
	u := fmt.Sprintf("%s/shows/%d/cast", c.cfg.BaseURL, showID)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []rawCast
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode cast: %w", err)
	}
	out := make([]CastMember, len(raw))
	for i, m := range raw {
		out[i] = CastMember{
			PersonID:   m.Person.ID,
			PersonName: m.Person.Name,
			Birthday:   m.Person.Birthday,
			Country:    m.Person.Country.Name,
			Character:  m.Character.Name,
		}
	}
	return out, nil
}

// GetSchedule fetches the TV schedule for the given country and date from /schedule.
// Country defaults to "US" if empty. Date should be YYYY-MM-DD; if empty, the date param is omitted.
// It returns at most limit results (pass 0 for all).
func (c *Client) GetSchedule(ctx context.Context, country, date string, limit int) ([]ScheduleItem, error) {
	if country == "" {
		country = "US"
	}
	u := fmt.Sprintf("%s/schedule?country=%s", c.cfg.BaseURL, neturl.QueryEscape(country))
	if date != "" {
		u += "&date=" + neturl.QueryEscape(date)
	}
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []rawScheduleItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode schedule: %w", err)
	}
	out := make([]ScheduleItem, 0, len(raw))
	for _, item := range raw {
		out = append(out, ScheduleItem{
			ShowName: item.Show.Name,
			Episode:  item.Name,
			Season:   item.Season,
			Number:   item.Number,
			Airdate:  item.Airdate,
			Airtime:  item.Airtime,
			Network:  item.Show.Network.Name,
		})
	}
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}

// normalizeShow converts a rawShow into the public Show type.
func normalizeShow(r rawShow) Show {
	var rating string
	if r.Rating.Average != nil {
		rating = fmt.Sprintf("%.1f", *r.Rating.Average)
	}
	summary := stripHTML(r.Summary)
	if len(summary) > 200 {
		summary = summary[:200]
	}
	genres := strings.Join(r.Genres, ", ")
	return Show{
		ID:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Status:    r.Status,
		Premiered: r.Premiered,
		Ended:     r.Ended,
		Rating:    rating,
		Network:   r.Network.Name,
		Genres:    genres,
		Summary:   summary,
	}
}

var htmlTagRE = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRE.ReplaceAllString(s, ""))
}
