package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Zoomish/go-ggsel-api"
)

func main() {
	key := os.Getenv("GGSEL_API_KEY")
	if key == "" {
		fmt.Println("GGSEL_API_KEY не задан")
		return
	}
	sellerID, err := strconv.ParseInt(os.Getenv("GGSEL_SELLER_ID"), 10, 64)
	if err != nil || sellerID <= 0 {
		fmt.Println("GGSEL_SELLER_ID не задан")
		return
	}
	invoiceID, err := strconv.ParseInt(os.Getenv("GGSEL_INVOICE_ID"), 10, 64)
	if err != nil || invoiceID <= 0 {
		fmt.Println("GGSEL_INVOICE_ID не задан")
		return
	}
	client, err := ggsel.New(ggsel.Options{APIKey: key, SellerID: sellerID})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	order, err := client.GetOrder(context.Background(), invoiceID)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(order.InvoiceState)
}
