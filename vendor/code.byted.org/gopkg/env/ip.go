package env

import (
	"os"

	"code.byted.org/gopkg/net2"
)

// HostIP .
func HostIP() string {
	if os.Getenv("IS_TCE_DOCKER_ENV") == "1" {
		return os.Getenv("HOST_IP_ADDR")
	}

	return net2.GetLocalIp()
}
