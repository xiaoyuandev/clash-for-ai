package main

import (
	"log"
	"os"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/app"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewayruntime"
)

func main() {
	if os.Getenv("CLASH_FOR_AI_MODE") == "local-gateway-runtime" {
		if err := localgatewayruntime.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
