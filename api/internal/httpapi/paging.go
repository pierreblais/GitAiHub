package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

const (
	defaultPageSize = 30
	maxPageSize     = 100
)

// query returns a query parameter by its JSON (lowerCamelCase) or proto
// (snake_case) name.
func query(r *http.Request, camel, snake string) string {
	if v := r.URL.Query().Get(camel); v != "" {
		return v
	}
	return r.URL.Query().Get(snake)
}

func pageSize(r *http.Request) (int, error) {
	s := query(r, "pageSize", "page_size")
	if s == "" {
		return defaultPageSize, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%w: page_size must be a non-negative integer", source.ErrInvalidArgument)
	}
	if n == 0 {
		return defaultPageSize, nil
	}
	return min(n, maxPageSize), nil
}

// pageToken is the opaque cursor of paginated lists (AIP-158). Clients must
// not parse it; it is base64 JSON so the format can change.
type pageToken struct {
	Offset int    `json:"o,omitempty"`
	Time   int64  `json:"t,omitempty"`
	Key    string `json:"k,omitempty"`
}

func encodeToken(t pageToken) string {
	b, _ := json.Marshal(t)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeToken(r *http.Request) (pageToken, error) {
	var t pageToken
	s := query(r, "pageToken", "page_token")
	if s == "" {
		return t, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		err = json.Unmarshal(b, &t)
	}
	if err != nil || t.Offset < 0 {
		return t, fmt.Errorf("%w: invalid page_token", source.ErrInvalidArgument)
	}
	return t, nil
}
