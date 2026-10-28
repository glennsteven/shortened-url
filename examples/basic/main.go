package main

import (
	"context"
	"fmt"
	"log"
	"time"

	shorturl "github.com/glennsteven/shortened-url"
)

func main() {
	// MemoryStore is for demonstration: its data disappears when the process exits.
	service, err := shorturl.New(&shorturl.MemoryStore{}, shorturl.Config{})
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	link, err := service.Shorten(ctx, "https://example.com/docs?a=1&b=2#start", time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	destination, err := service.Resolve(ctx, link.Code)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Code:", link.Code)
	fmt.Println("Destination:", destination)
	if err := service.Revoke(ctx, link.Code); err != nil {
		log.Fatal(err)
	}
}
