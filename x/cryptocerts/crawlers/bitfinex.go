/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package crawlers

import (
	"encoding/json"
	"reflect"
	"time"

	"../types"
)

const (
	BITFINEX_MODULE_NAME = "Bitfinex REST API"
	BITFINEX_APIURL      = "https://api-pub.bitfinex.com/v2/ticker/tBTCUSD"
)

// The REST API client to get data from Bitfinex
type BitfinexCrawler struct {
	DataCrawler types.Crawler
	Ticker      string
}

// Creates a new crawler
func NewBitfinexCrawler() BitfinexCrawler {
	crawler := types.NewCrawler(BITFINEX_APIURL)

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
func (c BitfinexCrawler) ToQuotePriceInfo(jsonData []byte) types.QuotePriceInfo {

	var result types.QuotePriceInfo

	aux := make([]interface{}, 16)
	if err := json.Unmarshal(jsonData, &aux); err != nil {
		panic(err)
	}

	result = types.QuotePriceInfo{}
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

// Helper function to convert the json from Bitfinex´s API to a QuotePriceInfo instance
func (c BitfinexCrawler) Crawl(quotedCurrency string, done chan types.QuotePriceInfo) {

	c.SetTicker(quotedCurrency)
	jsonData, err := c.DataCrawler.Get()

	if err != nil {
		return
	}

	priceInfo := c.ToQuotePriceInfo(jsonData)
	priceInfo.Timestamp = time.Now().Unix()
	priceInfo.DataURL = BITFINEX_APIURL
	priceInfo.ExchangeUID = "bitfinex"
	done <- priceInfo
}
