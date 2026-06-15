package espn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDomainInfo checks the domain registration metadata.
func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "espn" {
		t.Errorf("Scheme = %q, want espn", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "espn" {
		t.Errorf("Identity.Binary = %q, want espn", info.Identity.Binary)
	}
}

func newTestClient(ts *httptest.Server) *Client {
	c := NewClient()
	c.BaseURL = ts.URL
	c.Rate = 0
	c.Retries = 0
	return c
}

func TestListTeams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "teams") {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sports": []map[string]interface{}{{
				"leagues": []map[string]interface{}{{
					"teams": []map[string]interface{}{
						{
							"team": map[string]interface{}{
								"id":          "1",
								"displayName": "Arizona Cardinals",
								"abbreviation": "ARI",
								"location":    "Arizona",
								"slug":        "arizona-cardinals",
							},
						},
						{
							"team": map[string]interface{}{
								"id":          "2",
								"displayName": "Atlanta Falcons",
								"abbreviation": "ATL",
								"location":    "Atlanta",
								"slug":        "atlanta-falcons",
							},
						},
					},
				}},
			}},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	teams, err := c.ListTeams(context.Background(), "football", "nfl")
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 2 {
		t.Fatalf("got %d teams, want 2", len(teams))
	}
	if teams[0].Name != "Arizona Cardinals" {
		t.Errorf("teams[0].Name = %q, want Arizona Cardinals", teams[0].Name)
	}
	if teams[0].Abbreviation != "ARI" {
		t.Errorf("teams[0].Abbreviation = %q, want ARI", teams[0].Abbreviation)
	}
}

func TestListTeamsDefaultLeague(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// should default to football/nfl
		if !strings.Contains(r.URL.Path, "football/nfl") {
			t.Errorf("unexpected path %q, want football/nfl", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sports": []map[string]interface{}{{
				"leagues": []map[string]interface{}{{
					"teams": []map[string]interface{}{},
				}},
			}},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.ListTeams(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestListScores(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "scoreboard") {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []map[string]interface{}{{
				"id":        "401671843",
				"date":      "2025-01-19T21:00Z",
				"shortName": "NE @ SEA",
				"season":    map[string]interface{}{"year": 2024, "type": 2},
				"status": map[string]interface{}{
					"type": map[string]interface{}{
						"name":        "STATUS_FINAL",
						"description": "Final",
						"completed":   true,
					},
				},
				"competitions": []map[string]interface{}{{
					"competitors": []map[string]interface{}{
						{
							"id":       "26",
							"homeAway": "home",
							"score":    "24",
							"team":     map[string]interface{}{"displayName": "Seattle Seahawks"},
						},
						{
							"id":       "17",
							"homeAway": "away",
							"score":    "14",
							"team":     map[string]interface{}{"displayName": "New England Patriots"},
						},
					},
				}},
			}},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	games, err := c.ListScores(context.Background(), "football", "nfl")
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 {
		t.Fatalf("got %d games, want 1", len(games))
	}
	g := games[0]
	if g.ID != "401671843" {
		t.Errorf("ID = %q, want 401671843", g.ID)
	}
	if g.HomeTeam != "Seattle Seahawks" {
		t.Errorf("HomeTeam = %q, want Seattle Seahawks", g.HomeTeam)
	}
	if g.HomeScore != "24" {
		t.Errorf("HomeScore = %q, want 24", g.HomeScore)
	}
	if g.AwayTeam != "New England Patriots" {
		t.Errorf("AwayTeam = %q, want New England Patriots", g.AwayTeam)
	}
	if g.Status != "Final" {
		t.Errorf("Status = %q, want Final", g.Status)
	}
	if g.Season != 2024 {
		t.Errorf("Season = %d, want 2024", g.Season)
	}
}

func TestListNews(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "news") {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"articles": []map[string]interface{}{{
				"headline":      "Knicks' victory takes them off the list",
				"description":   "A description.",
				"byline":        "ESPN Staff",
				"publishedDate": "2024-01-20T00:00:00Z",
				"premium":       false,
				"images":        []map[string]interface{}{{"url": "https://example.com/img.jpg"}},
				"links": map[string]interface{}{
					"web": map[string]interface{}{"href": "https://espn.com/article/1"},
				},
			}},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	articles, err := c.ListNews(context.Background(), "basketball", "nba", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 1 {
		t.Fatalf("got %d articles, want 1", len(articles))
	}
	a := articles[0]
	if a.Headline != "Knicks' victory takes them off the list" {
		t.Errorf("Headline = %q", a.Headline)
	}
	if a.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("ImageURL = %q", a.ImageURL)
	}
	if a.WebURL != "https://espn.com/article/1" {
		t.Errorf("WebURL = %q", a.WebURL)
	}
}

func TestListStandings(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "standings") {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"children": []map[string]interface{}{{
				"name": "AFC",
				"entries": []map[string]interface{}{{
					"team": map[string]interface{}{"displayName": "Kansas City Chiefs"},
					"stats": []map[string]interface{}{
						{"name": "wins", "value": 14, "displayValue": "14"},
						{"name": "losses", "value": 3, "displayValue": "3"},
						{"name": "ties", "value": 0, "displayValue": "0"},
						{"name": "winPercent", "value": 0.824, "displayValue": ".824"},
					},
				}},
			}},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	standings, err := c.ListStandings(context.Background(), "football", "nfl")
	if err != nil {
		t.Fatal(err)
	}
	if len(standings) != 1 {
		t.Fatalf("got %d standings entries, want 1", len(standings))
	}
	s := standings[0]
	if s.Team != "Kansas City Chiefs" {
		t.Errorf("Team = %q, want Kansas City Chiefs", s.Team)
	}
	if s.Wins != 14 {
		t.Errorf("Wins = %v, want 14", s.Wins)
	}
	if s.Conference != "AFC" {
		t.Errorf("Conference = %q, want AFC", s.Conference)
	}
}

func TestSportForLeague(t *testing.T) {
	cases := []struct {
		league string
		sport  string
	}{
		{"nfl", "football"},
		{"nba", "basketball"},
		{"mlb", "baseball"},
		{"nhl", "hockey"},
		{"eng.1", "soccer"},
	}
	for _, tc := range cases {
		got := sportForLeague(tc.league)
		if got != tc.sport {
			t.Errorf("sportForLeague(%q) = %q, want %q", tc.league, got, tc.sport)
		}
	}
}

func TestInvalidLeague(t *testing.T) {
	c := NewClient()
	c.Rate = 0
	_, err := c.ListTeams(context.Background(), "", "badleague")
	if err == nil {
		t.Error("expected error for unknown league, got nil")
	}
}
