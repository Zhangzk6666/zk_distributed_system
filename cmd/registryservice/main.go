package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zk_distributed_system/registry"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	router := gin.New()
	registry.RegisterHandlers(router)
	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", registry.ServiceHost, registry.ServicePort),
		Handler:        router,
		MaxHeaderBytes: 1 << 20,
	}
	go registry.Heartbeat(5 * time.Second)
	go func() {
		log.Println(srv.ListenAndServe())
		log.Println("注册中心退出")
		cancel()
	}()
	go func() {
		log.Println("注册中心 started. Press use 'Ctrl + c' to stop.")
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
		<-c
		srv.Shutdown(ctx)
		cancel()
	}()
	<-ctx.Done()
	fmt.Println("shutdown ....")
}
