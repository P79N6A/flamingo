package kitware

import (
	"context"
	"errors"

	"code.byted.org/gopkg/logs"
	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

// NewDegradationKey gets necessary information from the context and combines them into a degradation key
// /kite/switches/from/from_cluster/to/to_cluster/method
func NewDegradationKey(ctx context.Context) (string, error) {
	keyItems := make([]string, 7, 7)
	keyItems[0], keyItems[1] = "kite", "switches"

	caller, ok := kitutil.GetCtxServiceName(ctx)
	if !ok || caller == "" {
		err := errors.New("No serviceName")
		return "", err
	}
	keyItems[2] = caller

	// there are some normal situations that have no Clusters;
	// to compatible with these situations, we just let thisCluster empty and
	//   it will be ignored when we assemble ETCD key;
	// so if thisCluster is "", it needn't to return an error here.
	thisCluster, ok := kitutil.GetCtxCluster(ctx)
	if ok && thisCluster != DEFAULT_CLUSTER {
		keyItems[3] = thisCluster
	}

	sname, ok := kitutil.GetCtxTargetServiceName(ctx)
	if !ok || sname == "" {
		err := errors.New("No TargetServiceName")
		return "", err
	}
	keyItems[4] = sname

	// same as thisCluster
	targetCluster, ok := kitutil.GetCtxTargetClusterName(ctx)
	if ok && targetCluster != DEFAULT_CLUSTER {
		keyItems[5] = targetCluster
	}

	mname, ok := kitutil.GetCtxTargetMethod(ctx)
	if !ok || mname == "" {
		err := errors.New("No TargetMethod")
		return "", err

	}
	keyItems[6] = mname

	return EtcdKeyJoin(keyItems), nil
}

// Degradater .
type Degradater interface {
	// GetDegradationPercent gets a degradation percent by this key;
	// The returned number should be between [0, 100];
	GetDegradationPercent(key string) (int, error)

	// RandomPercent gets a random number between [0, 100);
	// To avoid using package rand's global lock, this MW use it to get a random number instead of rand.Intn(100);
	// And this function is also necessary for testing.
	RandomPercent() int
}

// NewDegradationMW return a degradation middleware;
// This MW is used to do service degradation;
//
// Description:
//   Using a specific RandomPercent to avoid global lock when using rand.Intn();
//
// Context Requires:
//   1. service name
//   2. cluster - not necessary
//   3. target service name
//   4. target cluster - not necessary
//   5. target method
//
// Context Modify:
//   nothing
func NewDegradationMW(degradater Degradater) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			key, err := NewDegradationKey(ctx)
			if err != nil { // don't do degradation by default
				logs.Errorf("Get degradation key err: %v", err)
				return next(ctx, request)
			}

			per, err := degradater.GetDegradationPercent(key)
			if err != nil { // don't do degradation by default
				logs.Errorf("Get degradation key err: %v", err)
				return next(ctx, request)
			}

			if per == 100 || degradater.RandomPercent() < per {
				kerr := kiterrno.NewKitErr(kiterrno.ForbiddenByDegradationCode, nil)
				return kiterrno.ErrRespForbiddenByDegradation, kerr
			}

			// don't do degradation if there is no key
			return next(ctx, request)
		}
	}
}
