package crawlers

import (
	"encoding/json"
	"strconv"

	"cratos.network/darkmatter/types"
)

const (
	UPBIT_MODULE_NAME = "UpBit REST API"
	UPBIT_APIURL      = "https://api.UpBit.com/products/1"
)

// The REST API client to get data from UpBit
type UpBitCrawler struct {
	DataCrawler Crawler
}

// Creates a new crawler
func NewUpBitCrawler() UpBitCrawler {
	crawler := NewCrawler(UPBIT_APIURL)
	crawler.Headers = make(map[string]string)
	crawler.Headers["X-Quoine-API-Version"] = "2"

	return UpBitCrawler{
		DataCrawler: crawler,
	}
}

// Return the name of this crawler
func (client UpBitCrawler) GetName() string {
	return UPBIT_MODULE_NAME
}

// Serializes a json to a TickerInfo24 type
func (c UpBitCrawler) ToPriceSummary(jsonData []byte) types.PriceSummary {

	var result types.PriceSummary
	aux := struct {
		Volume    string `json:"volume_24h"`
		HighPrice string `json:"high_market_ask"`
	}{}

	if err := json.Unmarshal(jsonData, &aux); err != nil {
		panic(err)
	}

	result = types.PriceSummary{}
	result.Volume, _ = strconv.ParseFloat(aux.Volume, 32)
	// result.QuoteVolume, _ = strconv.ParseFloat(aux.QuoteVolume, 32)
	result.HighPrice, _ = strconv.ParseFloat(aux.HighPrice, 32)
	// result.OpenPrice, _ = strconv.ParseFloat(aux.OpenPrice, 32)

	return result
}

// Helper function to convert the json from UpBit´s API to a PriceSummary instance
func (c UpBitCrawler) Crawl() types.PriceSummary {

	jsonData, err := c.DataCrawler.Get()
	if err != nil {
		panic(err)
	}
	info := c.ToPriceSummary(jsonData)

	return info
}
