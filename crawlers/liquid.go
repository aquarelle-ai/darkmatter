/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package crawlers

import (
	"encoding/json"
	"strconv"

	"aquarelle.ai/darkmatter/types"
)

const (
	LIQUID_MODULE_NAME = "Liquid REST API"
	LIQUID_APIURL      = "https://api.liquid.com/products/1"
)

// The REST API client to get data from Liquid
type LiquidCrawler struct {
	DataCrawler Crawler
	Ticker      string
}

// Creates a new crawler
func NewLiquidCrawler() LiquidCrawler {
	crawler := NewCrawler(LIQUID_APIURL)
	crawler.Headers = make(map[string]string)
	crawler.Headers["X-Quoine-API-Version"] = "2"

	return LiquidCrawler{
		DataCrawler: crawler,
	}
}

// Return the name of this crawler
func (c LiquidCrawler) GetName() string {
	return c.Ticker
}

func (c LiquidCrawler) GetTicker() string {
	return c.Ticker
}

// Serializes a json to a TickerInfo24 type
func (c LiquidCrawler) ToPriceSummary(jsonData []byte) types.PriceSummary {

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

// Helper function to convert the json from Liquid´s API to a PriceSummary instance
func (c LiquidCrawler) Crawl(quotedCurrency string, done chan types.PriceSummary) {

	jsonData, err := c.DataCrawler.Get()
	if err != nil {
		return
	}

	done <- c.ToPriceSummary(jsonData)
}
