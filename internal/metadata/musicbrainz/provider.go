package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/temic/go-music/internal/metadata/model"
)

const (
	apiBaseURL  = "https://musicbrainz.org/ws/2"
	coverAPIURL = "https://coverartarchive.org/release"
)

// Provider fetches album metadata from MusicBrainz and Cover Art Archive.
type Provider struct {
	client    *http.Client
	userAgent string
	apiBase   string

	mu          sync.Mutex
	lastRequest time.Time
}

// New creates a MusicBrainz metadata provider.
func New(userAgent string) *Provider {
	return NewWithAPIBase(apiBaseURL, userAgent)
}

// NewWithAPIBase creates a provider with a custom API base URL (for tests).
func NewWithAPIBase(apiBase, userAgent string) *Provider {
	if strings.TrimSpace(userAgent) == "" {
		userAgent = "GoMusic/1.0 (https://github.com/temic/go-music)"
	}
	if strings.TrimSpace(apiBase) == "" {
		apiBase = apiBaseURL
	}

	return &Provider{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		userAgent: userAgent,
		apiBase:   strings.TrimRight(apiBase, "/"),
	}
}

func (p *Provider) Name() string {
	return "musicbrainz"
}

func (p *Provider) SearchAlbum(ctx context.Context, query model.SearchQuery) (*model.Result, error) {
	p.throttle()

	release, err := p.searchRelease(ctx, query)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, nil
	}

	result := &model.Result{
		MusicBrainzID: release.ID,
		Title:         firstNonEmpty(release.Title, query.Album),
		Artist:        firstNonEmpty(release.Artist(), query.Artist),
		Year:          release.Year(),
		Genres:        release.Genres(),
		Description:   release.Disambiguation,
		CoverImageURL: fmt.Sprintf("%s/%s/front", coverAPIURL, release.ID),
	}

	return result, nil
}

func (p *Provider) throttle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if wait := time.Second - time.Since(p.lastRequest); wait > 0 {
		time.Sleep(wait)
	}
	p.lastRequest = time.Now()
}

func (p *Provider) searchRelease(ctx context.Context, query model.SearchQuery) (*release, error) {
	terms := []string{
		fmt.Sprintf(`artist:"%s"`, escapeQuery(query.Artist)),
		fmt.Sprintf(`release:"%s"`, escapeQuery(query.Album)),
	}
	if query.Year > 0 {
		terms = append(terms, fmt.Sprintf("date:%d", query.Year))
	}

	endpoint := p.apiBase + "/release?fmt=json&limit=5&query=" + url.QueryEscape(strings.Join(terms, " AND "))
	body, err := p.get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var response struct {
		Releases []release `json:"releases"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if len(response.Releases) == 0 {
		return nil, nil
	}

	return &response.Releases[0], nil
}

func (p *Provider) get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

type release struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Disambiguation string `json:"disambiguation"`
	Date           string `json:"date"`
	ArtistCredit   []struct {
		Name    string `json:"name"`
		Artist  struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

func (r release) Artist() string {
	if len(r.ArtistCredit) == 0 {
		return ""
	}
	if name := strings.TrimSpace(r.ArtistCredit[0].Artist.Name); name != "" {
		return name
	}
	return strings.TrimSpace(r.ArtistCredit[0].Name)
}

func (r release) Year() int {
	if len(r.Date) < 4 {
		return 0
	}
	year, err := strconv.Atoi(r.Date[:4])
	if err != nil {
		return 0
	}
	return year
}

func (r release) Genres() []string {
	if len(r.Tags) == 0 {
		return nil
	}

	genres := make([]string, 0, len(r.Tags))
	seen := make(map[string]struct{}, len(r.Tags))
	for _, tag := range r.Tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		genres = append(genres, name)
	}
	return genres
}

func escapeQuery(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
