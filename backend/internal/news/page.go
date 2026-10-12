package news

import (
	"errors"
	"fmt"
	"html/template"
	"strings"
)

const (
	ogStartMarker = "<!--og:start-->"
	ogEndMarker   = "<!--og:end-->"

	siteDescription = "A PvE guild for World of Warcraft: Forever on the US servers. Raiding, dungeons and community."
)

var headTemplate = template.Must(template.New("head").Parse(`
    <title>{{.TabTitle}}</title>
    <meta name="description" content="{{.Description}}" />
    <link rel="canonical" href="{{.URL}}" />
    <meta property="og:type" content="article" />
    <meta property="og:site_name" content="Fuzion" />
    <meta property="og:title" content="{{.Title}}" />
    <meta property="og:description" content="{{.Description}}" />
    <meta property="og:url" content="{{.URL}}" />
    <meta property="og:image" content="{{.Image}}" />{{if .IsSiteImage}}
    <meta property="og:image:width" content="1200" />
    <meta property="og:image:height" content="630" />
    <meta property="og:image:alt" content="The Fuzion logo, a gold atom with glowing blue orbs, in front of an ancient vault with a glowing doorway." />{{end}}
    <meta name="twitter:card" content="summary_large_image" />
    `))

type pageHead struct {
	TabTitle    string
	Title       string
	Description string
	URL         string
	Image       string
	IsSiteImage bool
}

// RenderPage returns the shell with its og block replaced by tags describing the post.
func (s *Service) RenderPage(shell string, post Post) (string, error) {
	head := pageHead{
		TabTitle:    post.Title + " | Fuzion",
		Title:       post.Title,
		Description: post.Excerpt,
		URL:         s.siteBaseURL + "/news/" + post.ID.String(),
		Image:       s.firstImageAbsoluteURL(post.Body),
	}
	if head.Description == "" {
		head.Description = siteDescription
	}
	if head.Image == "" {
		head.Image = s.siteBaseURL + "/og-image.jpg"
		head.IsSiteImage = true
	}

	var block strings.Builder
	if err := headTemplate.Execute(&block, head); err != nil {
		return "", fmt.Errorf("render head: %w", err)
	}

	start := strings.Index(shell, ogStartMarker)
	end := strings.Index(shell, ogEndMarker)
	if start < 0 || end < start {
		return "", errors.New("shell has no og block markers")
	}
	return shell[:start+len(ogStartMarker)] + block.String() + shell[end:], nil
}
