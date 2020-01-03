/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package types

import (
	"encoding/json"
	"fmt"
)

// QuotePriceInfo is the model used to get the data
type QuotePriceInfo struct {
	// Symbol string `json:"symbol"`
	// PriceChange        float32 `json:"priceChange"`
	// PriceChangePercent float32 `json:"priceChangePercent"`
	// LastQty            float32 `json:"LastQty"`
	// // LastPrice          float32 `json:"lastPrice"`
	// BidPrice           float32 `json:"bidPrice"`
	// AskPrice           float32 `json:"askPrice"`
	// BidQty             float32 `json:"bidQty"`
	// AskQty             float32 `json:"askQty"`
	QuoteVolume float64 `json:"quoteVolumen"`
	Volume      float64 `json:"volume"`
	HighPrice   float64 `json:"highPrice"`
	OpenPrice   float64 `json:"openPrice"`
	Timestamp   int64   `json:"timestamp"`
	DataURL     string  `json:"dataUrl"`
	ExchangeUID string  `json:"provider"`
	// LowPrice           float64 `json:"lowPrice"`
	// OpenTime           int64  `json:"openTime"`
	// CloseTime          int64  `json:"closeTime"`
}

// String will print the JSON output
func (info QuotePriceInfo) String() string {
	result, err := json.Marshal(&info)

	if err != nil {
		panic(err)
	}

	return string(result)
}

// PriceEvidenceCrawler is the interface for clients
type PriceEvidenceCrawler interface {
	Crawl(quotedCurrency string, done chan QuotePriceInfo)
	GetName() string
	GetTicker() string
}

// ExchangeMarketEvent holds the order book price and quantity depth updates for any exchange. Used to unify
// the events in different exchanges
type ExchangeMarketEvent struct {
	ExchangeName  string     `json:"exchangeName"`
	Symbol        string     `json:"symbol"`
	FirstUpdateID int64      `json:"firstUpdateId"` // First update ID in event
	LastUpdateID  int64      `json:"lastUpdateId"`  // Final update ID in event
	Bids          [][]string `json:"bids"`          // Bids to be updated
	Asks          [][]string `json:"asks"`          // Asks to be updated
}

// String implements the Stringer interface
func (e ExchangeMarketEvent) String() string {
	return fmt.Sprintf("%s (%s): firstUpdateID:%d, lastUpdateID:%d ", e.ExchangeName, e.Symbol, e.FirstUpdateID, e.LastUpdateID)
}
