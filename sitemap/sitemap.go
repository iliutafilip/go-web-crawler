package sitemap

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"web-crawler/link_parser"
)

const xmlns = "http://sitemaps.org/schemas/sitemap/0.9"

type loc struct {
	Value string `xml:"loc"`
}

type urlset struct {
	Urls  []loc  `xml:"url"`
	Xmlns string `xml:"xmlns,attr"`
}

type Job struct {
	URL      string
	MaxDepth int
}

func workers(id int, jobs <-chan Job, results chan<- []Job, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		links := get(job.URL)
		if links != nil {
			newJobs := make([]Job, 0, len(links))
			for _, link := range links {
				if job.MaxDepth > 0 {
					newJobs = append(newJobs, Job{URL: link, MaxDepth: job.MaxDepth - 1})
				} else {
					newJobs = append(newJobs, Job{URL: link, MaxDepth: 0})
				}
			}
			results <- newJobs
		} else {
			log.Printf("worker %d: no links found for %s", id, job.URL)
			results <- nil
		}
	}
}

func GetLinks(url string, maxDepth int) []string {

	const numWorkers = 10

	seen := make(map[string]struct{})
	jobs := make(chan Job, 100)
	results := make(chan []Job, 100)

	var seenMu sync.Mutex
	var wg sync.WaitGroup

	for i := range numWorkers {
		wg.Add(1)
		go workers(i, jobs, results, &wg)
	}

	seen[url] = struct{}{}
	pendingJobs := 1
	jobs <- Job{URL: url, MaxDepth: maxDepth}

	for pendingJobs > 0 {
		newJobs := <-results
		pendingJobs--

		for _, job := range newJobs {
			seenMu.Lock()
			if _, ok := seen[job.URL]; !ok {
				seen[job.URL] = struct{}{}
				pendingJobs++
				jobs <- job
			}
			seenMu.Unlock()
		}
	}

	close(jobs)
	wg.Wait()
	close(results)

	ret := make([]string, 0, len(seen))
	for link := range seen {
		ret = append(ret, link)
	}
	return ret
	// return bfs(url, maxDepth)
}

type empty struct{}

func bfs(urlStr string, maxDepth int) []string {

	seen := make(map[string]empty)
	var q map[string]empty
	nq := map[string]empty{
		urlStr: {},
	}
	for range maxDepth + 1 {
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
	Timeout: 5 * time.Second,
}

func get(urlStr string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
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

func filter(links []string, keepFn func(string) bool) []string {
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
