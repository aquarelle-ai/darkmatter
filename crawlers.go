package crawlers

import (
	"io/ioutil"
	"net/http"
)

type Crawler struct {
	Url string
}

func NewCrawler(url string) Crawler {
	return Crawler{
		Url: url,
	}
}

// Return the data
func (crawler Crawler) Get() (string, error) {

	response, err := http.Get(crawler.Url)

	if err != nil {
		return string([]byte(nil)), err
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		return string(data), nil
	}
}
