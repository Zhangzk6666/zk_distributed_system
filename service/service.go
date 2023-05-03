package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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

// 获取服务
func GetService(sericeName registry.ServiceName) (string, error) {
	reqUrl := fmt.Sprintf("%s?serviceName=%s", registry.ServiceURL, url.QueryEscape(string(sericeName)))
	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, reqUrl, nil)
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", err

	}
	msg := MSG{}
	json.Unmarshal(body, &msg)
	if msg.Code == response.SUCCESS {
		return msg.Data.URL, nil
	}
	return "", response.NewErr(response.ERROR)
}

// {"code":200,"data":{"url":"http://localhost:8111"},"msg":"success"}
type MSG struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data URL    `json:"data"`
}
type URL struct {
	URL string `json:"url"`
}
