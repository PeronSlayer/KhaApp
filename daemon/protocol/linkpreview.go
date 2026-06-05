package protocol

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// LinkPreview contains minimal Open Graph metadata for one URL.
type LinkPreview struct {
	Title       string
	Description string
	ImageURL    string
}

// FetchLinkPreview retrieves a basic OG preview for an HTTP(S) URL.
func FetchLinkPreview(rawURL string) LinkPreview {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return LinkPreview{}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return LinkPreview{}
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) KhaApp/0.2")

	response, err := client.Do(request)
	if err != nil {
		return LinkPreview{}
	}
	defer response.Body.Close()

	document, err := html.Parse(response.Body)
	if err != nil {
		return LinkPreview{}
	}

	preview := LinkPreview{}
	var titleFallback string

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil || (preview.Title != "" && preview.Description != "" && preview.ImageURL != "" && titleFallback != "") {
			return
		}

		if node.Type == html.ElementNode {
			switch node.Data {
			case "meta":
				property := attrValue(node, "property")
				if property == "" {
					property = attrValue(node, "name")
				}
				content := attrValue(node, "content")
				switch strings.ToLower(property) {
				case "og:title":
					if preview.Title == "" {
						preview.Title = content
					}
				case "og:description":
					if preview.Description == "" {
						preview.Description = content
					}
				case "og:image":
					if preview.ImageURL == "" {
						preview.ImageURL = resolveRelativeURL(parsedURL, content)
					}
				}
			case "title":
				if titleFallback == "" && node.FirstChild != nil {
					titleFallback = strings.TrimSpace(node.FirstChild.Data)
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(document)
	if preview.Title == "" {
		preview.Title = titleFallback
	}

	return preview
}

func attrValue(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func resolveRelativeURL(baseURL *url.URL, value string) string {
	if value == "" {
		return ""
	}

	relative, err := url.Parse(value)
	if err != nil {
		return value
	}

	return baseURL.ResolveReference(relative).String()
}
