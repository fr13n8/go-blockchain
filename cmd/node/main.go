package main

import (
	"flag"
	"fmt"
)

func main() {
	port := flag.Uint("port", 5050, "port to listen on")
	flag.Parse()
	cfg := ServerConfig{
		port:       uint16(*port),
		serverName: "Node",
		host:       "0.0.0.0",
	}
	s := NewServer(&cfg)
	quit := s.Run()
	<-quit
	fmt.Println("Shutting down client server...")
	s.ShutdownGracefully()
}
