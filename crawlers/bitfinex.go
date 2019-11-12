/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package crawlers

import (
	"encoding/json"
	"reflect"

	"aquarelle.ai/darkmatter/types"
)

const (
	BITFINEX_MODULE_NAME = "Bitfinex REST API"
	BITFINEX_APIURL      = "https://api-pub.bitfinex.com/v2/ticker/tBTCUSD"
)

// The REST API client to get data from Bitfinex
type BitfinexCrawler struct {
	DataCrawler Crawler
	Ticker      string
}

// Creates a new crawler
func NewBitfinexCrawler() BitfinexCrawler {
	crawler := NewCrawler(BITFINEX_APIURL)

	return BitfinexCrawler{
		DataCrawler: crawler,
	}
}

// Return the name of this crawler
func (c BitfinexCrawler) GetName() string {
	return BITFINEX_MODULE_NAME
}

func (c BitfinexCrawler) GetTicker() string {
	return c.Ticker
}

// Serializes a json to a TickerInfo24 type
func (c BitfinexCrawler) ToPriceSummary(jsonData []byte) types.PriceSummary {

	var result types.PriceSummary

	aux := make([]interface{}, 16)
	if err := json.Unmarshal(jsonData, &aux); err != nil {
		panic(err)
	}

	result = types.PriceSummary{}
	result.Volume = getFloat(aux[7])
	result.HighPrice = getFloat(aux[8])
	// result.OpenPrice, _ = strconv.ParseFloat(aux.OpenPrice, 32)

	return result
}

func getFloat(unk interface{}) float64 {
	v := reflect.ValueOf(unk)
	floatType := reflect.TypeOf(float64(0))
	fv := v.Convert(floatType)
	return fv.Float()
}

func (c BitfinexCrawler) SetTicker(quotedCurrency string) {

	switch quotedCurrency {
	case "USD":
		c.Ticker = "BTCUSD"
	}
}

// Helper function to convert the json from Bitfinex´s API to a PriceSummary instance
func (c BitfinexCrawler) Crawl(quotedCurrency string, done chan types.PriceSummary) {

	c.SetTicker(quotedCurrency)
	jsonData, err := c.DataCrawler.Get()

	if err != nil {
		return
	}

	done <- c.ToPriceSummary(jsonData)
}
