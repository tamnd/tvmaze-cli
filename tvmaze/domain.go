package tvmaze

import (
	"context"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes tvmaze as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/tvmaze-cli/tvmaze"
//
// The same Domain also builds the standalone tvmaze binary (see cli.NewApp).
func init() { kit.Register(Domain{}) }

// Domain is the tvmaze driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "tvmaze",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "tvmaze",
			Short:  "Search TV shows and view today's schedule via TVMaze",
			Long: `tvmaze searches the TVMaze database and fetches today's TV schedule.
No API key required. Data is fetched from the public TVMaze REST API.`,
			Site: Host,
			Repo: "https://github.com/tamnd/tvmaze-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// search: find shows by name
	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search TV shows by name",
		Args:    []kit.Arg{{Name: "query", Help: "show name to search"}},
	}, searchOp)

	// schedule: today's TV schedule
	kit.Handle(app, kit.OpMeta{
		Name:    "schedule",
		Group:   "read",
		List:    true,
		Summary: "List today's TV schedule",
	}, scheduleOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type searchInput struct {
	Query  string        `kit:"arg"          help:"show name to search"`
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type scheduleInput struct {
	Country string        `kit:"flag"         help:"country code (default US)"`
	Limit   int           `kit:"flag,inherit" help:"max results"`
	Delay   time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client  *Client       `kit:"inject"`
}

// --- handlers ---

func searchOp(ctx context.Context, in searchInput, emit func(Show) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	items, err := in.Client.Search(ctx, in.Query, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

func scheduleOp(ctx context.Context, in scheduleInput, emit func(Show) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	country := in.Country
	if country == "" {
		country = "US"
	}
	items, err := in.Client.Schedule(ctx, country, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver: pure string functions, no network ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty tvmaze reference")
	}
	return "show", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "show":
		return "https://www.tvmaze.com/shows/" + id, nil
	default:
		return "", errs.Usage("tvmaze has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}
