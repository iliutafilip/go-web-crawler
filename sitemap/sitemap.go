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
	URL   string
	Depth int
}

func GetLinks(startURL string, maxDepth int, maxJobCount int) []string {
	const numWorkers = 10

	var (
		seen     = map[string]struct{}{startURL: {}}
		seenMu   sync.Mutex
		jobCount = 1 
		countMu  sync.Mutex

		jobs    = make(chan Job, maxJobCount)
		results = make(chan []Job, maxJobCount)

		workerWg sync.WaitGroup
		resultWg sync.WaitGroup
	)

	for i := range numWorkers {
		workerWg.Add(1)
		go func(id int) {
			defer workerWg.Done()
			for job := range jobs {
				log.Printf("Worker %d processing job for URL: %s", id, job.URL)
				links := get(job.URL)

				var newJobs []Job
				for _, link := range links {
					newJobs = append(newJobs, Job{URL: link, Depth: job.Depth + 1})
				}
				results <- newJobs
				log.Printf("Worker %d finished job for URL: %s", id, job.URL)
			}
		}(i)
	}

	resultWg.Add(1)
	jobs <- Job{URL: startURL, Depth: 0}

	go func() {
		for newJobs := range results {
			for _, job := range newJobs {
				if job.Depth > maxDepth {
					continue
				}

				seenMu.Lock()
				if _, ok := seen[job.URL]; ok {
					seenMu.Unlock()
					continue
				}
				seen[job.URL] = struct{}{}
				seenMu.Unlock()

				countMu.Lock()
				if jobCount >= maxJobCount {
					countMu.Unlock()
					continue
				}
				jobCount++
				countMu.Unlock()

				resultWg.Add(1)
				jobs <- job
			}
			resultWg.Done()
		}
	}()

	go func() {
		resultWg.Wait()
		close(jobs)
	}()

	workerWg.Wait()
	close(results)

	final := make([]string, 0, len(seen))
	for link := range seen {
		final = append(final, link)
	}
	return final
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
			//strip fragments and queries
			u.Fragment = ""
			//optionally strip query parameters:
			u.RawQuery = ""
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
