/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package crawlers

import (
	"encoding/json"
	"strconv"

	"cratos.network/darkmatter/types"
)

const (
	BINANCE_MODULE_NAME = "Binance REST API"
	BINANCE_APIURL      = "https://api.binance.com/api/v3/ticker/24hr?symbol=BTCUSDT"
)

// The REST API client to get data from Binance
type BinanceCrawler struct {
	DataCrawler Crawler
	Ticker      string
}

// Creates a new crawler
func NewBinanceCrawler() BinanceCrawler {
	crawler := NewCrawler(BINANCE_APIURL)

	return BinanceCrawler{
		DataCrawler: crawler,
	}
}

// Return the name of this crawler
func (c BinanceCrawler) GetName() string {
	return BINANCE_MODULE_NAME
}

func (c BinanceCrawler) GetTicker() string {
	return c.Ticker
}

// Serializes a json to a TickerInfo24 type
func (c BinanceCrawler) ToPriceSummary(jsonData []byte) types.PriceSummary {

	var result types.PriceSummary
	aux := struct {
		Volume      string `json:"volume"`
		QuoteVolume string `json:"quoteVolume"`
		HighPrice   string `json:"highPrice"`
		OpenPrice   string `json:"openPrice"`
	}{}

	if err := json.Unmarshal(jsonData, &aux); err != nil {
		panic(err)
	}

	result = types.PriceSummary{}
	result.Volume, _ = strconv.ParseFloat(aux.Volume, 32)
	result.QuoteVolume, _ = strconv.ParseFloat(aux.QuoteVolume, 32)
	result.HighPrice, _ = strconv.ParseFloat(aux.HighPrice, 32)
	result.OpenPrice, _ = strconv.ParseFloat(aux.OpenPrice, 32)

	return result
}

// Set the ticker name according the quoted currency requested
func (c BinanceCrawler) SetTicker(quotedCurrency string) {

	switch quotedCurrency {
	case "USD":
		c.Ticker = "BTCUSDT"
	}
}

// Helper function to convert the json from Binance´s API to a PriceSummary instance
func (c BinanceCrawler) Crawl(quotedCurrency string, done chan types.PriceSummary) {

	c.SetTicker(quotedCurrency)
	jsonData, err := c.DataCrawler.Get()
	if err != nil {
		return
	}

	done <- c.ToPriceSummary(jsonData)
}
