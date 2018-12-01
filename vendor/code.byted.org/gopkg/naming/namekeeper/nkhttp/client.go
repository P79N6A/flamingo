package nkhttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.byted.org/gopkg/naming/namekeeper"
)

var dialer = net.Dialer{Timeout: 3 * time.Second}

const (
	ClusterSep     = "$"
	ClusterDefault = "default"
)

func SplitHostCluster(s string) (string, string) {
	if idx := strings.Index(s, ClusterSep); idx > 0 {
		return s[:idx], s[idx+1:]
	}
	return s, ""
}

type ClientOption func(t *http.Transport)

func WithMaxIdleConnsPerCluster(n int) ClientOption {
	return func(t *http.Transport) {
		t.MaxIdleConnsPerHost = n
	}
}

func WithMaxIdleConns(n int) ClientOption {
	return func(t *http.Transport) {
		t.MaxIdleConns = n
	}
}

func WithIdleConnTimeout(timeout time.Duration) ClientOption {
	return func(t *http.Transport) {
		t.IdleConnTimeout = timeout
	}
}

func WithKeepAlives(b bool) ClientOption {
	return func(t *http.Transport) {
		t.DisableKeepAlives = !b
	}
}

type HttpClient struct {
	http.Client
}

func isIPAddr(s string) bool {
	h, s, _ := net.SplitHostPort(s)
	return h != "" && net.ParseIP(h) != nil
}

// NewHttpClient returns HttpClient with `http://{YOUR_SERVICE}/path/to/your/api` support
func NewHttpClient(opts ...ClientOption) *HttpClient {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var name string
			nk, err := namekeeper.GetDefaultNamekeeper()
			if err != nil {
				return nil, err
			}
			oo := make([]namekeeper.GetOption, 0, 10)
			oo = append(oo, namekeeper.WithSingleShot(), namekeeper.WithTimeout(500*time.Millisecond))
			if idx := strings.Index(addr, ":"); idx > 0 {
				name = addr[:idx]
			}
			name, cluster := SplitHostCluster(name)
			if cluster == "" {
				cluster = ClusterDefault
			}
			if net.ParseIP(name) != nil {
				return dialer.DialContext(ctx, network, addr)
			}
			oo = append(oo, namekeeper.WithCluster(cluster))
			si, err := nk.Get(name, oo...)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, si.Instances[0].Addr)
		},
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     10 * time.Second,
	}
	for _, op := range opts {
		op(transport)
	}
	c := http.Client{Transport: transport}
	return &HttpClient{Client: c}
}

type callOptions struct {
	ctx     context.Context
	cluster string
	addr    string
}

type CallOption func(opts *callOptions)

func WithContext(ctx context.Context) CallOption {
	return func(opts *callOptions) {
		opts.ctx = ctx
	}
}

func WithCluster(cluster string) CallOption {
	return func(opts *callOptions) {
		opts.cluster = cluster
	}
}

// force use the addr
func WithAddr(addr string) CallOption {
	return func(opts *callOptions) {
		opts.addr = addr
	}
}

func (c *HttpClient) Do(req *http.Request, opts ...CallOption) (*http.Response, error) {
	var copts callOptions
	for _, op := range opts {
		op(&copts)
	}
	if copts.ctx != nil {
		req = req.WithContext(copts.ctx)
	}
	h, cluster := SplitHostCluster(req.URL.Host)
	if copts.cluster != "" {
		cluster = copts.cluster
	}
	if copts.addr != "" && isIPAddr(copts.addr) {
		req.URL.Host = copts.addr
		return c.Client.Do(req)
	}
	if isIPAddr(h) {
		req.URL.Host = h
		return c.Client.Do(req)
	}
	if cluster != "" {
		req.URL.Host = h + ClusterSep + cluster
	}
	return c.Client.Do(req)
}

func (c *HttpClient) Get(url string, opts ...CallOption) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req, opts...)
}

func (c *HttpClient) Head(url string, opts ...CallOption) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req, opts...)
}

func (c *HttpClient) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

func (c *HttpClient) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}
