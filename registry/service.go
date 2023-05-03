package registry

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"zk_distributed_system/pkg/response"
	"zk_distributed_system/pkg/valid"
	"zk_distributed_system/zklog"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	ServiceHost = "localhost"
	ServicePort = "8222"
	ServiceURL  = "http://" + ServiceHost + ":" + ServicePort + "/services"
)

type registry struct {
	registration map[ServiceName][]string // sericeName:[]string || 服务名:URLS
	mutex        *sync.RWMutex

	// 负载均衡
	// 心跳检测
	// Notify 当注册中心易主后通知所有服务
}

var selfReg = registry{
	registration: make(map[ServiceName][]string, 0),
	mutex:        new(sync.RWMutex),
}

func RegisterHandlers(router *gin.Engine) {
	zklog.Logger.Info("Request received")
	// 获取服务
	router.GET("/services", getService)
	router.POST("/services/get", getService)
	// 注册服务
	router.POST("/services", addService)
	// 注销服务
	router.DELETE("/services", removeService)
}

// / 服务注册
func addService(ctx *gin.Context) {
	var r RegistrationVO
	ctx.ShouldBind(&r)
	err := valid.Verification.Verify(r)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	zklog.Logger.WithFields(logrus.Fields{
		"ServiceName": r.ServiceName,
		"ServiceURL":  r.ServiceURL,
	}).Info("Adding service:")

	err = selfReg.add(r)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	response.ResponseMsg.SuccessResponse(ctx, nil)
}

// /服务注销
func removeService(ctx *gin.Context) {
	var r RegistrationVO
	ctx.ShouldBind(&r)
	err := valid.Verification.Verify(r)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	url := r.ServiceURL
	zklog.Logger.Info("Remove service at URL:", url)
	err = selfReg.remove(r)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	response.ResponseMsg.SuccessResponse(ctx, nil)
}

func urlsExistUrl(urls []string, serviceUrl string) bool {
	urlMap := make(map[string]struct{}, 0)
	for i := 0; i < len(urls); i++ {
		urlMap[urls[i]] = struct{}{}
	}
	_, exist := urlMap[serviceUrl]
	return exist
}
func (r *registry) add(reg RegistrationVO) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	serviceName := reg.ServiceName
	serviceUrl := reg.ServiceURL
	if _, ok := r.registration[serviceName]; !ok {
		r.registration[serviceName] = make([]string, 0)
	}

	if exist := urlsExistUrl(r.registration[serviceName], serviceUrl); !exist {
		fmt.Println(exist, "======= ======", r.registration)
		r.registration[serviceName] = append(r.registration[serviceName], serviceUrl)
	}
	return nil
}
func (r *registry) remove(reg RegistrationVO) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	serviceName := reg.ServiceName
	serviceUrl := reg.ServiceURL
	if _, exist := r.registration[serviceName]; exist {
		for i := range r.registration[serviceName] {
			if r.registration[serviceName][i] == serviceUrl {
				r.registration[serviceName] = append(r.registration[serviceName][:i], r.registration[serviceName][i+1:]...)
				return nil
			}
		}
		return response.NewErrWithMsg(response.PARAMETER_ERROR,
			fmt.Sprintf("Found serviceName: %s ,not found URL: %s", serviceName, serviceUrl))
	}
	return response.NewErrWithMsg(response.PARAMETER_ERROR,
		fmt.Sprintf("Not found serviceName: %s ,not found URL: %s", serviceName, serviceUrl))
}

// 心跳检测
func Heartbeat(interval time.Duration) {
	for {
		checkReg := selfReg
		tempUrlsMap := make(map[ServiceName]map[string]int)
		for i := 0; i < 3; i++ {
			for serviceName, serviceURLs := range checkReg.registration {
				for _, url := range serviceURLs {
					resp, err := http.Get(url + "/healthy")
					if err != nil || resp.StatusCode != http.StatusOK {
						zklog.Logger.WithFields(logrus.Fields{
							"sericeName": serviceName,
							"serviceURL": url,
						}).Error("[心跳检测] 检测错误...")
						urlsMap, ok := tempUrlsMap[serviceName]
						if !ok {
							tempUrlsMap[serviceName] = make(map[string]int)
							urlsMap = tempUrlsMap[serviceName]
						}
						counts := urlsMap[url]
						urlsMap[url] = counts + 1
					}
					// else {
					// zklog.Logger.WithFields(logrus.Fields{
					// 	"sericeName": serviceName,
					// 	"serviceURL": url,
					// }).Info("[心跳检测] 检测通过...")
					// }
				}
			}
		}
		removeUrlsMap := make(map[ServiceName][]string)
		for serviceName, urlsMap := range tempUrlsMap {
			for url, counts := range urlsMap {
				if counts == 3 {
					_, exist := removeUrlsMap[serviceName]
					if !exist {
						removeUrlsMap[serviceName] = make([]string, 0)
					}
					removeUrlsMap[serviceName] = append(removeUrlsMap[serviceName], url)
				}
			}
		}
		//删除 心跳检测失败的
		go removeUrls(removeUrlsMap)
		time.Sleep(interval)
	}
}
func removeUrls(removeUrlsMap map[ServiceName][]string) {
	for serviceName, serviceUrls := range removeUrlsMap {
		for _, url := range serviceUrls {
			selfReg.remove(RegistrationVO{
				ServiceName: serviceName,
				ServiceURL:  url,
			})
		}
	}

}

// 注册中心拉取服务 | 随机负载均衡
func getService(ctx *gin.Context) {
	var r GetServiceVO
	ctx.ShouldBind(&r)
	err := valid.Verification.Verify(r)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	selfReg.mutex.RLock()
	defer selfReg.mutex.RUnlock()
	if len(selfReg.registration[r.ServiceName]) == 0 {
		response.ResponseMsg.FailResponse(ctx, response.NewErr(response.ERROR), nil)
		return
	}
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(selfReg.registration[r.ServiceName]))
	url := selfReg.registration[r.ServiceName][index]
	zklog.Logger.WithFields(logrus.Fields{
		"Selected Instance:": url,
		"index":              index,
		"counts":             len(selfReg.registration[r.ServiceName]),
	}).Info("Selected Instance:", url)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	response.ResponseMsg.SuccessResponse(ctx, GetServiceDTO{
		Url: url,
	})
}
