package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

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

func main() {
	urlFlag := flag.String("url", "https://example.com", "the url that you want to crawl")
	maxDepth := flag.Int("depth", 3, "the maximum number of links deep to traverse")
	flag.Parse()

	links := bfs(*urlFlag, *maxDepth)

	toXml := urlset {
		Xmlns: xmlns,
	}
	
	for _, link := range links {
		toXml.Urls = append(toXml.Urls, loc{link})
	}
	fmt.Print(xml.Header)
	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("","  ")
	if err := enc.Encode(toXml); err != nil {
		log.Fatalf("error on encoding xml: %v", err)
	}
	fmt.Println()
}

type empty struct{}

func bfs(urlStr string, maxDepth int) []string {
	
	seen := make(map[string]empty)
	var q map[string]empty
	nq := map[string]empty{
		urlStr: empty{},
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


func get(urlStr string) []string {
	resp, err := http.Get(urlStr)
	if err != nil {
		log.Fatalf("GET error on specified url (%s): %v", *&urlStr, err)
	}
	defer resp.Body.Close()

	// if url starts with '/', assume that it's the same domain
	
	// get the root site URL and build base URL
	reqUrl := resp.Request.URL
	baseUrl := &url.URL{
		Scheme: reqUrl.Scheme,
		Host: reqUrl.Host,
	}
	base := baseUrl.String()
  
	return filter(hrefs(resp.Body, base), withPrefix(base))
} 


func hrefs(body io.Reader, base string) []string {
	
	links, err := link_parser.Parse(body)
	if err != nil {
		log.Fatalf("error parsing links: %v",  err)
	}
	var ret[] string
	for _, l := range links {
		switch {
		case strings.HasPrefix(l.Href, "/"):
			ret = append(ret, base + l.Href)
		case strings.HasPrefix(l.Href, "http"):
			ret = append(ret, l.Href)
		default:
			
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














