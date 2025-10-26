package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"veriChat/go/internal/api"
	"veriChat/go/internal/db"
	"veriChat/go/internal/service"
)

func main() {
	if err := db.Init("user:pass@tcp(localhost:3306)/verichat?parseTime=true"); err != nil {
		log.Fatal(err)
	}

	db.InitRedis("localhost:6379", "", 0)

	svc := service.NewMessageService(service.Config{
		BatchSize:    64,
		BatchTimeout: 300 * time.Millisecond,
		LockTTL:      5 * time.Second,
		RedisClient:  db.RedisClient,
	})

	server := api.NewServer(":8080", svc)

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	svc.Shutdown(ctx)
	log.Println("Server exiting")
}
