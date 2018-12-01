package session

import (
	"code.byted.org/gopkg/thrift"
	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitc"
	"code.byted.org/passport/session_lib/goclient/thrift_gen/base"
	"code.byted.org/passport/session_lib/goclient/thrift_gen/session"
	"context"
)

func init() {
	kitc.Register("toutiao.passport.session", &KitcSessionServiceClient{})
}

type KitcSessionServiceClient struct{}

func (c *KitcSessionServiceClient) New(kc *kitc.KitcClient) kitc.Caller {
	t := kitc.NewBufferedTransport(kc)
	f := thrift.NewTBinaryProtocolFactoryDefault()
	client := &session.SessionServiceClient{
		Transport:       t,
		ProtocolFactory: f,
		InputProtocol:   f.GetProtocol(t),
		OutputProtocol:  f.GetProtocol(t),
	}
	return &KitcSessionServiceCaller{client}
}

type KitcSessionServiceCaller struct {
	client *session.SessionServiceClient
}

func (c *KitcSessionServiceCaller) Call(name string, request interface{}) (endpoint.EndPoint, endpoint.KitcCallRequest) {
	switch name {

	case "Add":
		return mkAdd(c.client), &KiteAddRequest{request.(*session.AddRequest)}

	case "Del":
		return mkDel(c.client), &KiteDelRequest{request.(*session.DelRequest)}

	case "Get":
		return mkGet(c.client), &KiteGetRequest{request.(*session.GetRequest)}

	case "Update":
		return mkUpdate(c.client), &KiteUpdateRequest{request.(*session.UpdateRequest)}

	}
	return nil, nil
}

type KiteAddRequest struct {
	*session.AddRequest
}

func (kr *KiteAddRequest) RealRequest() interface{} {
	return kr.AddRequest
}

func (kr *KiteAddRequest) SetBase(kb endpoint.KiteBase) error {

	kr.AddRequest.Base = &base.Base{
		LogID:  kb.GetLogID(),
		Caller: kb.GetCaller(),
		Addr:   kb.GetAddr(),
		Client: kb.GetClient(),
		Extra: map[string]string{
			"cluster": kb.GetCluster(),
			"env":     kb.GetEnv(),
		},
	}

	return nil
}

type KiteDelRequest struct {
	*session.DelRequest
}

func (kr *KiteDelRequest) RealRequest() interface{} {
	return kr.DelRequest
}

func (kr *KiteDelRequest) SetBase(kb endpoint.KiteBase) error {

	kr.DelRequest.Base = &base.Base{
		LogID:  kb.GetLogID(),
		Caller: kb.GetCaller(),
		Addr:   kb.GetAddr(),
		Client: kb.GetClient(),
		Extra: map[string]string{
			"cluster": kb.GetCluster(),
			"env":     kb.GetEnv(),
		},
	}

	return nil
}

type KiteGetRequest struct {
	*session.GetRequest
}

func (kr *KiteGetRequest) RealRequest() interface{} {
	return kr.GetRequest
}

func (kr *KiteGetRequest) SetBase(kb endpoint.KiteBase) error {

	kr.GetRequest.Base = &base.Base{
		LogID:  kb.GetLogID(),
		Caller: kb.GetCaller(),
		Addr:   kb.GetAddr(),
		Client: kb.GetClient(),
		Extra: map[string]string{
			"cluster": kb.GetCluster(),
			"env":     kb.GetEnv(),
		},
	}

	return nil
}

type KiteUpdateRequest struct {
	*session.UpdateRequest
}

func (kr *KiteUpdateRequest) RealRequest() interface{} {
	return kr.UpdateRequest
}

func (kr *KiteUpdateRequest) SetBase(kb endpoint.KiteBase) error {

	kr.UpdateRequest.Base = &base.Base{
		LogID:  kb.GetLogID(),
		Caller: kb.GetCaller(),
		Addr:   kb.GetAddr(),
		Client: kb.GetClient(),
		Extra: map[string]string{
			"cluster": kb.GetCluster(),
			"env":     kb.GetEnv(),
		},
	}

	return nil
}

type KitcAddResponse struct {
	*session.AddResponse
	addr string
}

func (kp *KitcAddResponse) GetBaseResp() endpoint.KiteBaseResp {
	if kp.AddResponse != nil {
		if ret := kp.AddResponse.GetBaseResp(); ret != nil {
			return ret
		}
	}
	return nil
}

func (kp *KitcAddResponse) RemoteAddr() string {
	return kp.addr
}

func (kp *KitcAddResponse) RealResponse() interface{} {
	return kp.AddResponse
}

type KitcDelResponse struct {
	*session.DelResponse
	addr string
}

func (kp *KitcDelResponse) GetBaseResp() endpoint.KiteBaseResp {
	if kp.DelResponse != nil {
		if ret := kp.DelResponse.GetBaseResp(); ret != nil {
			return ret
		}
	}
	return nil
}

func (kp *KitcDelResponse) RemoteAddr() string {
	return kp.addr
}

func (kp *KitcDelResponse) RealResponse() interface{} {
	return kp.DelResponse
}

type KitcGetResponse struct {
	*session.GetResponse
	addr string
}

func (kp *KitcGetResponse) GetBaseResp() endpoint.KiteBaseResp {
	if kp.GetResponse != nil {
		if ret := kp.GetResponse.GetBaseResp(); ret != nil {
			return ret
		}
	}
	return nil
}

func (kp *KitcGetResponse) RemoteAddr() string {
	return kp.addr
}

func (kp *KitcGetResponse) RealResponse() interface{} {
	return kp.GetResponse
}

type KitcUpdateResponse struct {
	*session.UpdateResponse
	addr string
}

func (kp *KitcUpdateResponse) GetBaseResp() endpoint.KiteBaseResp {
	if kp.UpdateResponse != nil {
		if ret := kp.UpdateResponse.GetBaseResp(); ret != nil {
			return ret
		}
	}
	return nil
}

func (kp *KitcUpdateResponse) RemoteAddr() string {
	return kp.addr
}

func (kp *KitcUpdateResponse) RealResponse() interface{} {
	return kp.UpdateResponse
}

func mkAdd(client *session.SessionServiceClient) endpoint.EndPoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		transport := client.Transport.(kitc.Transport)
		err := transport.OpenWithContext(ctx)
		if err != nil {
			return nil, err
		}
		defer transport.Close()
		resp, err := client.Add(request.(endpoint.KitcCallRequest).RealRequest().(*session.AddRequest))
		addr := transport.RemoteAddr()
		return &KitcAddResponse{resp, addr}, err
	}
}

func mkDel(client *session.SessionServiceClient) endpoint.EndPoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		transport := client.Transport.(kitc.Transport)
		err := transport.OpenWithContext(ctx)
		if err != nil {
			return nil, err
		}
		defer transport.Close()
		resp, err := client.Del(request.(endpoint.KitcCallRequest).RealRequest().(*session.DelRequest))
		addr := transport.RemoteAddr()
		return &KitcDelResponse{resp, addr}, err
	}
}

func mkGet(client *session.SessionServiceClient) endpoint.EndPoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		transport := client.Transport.(kitc.Transport)
		err := transport.OpenWithContext(ctx)
		if err != nil {
			return nil, err
		}
		defer transport.Close()
		resp, err := client.Get(request.(endpoint.KitcCallRequest).RealRequest().(*session.GetRequest))
		addr := transport.RemoteAddr()
		return &KitcGetResponse{resp, addr}, err
	}
}

func mkUpdate(client *session.SessionServiceClient) endpoint.EndPoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		transport := client.Transport.(kitc.Transport)
		err := transport.OpenWithContext(ctx)
		if err != nil {
			return nil, err
		}
		defer transport.Close()
		resp, err := client.Update(request.(endpoint.KitcCallRequest).RealRequest().(*session.UpdateRequest))
		addr := transport.RemoteAddr()
		return &KitcUpdateResponse{resp, addr}, err
	}
}
