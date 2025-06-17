package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"web-crawler/sitemap"
)

const xmlns = "http://sitemaps.org/schemas/sitemap/0.9"

type loc struct {
	Value string `xml:"loc"`
}

type urlset struct {
	Urls  []loc  `xml:"url"`
	Xmlns string `xml:"xmlns,attr"`
}

func main() {
	urlFlag := flag.String("url", "https://gophercises.com", "the url that you want to crawl on")
	maxDepth := flag.Int("depth", 3, "the maximum number of links deep to traverse")
	flag.Parse()

	parsedUrl, err := url.Parse(*urlFlag)
	if err != nil {
		log.Fatalf("Invalid URL provided: %v", err)
	}

	outputDir := "output"
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.Mkdir(outputDir, 0755)
		if err != nil {
			log.Fatalf("error creating output directory: %v", err)
		}
	}

	domain := strings.ReplaceAll(parsedUrl.Hostname(), ".", "-")
	filename := fmt.Sprintf("output/url-%s.xml", domain)

	links := sitemap.GetLinks(*urlFlag, *maxDepth)

	toXml := urlset{
		Xmlns: xmlns,
	}
	for _, link := range links {
		toXml.Urls = append(toXml.Urls, loc{link})
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("error creating file %s: %v", filename, err)
	}
	defer file.Close()

	fmt.Fprint(file, xml.Header)
	enc := xml.NewEncoder(file)
	enc.Indent("", "  ")
	if err := enc.Encode(toXml); err != nil {
		log.Fatalf("error encoding xml: %v", err)
	}

	fmt.Printf("Succesfully written to %s\n", filename)
}
