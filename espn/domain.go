package espn

import (
	"context"

	"github.com/tamnd/any-cli/kit"
)

// domain.go exposes espn as a kit Domain. A host (ant) enables it with
// a single blank import:
//
//	import _ "github.com/tamnd/espn-cli/espn"
//
// The same Domain also builds the standalone espn binary (see cli.NewApp),
// so the binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the ESPN driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "espn",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "espn",
			Short:  "A command line for ESPN sports data.",
			Long: `A command line for ESPN sports data.

espn reads public ESPN data over plain HTTPS from site.api.espn.com,
shapes it into clean records, and prints output that pipes into the rest
of your tools. No API key, nothing to run alongside it.

Supported leagues: nfl, mlb, nba, nhl, eng.1 (EPL)`,
			Site: Host,
			Repo: "https://github.com/tamnd/espn-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "teams", Group: "read", List: true,
		Summary: "List teams in a sport/league (--sport=football --league=nfl)"}, listTeams)

	kit.Handle(app, kit.OpMeta{Name: "scores", Group: "read", List: true,
		Summary: "Current scoreboard for a sport/league"}, listScores)

	kit.Handle(app, kit.OpMeta{Name: "news", Group: "read", List: true,
		Summary: "Latest news articles for a sport/league"}, listNews)

	kit.Handle(app, kit.OpMeta{Name: "standings", Group: "read", List: true,
		Summary: "Current standings for a sport/league"}, listStandings)

	kit.Handle(app, kit.OpMeta{Name: "schedule", Group: "read", List: true,
		Summary: "Upcoming schedule for a sport/league"}, listSchedule)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
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
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// --- input structs ---

type leagueInput struct {
	Sport  string  `kit:"flag" help:"sport name (football, basketball, baseball, hockey, soccer)"`
	League string  `kit:"flag" help:"league code (nfl, nba, mlb, nhl, eng.1)"`
	Client *Client `kit:"inject"`
}

type newsInput struct {
	Sport  string  `kit:"flag" help:"sport name (football, basketball, baseball, hockey, soccer)"`
	League string  `kit:"flag" help:"league code (nfl, nba, mlb, nhl, eng.1)"`
	Limit  int     `kit:"flag,inherit" help:"max articles"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listTeams(ctx context.Context, in leagueInput, emit func(*Team) error) error {
	teams, err := in.Client.ListTeams(ctx, in.Sport, in.League)
	if err != nil {
		return err
	}
	for _, t := range teams {
		if err := emit(t); err != nil {
			return err
		}
	}
	return nil
}

func listScores(ctx context.Context, in leagueInput, emit func(*Game) error) error {
	games, err := in.Client.ListScores(ctx, in.Sport, in.League)
	if err != nil {
		return err
	}
	for _, g := range games {
		if err := emit(g); err != nil {
			return err
		}
	}
	return nil
}

func listNews(ctx context.Context, in newsInput, emit func(*Article) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	articles, err := in.Client.ListNews(ctx, in.Sport, in.League, limit)
	if err != nil {
		return err
	}
	for _, a := range articles {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func listStandings(ctx context.Context, in leagueInput, emit func(*Standing) error) error {
	standings, err := in.Client.ListStandings(ctx, in.Sport, in.League)
	if err != nil {
		return err
	}
	for _, s := range standings {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}

func listSchedule(ctx context.Context, in leagueInput, emit func(*Game) error) error {
	games, err := in.Client.ListSchedule(ctx, in.Sport, in.League)
	if err != nil {
		return err
	}
	for _, g := range games {
		if err := emit(g); err != nil {
			return err
		}
	}
	return nil
}
