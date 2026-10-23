// Package registry finds server-streampack's newest release in its image registry.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// TagsURL lists the image's tags; it needs no sign-in.
const TagsURL = "https://nexus.streampack.dev/v2/images/server-streampack/tags/list"

var release = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// Latest returns the highest x.y.z tag at url, ignoring "latest" and suffixed tags such as 1.2.1a.
func Latest(ctx context.Context, client *http.Client, url string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s answered %s", url, response.Status)
	}
	var list struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
		return "", err
	}
	return Highest(list.Tags)
}

// Highest returns the highest x.y.z among tags.
func Highest(tags []string) (string, error) {
	best := ""
	for _, tag := range tags {
		if IsRelease(tag) && (best == "" || Less(best, tag)) {
			best = tag
		}
	}
	if best == "" {
		return "", fmt.Errorf("no release among %d tags", len(tags))
	}
	return best, nil
}

// IsRelease says whether tag is a release, x.y.z, and not latest or a suffixed tag such as 1.2.1a.
func IsRelease(tag string) bool { return release.MatchString(tag) }

// Less says whether release a comes before release b, comparing each part as a number: 1.9.0
// comes before 1.10.0. Both must be releases (IsRelease).
func Less(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range 3 {
		x, _ := strconv.Atoi(as[i])
		y, _ := strconv.Atoi(bs[i])
		if x != y {
			return x < y
		}
	}
	return false
}
