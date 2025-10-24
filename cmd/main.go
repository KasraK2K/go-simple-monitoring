package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {

	RegisterRouter()

	addr := fmt.Sprintf(":%s", "3500")
	log.Println("Server running on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
