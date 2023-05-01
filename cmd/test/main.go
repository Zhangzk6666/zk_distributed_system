package main

import (
	"context"
	"zk_distributed_system/registry"
	"zk_distributed_system/service"
	"zk_distributed_system/zklog"

	"fmt"

	"github.com/gin-gonic/gin"
)

// "test服务"
func main() {
	host, port := "localhost", "8111"
	r := registry.RegistrationVO{
		ServiceName: "Test Service",
		ServiceURL:  fmt.Sprintf("http://%s:%s", host, port),
	}
	ctx, err := service.Start(
		context.Background(),
		host,
		port,
		r,
		func(router *gin.Engine) {

		})

	if err != nil {
		zklog.Logger.Error(err)
		panic(err)
	}
	<-ctx.Done()
	fmt.Println("shutdown ....")
}
