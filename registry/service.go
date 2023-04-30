package registry

import (
	"fmt"
	"sync"
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
	registration []Registration
	mutex        *sync.Mutex
}

var selfReg = registry{
	registration: make([]Registration, 0),
	mutex:        new(sync.Mutex),
}

func RegisterHandlers(router *gin.Engine) {
	zklog.Logger.Info("Request received")
	router.POST("/services", addService)
	router.DELETE("/services", removeService)
}

// / 服务注册
func addService(ctx *gin.Context) {
	var r Registration
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
	var r Registration
	ctx.ShouldBind(&r)
	err := valid.Verification.Verify(r)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	url := r.ServiceURL
	zklog.Logger.Info("Remove service at URL:", url)
	err = selfReg.remove(url)
	if err != nil {
		zklog.Logger.Error(err)
		response.ResponseMsg.FailResponse(ctx, err, nil)
		return
	}
	response.ResponseMsg.SuccessResponse(ctx, nil)
}

func (r *registry) add(reg Registration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.registration = append(r.registration, reg)
	return nil
}
func (r *registry) remove(url string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for i := range r.registration {
		if r.registration[i].ServiceURL == url {
			r.registration = append(r.registration[:i], r.registration[i+1:]...)
			return nil
		}
	}

	return response.NewErrWithMsg(response.PARAMETER_ERROR, fmt.Sprintf("Service at URL %s not found", url))
}
