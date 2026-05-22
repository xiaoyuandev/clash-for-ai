package main

import (
	"log"

	"github.com/xiaoyuandev/relay-switch/core/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
