package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrawlerStart(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		res.Write([]byte("no links"))
	}))

	defer func() { testServer.Close() }()

	crawler := webCrawler{testServer.URL, 2, 0.5}
	err := crawler.start()
	assert.NoError(t, err)
}

func TestFetch(t *testing.T) {
	BODY :="<a href=\"https://foobar.com\" />"
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		res.Write([]byte(BODY))
	}))

	defer func() { testServer.Close() }()

	crawler := webCrawler{testServer.URL, 2, 0.5}
	bodyBytes, err := crawler.fetch(testServer.URL)
	assert.NoError(t, err)
	assert.Equal(t, BODY, string(bodyBytes[:]))

}

func TestExtractLinksReturnsSameHost(t *testing.T) {
	rootURL, _ := url.Parse("https://foobar.com/")
	{

		body := []byte("<a href=\"https://foobar.com/ \" />")
		links, err := extractLinks(body, rootURL)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(links))
	}
}
func TestExtractLinksIgrnoresWrongHost(t *testing.T) {
	rootURL, _ := url.Parse("https://foobar.com/")
	{
		body := []byte("<a href=\"https://web.foobar.com/ \" />")
		links, err := extractLinks(body, rootURL)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(links))
	}
}
func TestExtractLinksHandlesRelativeLinks(t *testing.T) {
	rootURL, _ := url.Parse("https://foobar.com/")
	{
		body := []byte("<a href=\"/foo\" />")
		links, err := extractLinks(body, rootURL)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(links))
		assert.Equal(t, "https://foobar.com/foo", links[0])
	}
}
