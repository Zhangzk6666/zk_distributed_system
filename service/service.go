package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"zk_distributed_system/pkg/response"
	"zk_distributed_system/registry"

	"github.com/gin-gonic/gin"
)

// 启动服务并注册
func Start(ctx context.Context, host, port string,
	reg registry.RegistrationVO, routerFunc func(router *gin.Engine)) (context.Context, error) {
	ctx = startService(ctx, reg.ServiceName, host, port, routerFunc)
	err := registry.RegisterService(reg)
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

func startService(ctx context.Context, serviceName registry.ServiceName, host, port string, routerFunc func(router *gin.Engine)) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	router := gin.New()
	healthyService(router)
	routerFunc(router)
	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", host, port),
		Handler:        router,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		log.Println(srv.ListenAndServe())
		err := registry.ShutdownService(serviceName, fmt.Sprintf("http://%s:%s", host, port))
		if err != nil {
			log.Println(err)
		}
		cancel()
	}()
	go func() {
		log.Printf("%v started. Press use 'Ctrl + c' to stop. \n", serviceName)
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
		<-c
		srv.Shutdown(ctx)
	}()
	return ctx
}

func healthyService(router *gin.Engine) {
	router.GET("/healthy", func(ctx *gin.Context) {
		response.ResponseMsg.SuccessResponse(ctx, nil)
	})
}
