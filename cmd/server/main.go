package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/liigoo/pi-fm-rds-go/internal/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Configuration loaded successfully:\n")
	fmt.Printf("  Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  FM Frequency: %.1f MHz\n", cfg.PiFmRds.DefaultFrequency)
	fmt.Printf("  Upload Dir: %s\n", cfg.Storage.UploadDir)
	fmt.Printf("  Sample Rate: %d Hz\n", cfg.Audio.SampleRate)
	fmt.Printf("  Max WebSocket Clients: %d\n", cfg.WebSocket.MaxClients)

	os.Exit(0)
}
