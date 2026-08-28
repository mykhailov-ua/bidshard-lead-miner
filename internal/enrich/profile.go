package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/sources/forum"
)

// ProfileEnricher fetches forum/github/reddit profiles when leads lack direct contacts.
type ProfileEnricher struct {
	githubToken string
	githubBase  string
	redditBase  string
	forum       *forum.Fetcher
	client      *http.Client
}

func NewProfileEnricher(cfg config.Config) *ProfileEnricher {
	if !cfg.ProfileEnrichEnabled {
		return nil
	}
	return &ProfileEnricher{
		githubToken: strings.TrimSpace(cfg.GitHubToken),
		forum:       forum.NewFetcherForSource(cfg, "forum"),
		client:      httpclient.CrawlClient(cfg.HTTPTimeout, cfg.ProxyURLsForSource("forum"), "profile"),
	}
}

// MergeContacts enriches identity-only leads with email/telegram from public profiles.
func (p *ProfileEnricher) MergeContacts(ctx context.Context, source string, contacts []extract.Contact, username, forumUID string) []extract.Contact {
	if p == nil || extract.HasReachableContact(contacts) {
		return contacts
	}

	var merged []extract.Contact
	if host := entity.ForumHostFromSource(source); host != "" {
		user := strings.TrimSpace(username)
		if user == "" {
			user = forumUserFromContacts(contacts)
		}
		if user != "" {
			if html, err := p.fetchForumProfile(ctx, host, user, forumUID); err == nil && html != "" {
				merged = extract.MergeContacts(merged, extract.Extract(html).Contacts)
			}
		}
	}

	for _, c := range contacts {
		switch c.Type {
		case "github":
			login := strings.TrimSpace(c.Value)
			if login == "" {
				continue
			}
			if text, err := p.fetchGitHubUser(ctx, login); err == nil && text != "" {
				merged = extract.MergeContacts(merged, extract.Extract(text).Contacts)
			}
		case "reddit":
			if user := redditUserFromContact(c.Value); user != "" {
				if text, err := p.fetchRedditAbout(ctx, user); err == nil && text != "" {
					merged = extract.MergeContacts(merged, extract.Extract(text).Contacts)
				}
			}
		}
	}

	if len(merged) == 0 {
		return contacts
	}
	return extract.MergeContacts(contacts, merged)
}

func (p *ProfileEnricher) fetchForumProfile(ctx context.Context, host, user, uid string) (string, error) {
	rawURL := forumProfileURL(host, user, uid)
	if rawURL == "" || p.forum == nil {
		return "", fmt.Errorf("forum profile url missing")
	}
	return p.forum.Get(ctx, rawURL)
}

func forumProfileURL(host, user, uid string) string {
	host = entity.NormalizeForumHost(host)
	user = strings.TrimSpace(user)
	if host == "" || user == "" {
		return ""
	}
	user = strings.ToLower(user)
	uid = strings.TrimSpace(uid)
	if uid != "" {
		return "https://" + host + "/members/" + user + "." + uid + "/"
	}
	return "https://" + host + "/members/" + user + "/"
}

func forumUserFromContacts(contacts []extract.Contact) string {
	for _, c := range contacts {
		if c.Type != "forum_user" {
			continue
		}
		user := strings.TrimSpace(c.Value)
		user = strings.TrimPrefix(user, "forum:user/")
		user = strings.TrimPrefix(user, "warrior:user/")
		if user != "" {
			return user
		}
	}
	return ""
}

func redditUserFromContact(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(value), "reddit:"))
	value = strings.TrimPrefix(value, "u/")
	value = strings.TrimPrefix(value, "/u/")
	return strings.TrimSpace(value)
}

func (p *ProfileEnricher) fetchGitHubUser(ctx context.Context, login string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return "", fmt.Errorf("empty github login")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.githubAPIURL(login), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if p.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.githubToken)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var doc struct {
		Email string `json:"email"`
		Blog  string `json:"blog"`
		Bio   string `json:"bio"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.Join([]string{doc.Bio, doc.Blog, doc.Email}, "\n")), nil
}

func (p *ProfileEnricher) fetchRedditAbout(ctx context.Context, user string) (string, error) {
	user = strings.TrimSpace(user)
	if user == "" {
		return "", fmt.Errorf("empty reddit user")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.redditAboutURL(user), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "lead-intent-processor/1.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reddit about status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var doc struct {
		Data struct {
			Subreddit struct {
				PublicDescription string `json:"public_description"`
			} `json:"subreddit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	return strings.TrimSpace(doc.Data.Subreddit.PublicDescription), nil
}

func (p *ProfileEnricher) githubAPIURL(login string) string {
	base := strings.TrimSuffix(strings.TrimSpace(p.githubBase), "/")
	if base == "" {
		base = "https://api.github.com"
	}
	return base + "/users/" + login
}

func (p *ProfileEnricher) redditAboutURL(user string) string {
	base := strings.TrimSuffix(strings.TrimSpace(p.redditBase), "/")
	if base == "" {
		base = "https://old.reddit.com"
	}
	return base + "/user/" + user + "/about.json"
}

// ForumProfileURL builds a XenForo-style member page URL for tests and docs.
func ForumProfileURL(host, user, uid string) string {
	return forumProfileURL(host, user, uid)
}

// NewProfileEnricherForTest wires fetchers for httptest servers.
func NewProfileEnricherForTest(forumFetcher *forum.Fetcher, client *http.Client, githubToken string) *ProfileEnricher {
	if forumFetcher == nil && client == nil {
		return nil
	}
	if client == nil {
		client = httpclient.Shared(5 * time.Second)
	}
	return &ProfileEnricher{
		githubToken: githubToken,
		forum:       forumFetcher,
		client:      client,
	}
}

// SetTestAPIBases overrides GitHub and Reddit API roots (httptest only).
func (p *ProfileEnricher) SetTestAPIBases(githubBase, redditBase string) {
	if p == nil {
		return
	}
	p.githubBase = githubBase
	p.redditBase = redditBase
}
