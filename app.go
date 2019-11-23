/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package main

import (
	"log"
	"net/http"

	"github.com/aquarelle-tech/darkmatter/crawlers"
	"github.com/aquarelle-tech/darkmatter/mapreduce"
	"github.com/aquarelle-tech/darkmatter/service"
	"github.com/aquarelle-tech/darkmatter/types"
)

// List of available crawlers
var directory = []types.PriceEvidenceCrawler{
	crawlers.NewBinanceCrawler(),
	crawlers.NewLiquidCrawler(),
	crawlers.NewBitfinexCrawler(),
}

var publishedPrices = make(chan types.FullSignedBlock)

func main() {

	quotedCurrency := "USD"

	// Prepare and run the subroutines for the oracle service
	server := service.NewOracleServer(publishedPrices)
	server.Initialize()

	// Prepare and start the subroutines to manage the request of sources
	processor := mapreduce.NewMapReduceProcessor(directory, quotedCurrency, publishedPrices)
	processor.Initialize()

	// handler := cors.Default().Handler(mux)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
