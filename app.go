/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package main

import (
	// "encoding/binary"
	// "fmt"

	"log"
	"net/http"

	"github.com/aquarelle-tech/darkmatter/mapreduce"
	"github.com/aquarelle-tech/darkmatter/service"

	"github.com/aquarelle-tech/darkmatter/x/cryptocerts/crawlers"
	cryptocerts "github.com/aquarelle-tech/darkmatter/x/cryptocerts/types"

	// "github.com/aquarelle-tech/darkmatter/shamir"
	"github.com/aquarelle-tech/darkmatter/types"
)

// List of available crawlers
var directory = []cryptocerts.PriceEvidenceCrawler{
	crawlers.NewBinanceCrawler(),
	crawlers.NewLiquidCrawler(),
	crawlers.NewBitfinexCrawler(),
}

var publishedPrices = make(chan types.FullSignedBlock)

func main() {

	// s := uint64(123456)

	// buf := make([]byte, 8)
	// binary.LittleEndian.PutUint64(buf, s)
	// fmt.Printf("buf=%d\n", buf)

	// result, _ := shamir.Split(buf, 5, 3)
	// fmt.Printf("result=%d\n", result)

	// bytes, _ := shamir.Combine(result)
	// fmt.Printf("bytes=%d\n", bytes)

	// fmt.Println(binary.LittleEndian.Uint64(bytes))

	quotedCurrency := "USD"

	// Prepare and run the subroutines for the oracle service
	server := service.NewOracleServer(publishedPrices)
	server.Initialize()

	// Prepare and start the subroutines to manage the request of sources
	processor := mapreduce.NewMapReduceProcessor(directory, quotedCurrency, publishedPrices)
	processor.Initialize()

	// handler := cors.Default().Handler(mux)
	err := http.ListenAndServe(":6877", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
