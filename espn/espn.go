// Package espn is the library behind the espn command line:
// the HTTP client, request shaping, and the typed data models for ESPN.
//
// The Client talks to site.api.espn.com, ESPN's unofficial public API.
// No API key is required. It sets a real User-Agent, paces requests to
// stay polite, and retries transient failures (429 and 5xx).
package espn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Host is the ESPN API host this client talks to.
const Host = "site.api.espn.com"

// BaseURL is the root every request is built from.
const BaseURL = "https://" + Host

// DefaultUserAgent identifies this client to ESPN.
const DefaultUserAgent = "espn-cli/0.1 (tamnd87@gmail.com)"

// validLeagues maps league code to the sport segment used in the API path.
var validLeagues = map[string]string{
	"nfl":   "football",
	"mlb":   "baseball",
	"nba":   "basketball",
	"nhl":   "hockey",
	"eng.1": "soccer",
}

// sportForLeague returns the sport name for a given league code.
// If the league is unknown it returns an empty string.
func sportForLeague(league string) string {
	return validLeagues[league]
}

// Config holds the client settings.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		Rate:      300 * time.Millisecond,
		Timeout:   15 * time.Second,
		Retries:   3,
		UserAgent: DefaultUserAgent,
	}
}

// Client talks to the ESPN API over HTTP.
type Client struct {
	HTTP      *http.Client
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int

	last time.Time
}

// NewClient returns a Client with DefaultConfig settings.
func NewClient() *Client {
	cfg := DefaultConfig()
	return &Client{
		HTTP:      &http.Client{Timeout: cfg.Timeout},
		BaseURL:   cfg.BaseURL,
		UserAgent: cfg.UserAgent,
		Rate:      cfg.Rate,
		Retries:   cfg.Retries,
	}
}

// Get fetches url and returns the response body. It paces and retries
// according to the client settings. The body is read and closed here.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
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

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
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
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// sportLeaguePath returns "sport/league" and validates the league.
func (c *Client) sportLeaguePath(sport, league string) (string, error) {
	if league == "" {
		league = "nfl"
	}
	if sport == "" {
		s, ok := validLeagues[league]
		if !ok {
			return "", fmt.Errorf("unknown league %q; valid: nfl, mlb, nba, nhl, eng.1", league)
		}
		sport = s
	}
	return sport + "/" + league, nil
}

// --- Wire types (ESPN API response shapes) ---

type teamsResp struct {
	Sports []struct {
		Leagues []struct {
			Teams []struct {
				Team struct {
					ID           string `json:"id"`
					UID          string `json:"uid"`
					Slug         string `json:"slug"`
					DisplayName  string `json:"displayName"`
					Abbreviation string `json:"abbreviation"`
					Location     string `json:"location"`
					Logos        []struct {
						Href string `json:"href"`
					} `json:"logos"`
					Links []struct {
						Href string `json:"href"`
					} `json:"links"`
				} `json:"team"`
			} `json:"teams"`
		} `json:"leagues"`
	} `json:"sports"`
}

type scoreboardResp struct {
	Events []struct {
		ID        string `json:"id"`
		UID       string `json:"uid"`
		Date      string `json:"date"`
		ShortName string `json:"shortName"`
		Season    struct {
			Year int `json:"year"`
			Type int `json:"type"`
		} `json:"season"`
		Status struct {
			Type struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Completed   bool   `json:"completed"`
			} `json:"type"`
		} `json:"status"`
		Competitions []struct {
			Competitors []struct {
				ID       string `json:"id"`
				HomeAway string `json:"homeAway"`
				Score    string `json:"score"`
				Team     struct {
					DisplayName string `json:"displayName"`
				} `json:"team"`
			} `json:"competitors"`
		} `json:"competitions"`
	} `json:"events"`
}

type newsResp struct {
	Articles []struct {
		DataSourceIdentifier string `json:"dataSourceIdentifier"`
		Headline             string `json:"headline"`
		Description          string `json:"description"`
		Byline               string `json:"byline"`
		Type                 string `json:"type"`
		Premium              bool   `json:"premium"`
		PublishedDate        string `json:"publishedDate"`
		Images               []struct {
			URL string `json:"url"`
		} `json:"images"`
		Links struct {
			Web struct {
				Href string `json:"href"`
			} `json:"web"`
		} `json:"links"`
	} `json:"articles"`
}

type standingsResp struct {
	Children []struct {
		Name    string `json:"name"`
		Entries []struct {
			Team struct {
				DisplayName string `json:"displayName"`
			} `json:"team"`
			Stats []struct {
				Name         string  `json:"name"`
				DisplayName  string  `json:"displayName"`
				Value        float64 `json:"value"`
				DisplayValue string  `json:"displayValue"`
			} `json:"stats"`
		} `json:"entries"`
	} `json:"children"`
}

// --- Public record types ---

// Team is one team entry.
type Team struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Location     string `json:"location"`
	Slug         string `json:"slug"`
	LogoURL      string `json:"logo_url,omitempty"`
	WebURL       string `json:"web_url,omitempty"`
}

// Game is one scoreboard entry.
type Game struct {
	ID        string `json:"id"`
	Date      string `json:"date"`
	ShortName string `json:"short_name"`
	HomeTeam  string `json:"home_team"`
	HomeScore string `json:"home_score"`
	AwayTeam  string `json:"away_team"`
	AwayScore string `json:"away_score"`
	Status    string `json:"status"`
	Season    int    `json:"season"`
}

// Article is one news article.
type Article struct {
	Headline      string `json:"headline"`
	Description   string `json:"description"`
	Byline        string `json:"byline,omitempty"`
	PublishedDate string `json:"published_date"`
	Premium       bool   `json:"premium"`
	WebURL        string `json:"web_url,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
}

// Standing is one team's standing in the league.
type Standing struct {
	Team       string  `json:"team"`
	Wins       float64 `json:"wins"`
	Losses     float64 `json:"losses"`
	Ties       float64 `json:"ties"`
	WinPct     string  `json:"win_pct"`
	Division   string  `json:"division"`
	Conference string  `json:"conference"`
}

// --- API methods ---

// ListTeams fetches all teams for the given sport/league.
func (c *Client) ListTeams(ctx context.Context, sport, league string) ([]*Team, error) {
	path, err := c.sportLeaguePath(sport, league)
	if err != nil {
		return nil, err
	}
	url := c.BaseURL + "/apis/site/v2/sports/" + path + "/teams"
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp teamsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse teams: %w", err)
	}
	var teams []*Team
	for _, sp := range resp.Sports {
		for _, lg := range sp.Leagues {
			for _, t := range lg.Teams {
				team := &Team{
					ID:           t.Team.ID,
					Name:         t.Team.DisplayName,
					Abbreviation: t.Team.Abbreviation,
					Location:     t.Team.Location,
					Slug:         t.Team.Slug,
				}
				if len(t.Team.Logos) > 0 {
					team.LogoURL = t.Team.Logos[0].Href
				}
				if len(t.Team.Links) > 0 {
					team.WebURL = t.Team.Links[0].Href
				}
				teams = append(teams, team)
			}
		}
	}
	return teams, nil
}

// ListScores fetches current/recent scoreboard for the given sport/league.
func (c *Client) ListScores(ctx context.Context, sport, league string) ([]*Game, error) {
	path, err := c.sportLeaguePath(sport, league)
	if err != nil {
		return nil, err
	}
	url := c.BaseURL + "/apis/site/v2/sports/" + path + "/scoreboard"
	return c.fetchGames(ctx, url)
}

// ListSchedule fetches the schedule (same endpoint as scoreboard).
func (c *Client) ListSchedule(ctx context.Context, sport, league string) ([]*Game, error) {
	return c.ListScores(ctx, sport, league)
}

func (c *Client) fetchGames(ctx context.Context, url string) ([]*Game, error) {
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp scoreboardResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse scoreboard: %w", err)
	}
	var games []*Game
	for _, e := range resp.Events {
		g := &Game{
			ID:        e.ID,
			Date:      e.Date,
			ShortName: e.ShortName,
			Status:    e.Status.Type.Description,
			Season:    e.Season.Year,
		}
		if len(e.Competitions) > 0 {
			for _, comp := range e.Competitions[0].Competitors {
				if comp.HomeAway == "home" {
					g.HomeTeam = comp.Team.DisplayName
					g.HomeScore = comp.Score
				} else {
					g.AwayTeam = comp.Team.DisplayName
					g.AwayScore = comp.Score
				}
			}
		}
		games = append(games, g)
	}
	return games, nil
}

// ListNews fetches the latest news for the given sport/league.
func (c *Client) ListNews(ctx context.Context, sport, league string, limit int) ([]*Article, error) {
	path, err := c.sportLeaguePath(sport, league)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/apis/site/v2/sports/%s/news?limit=%d", c.BaseURL, path, limit)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp newsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse news: %w", err)
	}
	var articles []*Article
	for _, a := range resp.Articles {
		art := &Article{
			Headline:      a.Headline,
			Description:   a.Description,
			Byline:        a.Byline,
			PublishedDate: a.PublishedDate,
			Premium:       a.Premium,
			WebURL:        a.Links.Web.Href,
		}
		if len(a.Images) > 0 {
			art.ImageURL = a.Images[0].URL
		}
		articles = append(articles, art)
	}
	return articles, nil
}

// ListStandings fetches current standings for the given sport/league.
func (c *Client) ListStandings(ctx context.Context, sport, league string) ([]*Standing, error) {
	path, err := c.sportLeaguePath(sport, league)
	if err != nil {
		return nil, err
	}
	url := c.BaseURL + "/apis/v2/sports/" + path + "/standings"
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp standingsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse standings: %w", err)
	}
	var standings []*Standing
	for _, child := range resp.Children {
		conf := child.Name
		for _, entry := range child.Entries {
			s := &Standing{
				Team:       entry.Team.DisplayName,
				Conference: conf,
			}
			for _, stat := range entry.Stats {
				switch stat.Name {
				case "wins":
					s.Wins = stat.Value
				case "losses":
					s.Losses = stat.Value
				case "ties":
					s.Ties = stat.Value
				case "winPercent":
					s.WinPct = stat.DisplayValue
				case "division":
					s.Division = stat.DisplayValue
				}
			}
			standings = append(standings, s)
		}
	}
	return standings, nil
}
