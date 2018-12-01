package kitware

import (
	"context"
	"math/rand"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

// IDCSelector ..
type IDCSelector interface {
	SelectIDC(policies []kitutil.TPolicy) (string, error) // return a suitable IDC's name
}

// NewIDCSelectorMW create a IDC selector middleware.
//
// Description:
//   This middleware selects a suitable IDC and put it into context.
//   The suitable IDC should be represented by its name, which should be a string.
//   SelectIDC should have no error.
//
// Context Requires:
//	 1. traffic policies; see kitutil.TPolicy
//
// Context Modify:
//   1. put the specific IDC name to context
func NewIDCSelectorMW(idcSelector IDCSelector) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			idc, ok := kitutil.GetCtxConstIDC(ctx)
			if ok {
				ctx = kitutil.NewCtxWithIDC(ctx, idc)
				return next(ctx, request)
			}

			polices, _ := kitutil.GetCtxTPolicy(ctx)
			// if polices is nil or empty, defaultIDC will be used
			idc, err := idcSelector.SelectIDC(polices)
			if err != nil {
				kerr := kiterrno.NewKitErr(kiterrno.IDCSelectErrorCode, err)
				return kiterrno.ErrRespIDCSelectError, kerr
			}
			ctx = kitutil.NewCtxWithIDC(ctx, idc)
			return next(ctx, request)
		}
	}
}

// NewDefaultIDCSelectorMW ...
func NewDefaultIDCSelectorMW(localIDC string) endpoint.Middleware {
	return NewIDCSelectorMW(NewDefaultIDCSelector(localIDC))
}

// DefaultIDCSelector ...
type DefaultIDCSelector struct {
	localIDC string
}

// NewDefaultIDCSelector ...
func NewDefaultIDCSelector(localIDC string) *DefaultIDCSelector {
	return &DefaultIDCSelector{
		localIDC: localIDC,
	}
}

// SelectIDC ...
func (is *DefaultIDCSelector) SelectIDC(policies []kitutil.TPolicy) (string, error) {
	if len(policies) == 0 {
		return is.localIDC, nil
	}

	var sum int64
	for _, pl := range policies {
		sum += pl.Percent
	}

	if sum == 0 {
		return is.localIDC, nil
	}

	rd := rand.Int63n(sum)
	for _, pl := range policies {
		if rd < pl.Percent {
			return pl.IDC, nil
		}
		rd -= pl.Percent
	}
	return policies[0].IDC, nil
}
