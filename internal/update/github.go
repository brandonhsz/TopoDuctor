package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	defaultOwner = "brandonhsz"
	defaultRepo  = "TopoDuctor"
)

// ReleaseTag names the latest published release tag (e.g. "v1.2.3").
type ReleaseTag struct {
	Tag string
	URL string
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// FetchLatestRelease queries GitHub's latest release API.
func FetchLatestRelease(ctx context.Context, owner, repo string) (ReleaseTag, error) {
	if owner == "" {
		owner = defaultOwner
	}
	if repo == "" {
		repo = defaultRepo
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo), nil)
	if err != nil {
		return ReleaseTag{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "topoductor-update-check")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ReleaseTag{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ReleaseTag{}, fmt.Errorf("GitHub API: %s", resp.Status)
	}

	var body githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ReleaseTag{}, err
	}
	tag := strings.TrimSpace(body.TagName)
	if tag == "" {
		return ReleaseTag{}, fmt.Errorf("respuesta sin tag_name")
	}
	return ReleaseTag{Tag: tag, URL: body.HTMLURL}, nil
}

// Normalize semver for golang.org/x/mod/semver (must start with "v").
func Normalize(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" || strings.EqualFold(v, "dev") {
		return "v0.0.0"
	}
	i := strings.IndexByte(v, '-')
	if i >= 0 {
		v = v[:i]
	}
	if !semver.IsValid("v" + v) {
		return "v0.0.0"
	}
	return "v" + v
}

// IsNewer reports true if latest is strictly greater than current (both semver-normalized).
func IsNewer(current, latest string) bool {
	c := Normalize(current)
	l := Normalize(latest)
	return semver.Compare(c, l) < 0
}
