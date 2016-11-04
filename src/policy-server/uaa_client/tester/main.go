package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pivotal-cf-experimental/warrant"
)

func main() {
	clientName := os.Args[1]
	clientSecret := os.Args[2]

	w := warrant.New(warrant.Config{
		Host:          "https://uaa.bosh-lite.com",
		SkipVerifySSL: true,
	})

	clientToken, err := w.Clients.GetToken(clientName, clientSecret)
	if err != nil {
		log.Fatalf("Unable to fetch client token: %s", err)
	}

	fmt.Println(clientToken)
}
