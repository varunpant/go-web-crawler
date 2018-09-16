package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type webCrawler struct {
	root        string
	concurrency int
	delay       float64
}

func getHTTPClient() http.Client {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	return client
}

//convert relative to fully qualified
func getFullyQualifiedURI(rootURL *url.URL, relativeURI string) string {
	u := url.URL{
		Scheme: rootURL.Scheme,
		Host:   rootURL.Host,
		Path:   relativeURI,
	}
	return u.String()
}

func (w *webCrawler) start() error {

	rootURL, err := url.Parse(w.root)
	if err != nil {
		return fmt.Errorf("Bad root url[%s] passed. : %v", w.root, err)

	}
	if w.concurrency < 1 {
		return fmt.Errorf("Required min 1 worker,found %v", w.concurrency)
	}

	//working list.
	tobecrawled := make(chan []string)

	//unvisited urls
	unique := make(chan string)

	//sitemap of links.
	sitemap := make(map[string][]string)
	var mutex = &sync.Mutex{}
	t := time.Now()
	go func() { tobecrawled <- []string{w.root} }()

	for i := 0; i < w.concurrency; i++ {

		//create concurrent workers
		go func() {
			//this is a blocking call for workers till the main goroutine exists.
			for link := range unique {

				links, err := w.crawl(link, rootURL)
				if err != nil {
					continue
				}
				mutex.Lock()
				sitemap[link] = links
				t = time.Now()
				mutex.Unlock()

				if len(links) > 0 {
					go func() { tobecrawled <- links }()
				}

			}
		}()
	}
	go func(delay float64) {
		for {
			time.Sleep(time.Duration(1))
			if time.Now().Sub(t).Seconds() >= delay {
				close(tobecrawled)
				break
			}

		}

	}(w.delay)
	//Links already visited.
	crawled := make(map[string]bool)
	for list := range tobecrawled {
		for _, link := range list {
			if !crawled[link] {
				crawled[link] = true
				unique <- link
			}
		}
	}

	markup := "<html><head><link rel=\"stylesheet\" href=\"https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css\" ></head><body><table class=\"table table-hover table-condensed\">"
	for key, links := range sitemap {
		markup += "<tr><td>" + key + "</td><td><ul>"
		for _, l := range links {
			markup += "<li>" + l + "</li>"
		}
		markup += "</ul></td></tr>"
	}
	markup += "</table></body></html>"
	ioutil.WriteFile("sitemap.html", []byte(markup), 0644)

	log.Println("finito...")
	return nil
}

func (w *webCrawler) crawl(link string, rootURL *url.URL) ([]string, error) {

	body, err := w.fetch(link)
	if err != nil {
		return nil, err
	}
	links, err := extractLinks(body, rootURL)
	if err != nil {
		return nil, err
	}
	return links, nil
}

func extractLinks(data []byte, rootURL *url.URL) ([]string, error) {

	r := bytes.NewReader(data)
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing body: %v", err)
	}
	// Copied from here: https://godoc.org/golang.org/x/net/html#example-Parse
	var links []string

	var f func(*html.Node)

	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					link, err := url.Parse(a.Val)
					if err != nil {
						continue
					}
					if len(link.Host) == 0 {
						if strings.HasPrefix(a.Val, "/") {
							uri := getFullyQualifiedURI(rootURL, a.Val)
							links = append(links, uri)
						}
					} else if rootURL.Host == link.Host {
						links = append(links, link.String())
					}

				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return links, nil

}

func (w *webCrawler) fetch(url string) ([]byte, error) {
	log.Println("fetchin " + url)
	c := getHTTPClient()
	resp, err := c.Get(url)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("Failed %s: %s", url, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func main() {

	log.Println("starting...")

	base := "https://varunpant.com/"
	crawler := webCrawler{base, 10, 3}
	crawler.start()

}
