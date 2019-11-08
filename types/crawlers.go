package types

// Interface for clients
type DataWebSiteClient interface {
	Crawl24() TickerInfo24
}
