package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Zoomish/go-ggsel-api"
)

func main() {
	key := os.Getenv("GGSEL_API_KEY")
	if key == "" {
		fmt.Println("GGSEL_API_KEY не задан")
		return
	}
	client, err := ggsel.New(ggsel.Options{APIKey: key})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	page, err := client.ListOffers(context.Background(), ggsel.ListOffersParams{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(len(page.Offers))
}
