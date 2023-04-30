package registry

type Registration struct {
	ServiceName ServiceName `form:"serviceName" json:"serviceName" validate:"required"`
	ServiceURL  string      `form:"serviceURL" json:"serviceURL" validate:"required"`
}

type ServiceName string

const (
	LogService = ServiceName("LogService")
)
