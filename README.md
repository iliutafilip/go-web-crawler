# Web Crawler Sitemap Generator

A concurrent web crawler written in Go that builds an XML sitemap from a starting URL. It traverses links on the same domain up to a given depth and outputs a standards-compliant sitemap (`url-<domain>.xml`).

## Usage

### 1. Build

```bash
go build -o crawler
```

### 2. Run

```bash
./crawler -url=https://example.com -depth=2 -max-jobs=500
```

#### Flags:

| Flag         | Description                               | Default                   |
|--------------|-------------------------------------------|---------------------------|
| `-url`       | Starting URL for the crawl                | `https://gophercises.com` |
| `-depth`     | Maximum crawl depth                       | `2`                       |
| `-max-jobs`  | Maximum number of jobs (pages to process) | `500`                     |

### 3. Output

Generates an XML sitemap saved to:

```
output/url-<domain>.xml
```

## Sitemap Format

Generated sitemaps conform to the [sitemaps.org protocol](https://www.sitemaps.org/protocol.html):

```xml
<urlset xmlns="http://sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
  </url>
  ...
</urlset>
```

## Requirements

- Go 1.18+
- Internet connection

## Notes

- The crawler only follows links that Bblong to the same domain
- It strips URL fragments (`#...`) and query parameters (`?...`)
- Adjust timeouts or user-agent in `sitemap/crawler.go` if needed