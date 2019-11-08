package types

// Ticker info 24
type TickerInfo24 struct {
	Symbol             string `json:"symbol"`
	PriceChange        uint32 `json:"priceChange"`
	PriceChangePercent uint32 `json:"priceChangePercent"`
	LastQty            uint32 `json:"LastQty"`
	LastPrice          uint32 `json:"lastPrice"`
	BidPrice           uint32 `json:"bidPrice"`
	AskPrice           uint32 `json:"askPrice"`
	BidQty             uint32 `json:"bidQty"`
	AskQty             uint32 `json:"askQty"`
	QuoteVolumen       uint32 `json:"quoteVolumen"`
	Volume             uint32 `json:"volume"`
	HighPrice          uint32 `json:"highPrice"`
	LowPrice           uint32 `json:"lowPrice"`
	OpenPrice          uint32 `json:"openPrice"`
	OpenTime           uint32 `json:"openTime"`
	CloseTime          uint32 `json:"closeTime"`
}
