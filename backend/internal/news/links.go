package news

import (
	"fmt"
	"strings"
)

type link struct {
	Href string `json:"href"`
}

// pageLinks builds HAL links for one page of a filtered list. path is the
// request path, so the links stay valid wherever the router is mounted.
// prev and next are absent when there is no such page. Defaults are left out
// of the hrefs: offset 0 never appears, and limit appears only when the caller
// sent one (includeLimit).
func pageLinks(path string, params ListParams, total int, includeLimit bool) map[string]link {
	href := func(offset int) link {
		var query []string
		if includeLimit {
			query = append(query, fmt.Sprintf("limit=%d", params.Limit))
		}
		if offset > 0 {
			query = append(query, fmt.Sprintf("offset=%d", offset))
		}
		if params.Category != "" {
			query = append(query, "category="+params.Category)
		}
		if len(query) == 0 {
			return link{Href: path}
		}
		return link{Href: path + "?" + strings.Join(query, "&")}
	}

	lastOffset := 0
	if total > 0 {
		lastOffset = (total - 1) / params.Limit * params.Limit
	}

	links := map[string]link{
		"self":  href(params.Offset),
		"first": href(0),
		"last":  href(lastOffset),
	}
	if params.Offset > 0 {
		links["prev"] = href(min(max(0, params.Offset-params.Limit), lastOffset))
	}
	if params.Offset+params.Limit < total {
		links["next"] = href(params.Offset + params.Limit)
	}
	return links
}
