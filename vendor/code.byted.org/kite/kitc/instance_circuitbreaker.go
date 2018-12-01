package kitc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.byted.org/gopkg/circuitbreaker"
	"code.byted.org/kite/kitutil"
)

// InstanceCircuit implement InstanceCircuitBreaker interface
type InstanceCircuit struct {
	panel *circuit.Panel
}

func NewInstanceCircuit() *InstanceCircuit {
	p, _ := circuit.NewPanel(&circuit.Options{
		CoolingTimeout: 3 * time.Second,
		DetectTimeout:  100 * time.Millisecond,
		ShouldTrip:     circuit.RateTripFunc(0.15, 20),
	})
	return &InstanceCircuit{panel: p}
}

func (ic *InstanceCircuit) Timeout(key string) {
	ic.panel.Timeout(key)
}

func (ic *InstanceCircuit) Fail(key string) {
	ic.panel.Fail(key)
}

func (ic *InstanceCircuit) Succeed(key string) {
	ic.panel.Succeed(key)
}

func (ic *InstanceCircuit) Done(key string) {
	ic.panel.Done(key)
}

func (ic *InstanceCircuit) IsAllowed(key string) bool {
	return ic.panel.IsAllowed(key)
}

func (ic *InstanceCircuit) CircuitKey(ctx context.Context) (string, error) {
	ins, ok := kitutil.GetCtxTargetInstance(ctx)
	if !ok {
		return "", errors.New("no target instance in context for instance breaker")
	}
	return fmt.Sprintf("%s:%s", ins.Host(), ins.Port()), nil
}
