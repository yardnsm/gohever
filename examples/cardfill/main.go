package main

import (
	"fmt"
	"log"
	"math"

	"github.com/yardnsm/gohever"
)

func main() {
	config := gohever.Config{
		Credentials: gohever.BasicCredentials("myusername", "mypassword"),
		CreditCard:  gohever.BasicCreditCard("123456789", "04", "27"),
	}

	hvr := gohever.NewClient(config)
	keva := hvr.Cards.Keva

	status, err := keva.GetStatus()
	if err != nil {
		log.Fatalf("unable to fetch the card status: %v\n", err)
	}

	// Check how much is left to fill up the card
	amountToFill := math.Min(status.RemainingOnCardAmount, status.RemainingMonthlyAmount)
	if amountToFill == 0 {
		log.Fatalf("unable to fill the card: card is full / you've reached your monthly limit\n")
	}


	estimate, err := status.Estimate(status.RemainingOnCardAmount)
	if err != nil {
		log.Fatalf("unable to make estimations: %v\n", err)
	}

	fmt.Printf(
		"filling up the card up to %dILS by loading it with %.2fILS (%.2fILS after discount)\n",
		status.MaxOnCardAmount,
		estimate.Total,
		estimate.TotalFactored,
	)

	result, err := keva.Load(*status, int32(status.RemainingOnCardAmount))
	if err != nil {
		log.Fatalf("unable to perform load request: %v\n", err)
	}

	switch result.Status {
	case gohever.StatusSuccess:
		fmt.Printf("card was loaded successfuly! load number: %s\n", result.LoadNumber)
		fmt.Printf("raw message from Hever: %s", result.RawMessage)
	case gohever.StatusNone:
		fallthrough
	case gohever.StatusError:
		fmt.Printf("card load failed\n")
		fmt.Printf("raw message from Hever: %s\n", result.RawMessage)
	}
}
