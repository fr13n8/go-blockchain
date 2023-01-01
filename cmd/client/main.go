package main

import (
	"flag"
	"github.com/fr13n8/go-blockchain/wallet"
	"log"
)

func main() {
	// get gateway from flag
	gateway := flag.String("gateway", ":5050", "gateway address")
	flag.Parse()

	cfg := wallet.Config{
		Port:       8080,
		ServerName: "Wallet",
		Host:       "0.0.0.0",
		Gateway:    "0.0.0.0" + *gateway,
	}
	s := wallet.NewServer(&cfg)
	log.Printf("[WALLET] Start wallet listen on port %d\n", 8080)
	quit := s.Run()
	<-quit
	log.Printf("[WALLET] Stop wallet listen on port %d\n", 8080)
	s.ShutdownGracefully()
}
