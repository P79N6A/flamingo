package kitware

import (
	"context"
	"strings"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
)

// NewIOErrorHandlerMW .
//
// Description:
//  This Middleware parse error returned by thrift, and decide if this
//  error is an I/O error.
//  If it is an I/O error, then return the specific ErrResp the error has.
//
// Context Requires:
//
// Context Modify:
func NewIOErrorHandlerMW() endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			resp, err := next(ctx, request)
			if err == nil {
				return resp, err
			}

			errStr := err.Error()
			if strings.HasPrefix(errStr, "read tcp") &&
				strings.HasSuffix(errStr, "i/o timeout") {
				return kiterrno.ErrRespReadTimeout, err
			}

			if strings.HasPrefix(errStr, "write tcp") &&
				strings.HasSuffix(errStr, "i/o timeout") {
				return kiterrno.ErrRespWriteTimeout, err
			}

			// Closed forcely by remote server
			if strings.Contains(errStr, "connection reset by peer") {
				return kiterrno.ErrRespConnResetByPeer, err
			}

			return resp, err
		}
	}
}
