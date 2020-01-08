package clients

import (
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

// WebCrawler is a an utility struct to get the info from the web
type WebCrawler struct {
	URL     string
	Headers map[string]string
}

// NewWebCrawler is the  constructor for the type Crawler. Create a new Crawler
func NewWebCrawler(url string) *WebCrawler {
	glog.Infof("Creating new web crawler for %s", url)

	return &WebCrawler{
		URL: url,
	}
}

// Get return the data. For now, it is just a GET
func (crawler WebCrawler) Get() ([]byte, error) {

	glog.Infof("Reading data from %s", crawler.URL)
	client := &http.Client{}
	req, err := http.NewRequest("GET", crawler.URL, nil)
	if err != nil {
		glog.Errorf("Error preparing request for %s. %v", crawler.URL, err)
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
		glog.Errorf("Error requesting from %s. %v", crawler.URL, err)
		return nil, err
	} else {
		data, _ := ioutil.ReadAll(response.Body)

		glog.Infof("Read %d bytes successfully (%s)", len(data), crawler.URL)
		return data, nil
	}
}
