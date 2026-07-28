package main

import (
	"fmt"
	"log"

	"github.com/kozwoj/gobbler-agent/server"
)

func main() {
	const port = 9000
	fmt.Printf("Gobbler Agent starting on port %d\n", port)
	srv, err := server.New(port)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(srv.ListenAndServe())
}
