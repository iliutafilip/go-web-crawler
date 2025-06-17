package sitemap

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"web-crawler/link_parser"
)

const xmlns = "http://sitemaps.org/schemas/sitemap/0.9"

type loc struct {
	Value string `xml:"loc"`
}	

type urlset struct {
	Urls []loc `xml:"url"`
	Xmlns string `xml:"xmlns,attr"`
}

func GetLinks(url string, maxDepth int) []string {
	
	return bfs(url, maxDepth)
}

type empty struct{}

func bfs(urlStr string, maxDepth int) []string {
	
	seen := make(map[string]empty)
	var q map[string]empty
	nq := map[string]empty{
		urlStr: {},
	}
	for range maxDepth+1 {
		q, nq = nq, make(map[string]empty)
		if len(q) == 0 {
			break
		}
		for link := range q {
			if _, ok := seen[link]; ok {
				continue
			}
			seen[link] = empty{}
			for _, otherLink := range get(link) {
				if _, ok := seen[otherLink]; !ok {
					nq[otherLink] = empty{}
				}
			}
		}
	}
	ret := make([]string, 0, len(seen))
	for link := range seen {
		ret = append(ret, link)
	}
	return ret
}


var client = &http.Client{
	Timeout: 10 * time.Second,
}

func get(urlStr string) []string {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		log.Printf("error creating request for %s: %v", urlStr, err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyCrawler/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GET error on specified url (%s): %v", urlStr, err)
		return nil
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		log.Printf("skipping non-HTML content at %s (Content-Type: %s)", urlStr, contentType)
		return nil
	}

	reqUrl := resp.Request.URL
	baseUrl := &url.URL{
		Scheme: reqUrl.Scheme,
		Host:   reqUrl.Host,
	}
	base := baseUrl.String()

	return filter(hrefs(resp.Body, base), withPrefix(base))
}

func hrefs(body io.Reader, base string) []string {
	links, err := link_parser.Parse(body)
	if err != nil {
		log.Printf("error parsing links: %v", err)
		return nil
	}
	var ret []string
	for _, l := range links {
		href := l.Href
		switch {
		case strings.HasPrefix(href, "/"):
			ret = append(ret, base+href)
		case strings.HasPrefix(href, "http"):
			u, err := url.Parse(href)
			if err != nil {
				continue
			}
			// strip fragments and queries
			// u.Fragment = ""
			// optionally strip query parameters:
			// u.RawQuery = ""
			ret = append(ret, u.String())
		default:
			// skip things like javascript:...
		}
	}
	return ret
}

func filter(links []string, keepFn func(string) bool ) []string {
	var ret []string

	for _, link := range links {
		if keepFn(link) {
			ret = append(ret, link)
		}
	}

	return ret
}

func withPrefix(prfx string) func(string) bool {
	return func(link string) bool {
		return strings.HasPrefix(link, prfx)
	}
}
