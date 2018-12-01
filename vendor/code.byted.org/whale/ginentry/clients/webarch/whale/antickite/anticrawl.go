package antickite

import (
	"code.byted.org/gopkg/thrift"
	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitc"
	"code.byted.org/whale/ginentry/thrift_gen/base"
	"code.byted.org/whale/ginentry/thrift_gen/whale/anticrawl"
	"context"
)

var (
	responseBuilder map[string]func() thrift.TStruct
)

func init() {
	responseBuilder = map[string]func() thrift.TStruct{
		"GetDecision": func() thrift.TStruct {
			return &anticrawl.AnticrawlResponse{}
		},
	}
	kitc.Register("webarch.whale.antickite", &KitcAnticrawlServiceClient{})
}

func GetResponseBuilder() (string, map[string]func() thrift.TStruct) {
	return "webarch.whale.antickite", responseBuilder
}

type KitcAnticrawlServiceClient struct{}

func (c *KitcAnticrawlServiceClient) New(kc *kitc.KitcClient) kitc.Caller {
	t := kitc.NewBufferedTransport(kc)
	f := thrift.NewTBinaryProtocolFactoryDefault()
	client := &anticrawl.AnticrawlServiceClient{
		Transport:       t,
		ProtocolFactory: f,
		InputProtocol:   f.GetProtocol(t),
		OutputProtocol:  f.GetProtocol(t),
	}
	return &KitcAnticrawlServiceCaller{client}
}

type KitcAnticrawlServiceCaller struct {
	client *anticrawl.AnticrawlServiceClient
}

func (c *KitcAnticrawlServiceCaller) Call(name string, request interface{}) (endpoint.EndPoint, endpoint.KitcCallRequest) {
	switch name {

	case "GetDecision":
		return mkGetDecision(c.client), &KiteAnticrawlRequest{request.(*anticrawl.AnticrawlRequest)}

	}
	return nil, nil
}

type KiteAnticrawlRequest struct {
	*anticrawl.AnticrawlRequest
}

func (kr *KiteAnticrawlRequest) RealRequest() interface{} {
	return kr.AnticrawlRequest
}

func (kr *KiteAnticrawlRequest) SetBase(kb endpoint.KiteBase) error {

	kr.AnticrawlRequest.Base = &base.Base{
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

type KitcAnticrawlResponse struct {
	*anticrawl.AnticrawlResponse
	addr string
}

func (kp *KitcAnticrawlResponse) GetBaseResp() endpoint.KiteBaseResp {
	if kp.AnticrawlResponse != nil {
		if ret := kp.AnticrawlResponse.GetBaseResp(); ret != nil {
			return ret
		}
	}
	return nil
}

func (kp *KitcAnticrawlResponse) RemoteAddr() string {
	return kp.addr
}

func (kp *KitcAnticrawlResponse) RealResponse() interface{} {
	return kp.AnticrawlResponse
}

func mkGetDecision(client *anticrawl.AnticrawlServiceClient) endpoint.EndPoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		transport := client.Transport.(kitc.Transport)
		err := transport.OpenWithContext(ctx)
		if err != nil {
			return nil, err
		}
		defer transport.Close()
		resp, err := client.GetDecision(request.(endpoint.KitcCallRequest).RealRequest().(*anticrawl.AnticrawlRequest))
		addr := transport.RemoteAddr()
		return &KitcAnticrawlResponse{resp, addr}, err
	}
}
