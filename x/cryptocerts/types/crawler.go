/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package types

import (
	"io/ioutil"
	"net/http"
)

// Crawler is a general interface for all the crawlers
type Crawler struct {
	Url     string
	Headers map[string]string
}

// Create a new Crawler
func NewCrawler(url string) Crawler {
	return Crawler{
		Url: url,
	}
}

// Return the data. For now, it is just a GET
func (crawler Crawler) Get() ([]byte, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", crawler.Url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers, if any
	if crawler.Headers != nil && len(crawler.Headers) > 0 {
		for key, value := range crawler.Headers {
			_, exists := req.Header[key]
			if !exists {
				req.Header.Add(key, value)
			} else {
				req.Header.Set(key, value)
			}
		}
	}
	// read the data
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		return data, nil
	}
}
