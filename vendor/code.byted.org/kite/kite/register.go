package kite

import (
	"fmt"
	"os"
	"strconv"

	"code.byted.org/golf/consul"
	"code.byted.org/kite/kitenv"
)

const (
	isRegister = "IS_LOAD_REGISTERED"
	isProduct  = "IS_PROD_RUNTIME"
)

var (
	register *consul.RegisterContext
)

// Register write its name into consul for other services lookup
func Register() error {
	if !kitenv.Product() && os.Getenv(isProduct) == "" {
		// Only register in prod or IS_PROD_RUNTIME is setted
		return nil
	}
	if os.Getenv(isRegister) == "1" {
		// load script has registed
		return nil
	}
	var err error
	register, err = consul.InitRegister()
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(ServicePort)
	if err != nil {
		return fmt.Errorf("parse service port %s", err)
	}
	register.DefineService(ServiceName, port, map[string]string{
		"transport": "thrift.TBufferedTransport",
		"protocol":  "thrift.TBinaryProtocol",
		"version":   ServiceVersion,
	}, -1)
	return register.StartRegister()
}

// StopRegister stops register loop and deregisters service
func StopRegister() error {
	if register == nil {
		return nil
	}
	if err := register.StopRegister(); err != nil {
		return fmt.Errorf("Failed to stop register: %s", err.Error())
	}
	serviceId := fmt.Sprintf("%s-%s", ServiceName, ServicePort)
	agent := register.Sd.Client.Agent()
	if err := agent.ServiceDeregister(serviceId); err != nil {
		return fmt.Errorf("Failed to deregister service: %s, %s", serviceId, err)
	}
	return nil
}
