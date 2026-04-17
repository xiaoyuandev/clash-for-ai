package main

import (
	"log"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
