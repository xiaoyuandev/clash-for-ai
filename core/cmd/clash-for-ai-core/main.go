package main

import (
	"log"
	"os"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/app"
)

func main() {
	if os.Getenv("CLASH_FOR_AI_MODE") == "local-gateway-runtime" {
		if err := app.RunLocalGatewayRuntime(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
