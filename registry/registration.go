package registry

type RegistrationVO struct {
	ServiceName ServiceName `form:"serviceName" json:"serviceName" validate:"required"`
	ServiceURL  string      `form:"serviceURL" json:"serviceURL" validate:"required"`
}

// type Registration struct {
// 	ServiceName ServiceName
// 	ServiceURL  string
// }

type ServiceName string

const (
	LogService = ServiceName("LogService")
)
