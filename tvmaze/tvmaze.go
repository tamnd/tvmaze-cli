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
const Host = "tvmaze.com"

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
	for i, r := range results {
		items = append(items, normalizeShow(r.Show, i+1))
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items, nil
}

// Schedule fetches today's TV schedule for the given country from /schedule.
// Country defaults to "US" if empty. Shows are deduplicated by show ID.
// It returns at most limit results (pass 0 for all).
func (c *Client) Schedule(ctx context.Context, country string, limit int) ([]Show, error) {
	if country == "" {
		country = "US"
	}
	u := fmt.Sprintf("%s/schedule?country=%s", c.cfg.BaseURL, neturl.QueryEscape(country))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var items []scheduleItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("decode schedule: %w", err)
	}
	seen := map[int]bool{}
	var shows []Show
	for _, item := range items {
		if seen[item.Show.ID] {
			continue
		}
		seen[item.Show.ID] = true
		shows = append(shows, normalizeShow(item.Show, len(shows)+1))
	}
	if limit > 0 && limit < len(shows) {
		shows = shows[:limit]
	}
	return shows, nil
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
	s := normalizeShow(r, 0)
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
		out[i] = Episode{
			ID:      e.ID,
			Name:    e.Name,
			Season:  e.Season,
			Number:  e.Number,
			Airdate: e.Airdate,
			Summary: stripHTML(e.Summary),
			Runtime: e.Runtime,
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
			PersonName:    m.Person.Name,
			CharacterName: m.Character.Name,
			Birthday:      m.Person.Birthday,
		}
	}
	return out, nil
}

// normalizeShow converts a rawShow into the public Show type.
func normalizeShow(r rawShow, rank int) Show {
	var rating float64
	if r.Rating.Average != nil {
		rating = *r.Rating.Average
	}
	return Show{
		Rank:      rank,
		ID:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Genres:    r.Genres,
		Status:    r.Status,
		Premiered: r.Premiered,
		Rating:    rating,
		Network:   r.Network.Name,
		Summary:   stripHTML(r.Summary),
		URL:       r.URL,
	}
}

var htmlTagRE = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRE.ReplaceAllString(s, ""))
}
