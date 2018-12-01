package kite

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"code.byted.org/gopkg/logs"
)

// Runc start default RpcService
func Run() error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- RpcService.Serve()
	}()
	return waitSignal(errCh)
}

func waitSignal(errCh chan error) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

WaitSignal:
	for {
		select {
		case sig := <-signals:
			switch sig {
			// exit forcely
			case syscall.SIGTERM:
				StopRegister()
				return errors.New(sig.String())
			case syscall.SIGHUP, syscall.SIGINT:
				StopRegister()
				RpcService.Stop()
				break WaitSignal
			}
		case err := <-errCh:
			StopRegister()
			// Not expected error
			return err
		}
	}
	err := <-errCh
	if err != nil {
		logs.Fatal("AcceptLoop error: %s\n", err)
	}
	return err
}
