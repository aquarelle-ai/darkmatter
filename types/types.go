/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package types

import (
	"encoding/json"
	"time"
)

// The model used to get the data
type PriceSummary struct {
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
	// LowPrice           float64 `json:"lowPrice"`
	// OpenTime           int64  `json:"openTime"`
	// CloseTime          int64  `json:"closeTime"`
}

func (info PriceSummary) String() string {
	result, err := json.Marshal(&info)

	if err != nil {
		panic(err)
	}

	return string(result)
}

/**************   Service´s Public Messages ******************/
// Message to send to the connected clients through websocket
type PriceMessage struct {
	AveragePrice  float64 `json:"avgPrice"`
	AverageVolume float64 `json:"avgVolumen"`
	UID           string  `json:"uid"`
	Ticker        string  `json:"ticker"`
	Timestamp     int64   `json:"timestamp"`
}

/**************** Map & Reduce types ***************************/
// The job message to insert in a queue to be processed as part of the the Mapping Stage
type GetDataJob struct {
	Quote       string
	UID         string
	DataCrawler PriceSourceCrawler
}

// The result message that will receive the results from the mapped nodes in the Reduce Stage
type Result struct {
	CrawlerName string
	Data        PriceSummary
	UID         string
	HasError    bool
	Timestamp   time.Time
	Ticker      string
}

/*****  Crawkers types **********************/
// Interface for clients
type PriceSourceCrawler interface {
	Crawl(quotedCurrency string, done chan PriceSummary)
	GetName() string
	GetTicker() string
}
