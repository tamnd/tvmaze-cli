package tvmaze

import (
	"context"
	"strconv"
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

	// show: get a single show by TVMaze ID
	kit.Handle(app, kit.OpMeta{
		Name:    "show",
		Group:   "read",
		Single:  true,
		Summary: "Get a TV show by its TVMaze ID",
		Args:    []kit.Arg{{Name: "id", Help: "TVMaze show ID (e.g. 169)"}},
	}, showOp)

	// episodes: list all episodes for a show
	kit.Handle(app, kit.OpMeta{
		Name:    "episodes",
		Group:   "read",
		List:    true,
		Summary: "List all episodes for a show",
		Args:    []kit.Arg{{Name: "id", Help: "TVMaze show ID"}},
	}, episodesOp)

	// cast: list cast members for a show
	kit.Handle(app, kit.OpMeta{
		Name:    "cast",
		Group:   "read",
		List:    true,
		Summary: "List cast members for a show",
		Args:    []kit.Arg{{Name: "id", Help: "TVMaze show ID"}},
	}, castOp)

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

type showInput struct {
	ID     string  `kit:"arg"    help:"TVMaze show ID (e.g. 169)"`
	Client *Client `kit:"inject"`
}

type episodesInput struct {
	ID     string  `kit:"arg"    help:"TVMaze show ID"`
	Client *Client `kit:"inject"`
}

type castInput struct {
	ID     string  `kit:"arg"    help:"TVMaze show ID"`
	Client *Client `kit:"inject"`
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

func showOp(ctx context.Context, in showInput, emit func(*Show) error) error {
	id, err := strconv.Atoi(in.ID)
	if err != nil {
		return errs.Usage("show id must be a number, got %q", in.ID)
	}
	show, err := in.Client.GetShow(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	return emit(show)
}

func episodesOp(ctx context.Context, in episodesInput, emit func(*Episode) error) error {
	id, err := strconv.Atoi(in.ID)
	if err != nil {
		return errs.Usage("show id must be a number, got %q", in.ID)
	}
	episodes, err := in.Client.Episodes(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if len(episodes) == 0 {
		return errs.NotFound("no episodes found for show %d", id)
	}
	for i := range episodes {
		if err := emit(&episodes[i]); err != nil {
			return err
		}
	}
	return nil
}

func castOp(ctx context.Context, in castInput, emit func(*CastMember) error) error {
	id, err := strconv.Atoi(in.ID)
	if err != nil {
		return errs.Usage("show id must be a number, got %q", in.ID)
	}
	members, err := in.Client.Cast(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if len(members) == 0 {
		return errs.NotFound("no cast found for show %d", id)
	}
	for i := range members {
		if err := emit(&members[i]); err != nil {
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
