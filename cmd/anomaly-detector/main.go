package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aiops-platform/internal/config"
	"aiops-platform/internal/detector"
)

func main() {
	var (
		configPath = flag.String("config", "configs/detector.yaml", "Путь к файлу конфигурации")
		version    = flag.Bool("version", false, "Показать версию")
	)
	flag.Parse()

	if *version {
		fmt.Printf("AIOps Anomaly Detector v1.0.0\n")
		os.Exit(0)
	}

	// Загружаем конфигурацию
	var cfg *config.Config
	var err error

	if _, err := os.Stat(*configPath); err == nil {
		cfg, err = config.LoadConfig(*configPath)
		if err != nil {
			log.Printf("Failed to load config from file, using environment: %v", err)
			cfg = config.LoadConfigFromEnv()
		}
	} else {
		log.Println("Config file not found, using environment variables")
		cfg = config.LoadConfigFromEnv()
	}

	log.Printf("Starting AIOps Anomaly Detector...")
	log.Printf("Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Prometheus: %s", cfg.Monitoring.Prometheus.URL)
	log.Printf("ML Training: %v", cfg.ML.TrainingEnabled)

	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем детектор
	detector, err := detector.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create detector: %v", err)
	}

	// Запускаем детектор
	go func() {
		if err := detector.Start(ctx); err != nil {
			log.Printf("Detector error: %v", err)
			cancel()
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Received shutdown signal")
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// Graceful shutdown
	log.Println("Shutting down detector...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := detector.Stop(shutdownCtx); err != nil {
		log.Printf("Detector shutdown error: %v", err)
	}

	log.Println("Detector stopped")
}
