package tos

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.byted.org/gopkg/naming/namekeeper/nkhttp"
)

const (
	TosAccessHeader = "X-Tos-Access"

	tosService string = "toutiao.tos.tosapi"

	MinPartSize int64 = 5 * 1024 * 1024
)

var DefaultReqTimeout = 10 * time.Second

type Config struct {
	Cluster string
	// bucket -> accessKey
	KeyMap              map[string]string
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
}

type Tos struct {
	addrs  atomic.Value
	c      *Config
	client *nkhttp.HttpClient
}

// NewTos return a tos client instance
func NewTos(c *Config) (*Tos, error) {
	if c.MaxIdleConnsPerHost == 0 {
		c.MaxIdleConnsPerHost = 10
	}
	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = time.Minute
	}
	if c.Cluster == "" {
		c.Cluster = "default"
	}

	var opts []nkhttp.ClientOption
	opts = append(opts, nkhttp.WithIdleConnTimeout(c.IdleConnTimeout))
	opts = append(opts, nkhttp.WithMaxIdleConns(10*c.MaxIdleConnsPerHost))
	client := nkhttp.NewHttpClient(opts...)
	t := &Tos{c: c, client: client}
	return t, nil
}

func (t *Tos) uri(bucket, object string) string {
	if v := os.Getenv("TEST_TOSAPI_ADDR"); v != "" {
		if object == "" {
			return fmt.Sprintf("http://%s/%s", v, bucket)
		}
		return fmt.Sprintf("http://%s/%s/%s", v, bucket, object)
	}
	if object == "" {
		return fmt.Sprintf("http://%s/%s", tosService, bucket)
	}
	return fmt.Sprintf("http://%s/%s/%s", tosService, bucket, object)
}

func (t *Tos) doReq(ctx context.Context, req *http.Request) (*http.Response, error) {
	if ctx == nil {
		panic("ctx is nil")
	}
	var timeout time.Duration
	deadline, ok := ctx.Deadline()
	if !ok {
		timeout = DefaultReqTimeout
	} else {
		timeout = deadline.Sub(time.Now())
	}

	val := req.URL.Query()
	val.Add("timeout", timeout.String())
	req.URL.RawQuery = val.Encode()
	req = req.WithContext(ctx)
	return t.client.Do(req, nkhttp.WithCluster(t.c.Cluster))
}

type ListPrefixInput struct {
	Prefix     string
	Delimiter  string
	StartAfter string
	MaxKeys    int
}

type ListObject struct {
	Key          string `json:"key"`
	LastModified string `json:"lastModified"`
	Size         int64  `json:"size"`
}

type ListPrefixOutput struct {
	IsTruncated  bool         `json:"isTruncated"`
	CommonPrefix []string     `json:"commonPrefix"`
	Objects      []ListObject `json:"objects"`
}

type listPrefixRes struct {
	Success int              `json:"success"`
	Payload ListPrefixOutput `json:"payload"`
}

func (t *Tos) ListPrefix(ctx context.Context, bucket string, input ListPrefixInput) (*ListPrefixOutput, error) {
	uri := t.uri(bucket, "")
	req, _ := http.NewRequest(http.MethodGet, uri, nil)
	val := req.URL.Query()
	val.Add("prefix", input.Prefix)
	val.Add("delimiter", input.Delimiter)
	val.Add("start-after", input.StartAfter)
	val.Add("max-keys", strconv.Itoa(input.MaxKeys))
	req.URL.RawQuery = val.Encode()
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, DecodeErr(res)
	}
	listRes := &listPrefixRes{}
	err = json.NewDecoder(res.Body).Decode(listRes)
	if err != nil {
		return nil, err
	}
	return &listRes.Payload, nil
}

type ObjectInfo struct {
	R            io.ReadCloser
	Size         int64
	LastModified string
}

// INTERNAL FUNC, DONOT USE!
func (t *Tos) GetObjectRaw(ctx context.Context, bucket, object string) (*http.Response, error) {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodGet, uri, nil)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	return t.doReq(ctx, req)
}

func (t *Tos) GetObject2(ctx context.Context, bucket, object string) (*ObjectInfo, error) {
	res, err := t.GetObjectRaw(ctx, bucket, object)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, DecodeErr(res)
	}
	size, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, err
	}
	lastModified := res.Header.Get("Last-Modified")
	return &ObjectInfo{
		R:            res.Body,
		Size:         size,
		LastModified: lastModified,
	}, nil
}

// GetObject read object from remote and write object into w
func (t *Tos) GetObject(ctx context.Context, bucket, object string, w io.Writer) error {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodGet, uri, nil)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return DecodeErr(res)
	}
	_, err = io.Copy(w, res.Body)
	if err != nil {
		return err
	}
	return nil
}

// GetRangeObject read object with special begin, end offset.
func (t *Tos) GetRangeObject(ctx context.Context, bucket, object string, begin, end int64, w io.Writer) error {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodGet, uri, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", begin, end))
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusPartialContent {
		return DecodeErr(res)
	}
	_, err = io.Copy(w, res.Body)
	return err
}

func (t *Tos) HttpForward(ctx context.Context, bucket, object string, w http.ResponseWriter, r *http.Request) error {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(r.Method, uri, nil)
	for k, v := range r.Header {
		req.Header[k] = v
	}
	if _, ok := req.Header[TosAccessHeader]; !ok {
		req.Header.Set(TosAccessHeader, t.c.KeyMap[bucket])
	}
	resp, err := t.doReq(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return DecodeErr(resp)
	}
	wheader := w.Header()
	for k, vv := range resp.Header {
		if _, ok := wheader[k]; ok {
			continue // not overwrite
		}
		for _, v := range vv {
			wheader.Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return nil
}

type HeadObjRes struct {
	Size         int64
	LastModified time.Time
}

// HeadObject return an object's meta info
func (t *Tos) HeadObject(ctx context.Context, bucket, object string) (*HeadObjRes, error) {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodHead, uri, nil)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, DecodeErr(res)
	}
	size, _ := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	lmt, _ := time.Parse(http.TimeFormat, res.Header.Get("Last-Modified"))

	return &HeadObjRes{Size: size, LastModified: lmt}, nil
}

// PutObject write an object into the storage server
func (t *Tos) PutObject(ctx context.Context, bucket, object string, size int64, r io.Reader) error {
	uri := t.uri(bucket, object)
	m := md5.New()
	teeReader := io.TeeReader(r, m)
	req, _ := http.NewRequest(http.MethodPut, uri, teeReader)
	req.ContentLength = size
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return DecodeErr(res)
	}

	md5Hash := hex.EncodeToString(m.Sum(nil))
	if md5Hash != res.Header.Get("X-Tos-MD5") {
		return ErrChecksum
	}
	return nil
}

// PutLargeObject which object is larger than 10MB
func (t *Tos) PutLargeObject(ctx context.Context, bucket, object string, size int64, r io.Reader) error {
	if size < MinPartSize {
		return ErrContentTooSmall
	}
	left := size
	uploadID, err := t.InitUpload(ctx, bucket, object)
	if err != nil {
		return err
	}

	var (
		mu     sync.Mutex
		retErr []error

		partCnt = 0
	)

	uploadPartsResult := make(map[int]UploadPartResult)

	var PartSize int64 = 8 * 1024 * 1024
	if size/PartSize >= 10000 {
		PartSize = (((size / 10000) >> 10) + 1) << 10
	}

	lc := NewLimitCurrent(10)
	for ; left > 0; partCnt++ {
		limitSize := PartSize
		if left-PartSize < MinPartSize {
			limitSize = left
		}
		left -= limitSize
		partData, err := ioutil.ReadAll(io.LimitReader(r, limitSize))
		if err != nil {
			return err
		}

		idx, data := partCnt, partData
		lc.Do(func() {
			// partNumber should begin with 1, not 0
			partNumber := idx + 1
			uploadRet, err := t.UploadPart2(ctx, bucket, object, strconv.Itoa(partNumber), uploadID, limitSize, bytes.NewBuffer(data))
			mu.Lock()
			retErr = append(retErr, err)
			if err == nil {
				uploadPartsResult[partNumber] = *uploadRet
			}
			mu.Unlock()
		})
	}
	lc.Wait()

	for _, err := range retErr {
		//TODO(xiangchao): retry
		if err != nil {
			return err
		}
	}
	allParts := make([]UploadPartResult, partCnt)
	for i := 0; i < partCnt; i++ {
		allParts[i] = uploadPartsResult[i+1]
	}
	_, err = t.CompleteUpload2(ctx, bucket, object, uploadID, allParts)
	if err != nil {
		return err
	}
	return nil
}

// DelObject delete an object
func (t *Tos) DelObject(ctx context.Context, bucket, object string) error {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodDelete, uri, nil)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		return DecodeErr(res)
	}
	return nil
}

type initRes struct {
	Success int `json:"success"`
	Payload struct {
		UploadID string `json:"uploadID"`
	} `json:"payload"`
}

// InitUpload init an upload session
func (t *Tos) InitUpload(ctx context.Context, bucket, object string) (string, error) {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodPost, uri+"?uploads", nil)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		decoder := json.NewDecoder(res.Body)
		jsonRes := new(initRes)
		err := decoder.Decode(jsonRes)
		if err != nil {
			return "", err
		}
		return jsonRes.Payload.UploadID, nil
	}
	return "", DecodeErr(res)
}

type UploadPartResult struct {
	UploadID string
	PartID   string
	Etag     string
}

// UploadPart2 upload a part, then get a result.
func (t *Tos) UploadPart2(ctx context.Context, bucket, object, partNumber, uploadID string, size int64, r io.Reader) (*UploadPartResult, error) {
	uri := t.uri(bucket, object)
	m := md5.New()
	teeReader := io.TeeReader(r, m)
	req, _ := http.NewRequest(http.MethodPut, uri+fmt.Sprintf("?partNumber=%s&uploadID=%s", partNumber, uploadID), teeReader)
	req.ContentLength = size
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, DecodeErr(res)
	}
	md5Hash := hex.EncodeToString(m.Sum(nil))
	if md5Hash != res.Header.Get("X-Tos-MD5") {
		return nil, ErrChecksum
	}
	return &UploadPartResult{
		UploadID: uploadID,
		PartID:   partNumber,
		Etag:     res.Header.Get("X-Tos-ETag"),
	}, nil
}

// deprecated, must use UploadPart2 on aliyun or s3
// UploadPart upload a part with uploadID and partNumber
func (t *Tos) UploadPart(ctx context.Context, bucket, object, partNumber, uploadID string, size int64, r io.Reader) error {
	uri := t.uri(bucket, object)
	m := md5.New()
	teeReader := io.TeeReader(r, m)
	req, _ := http.NewRequest(http.MethodPut, uri+fmt.Sprintf("?partNumber=%s&uploadID=%s", partNumber, uploadID), teeReader)
	req.ContentLength = size
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return DecodeErr(res)
	}
	md5Hash := hex.EncodeToString(m.Sum(nil))
	if md5Hash != res.Header.Get("X-Tos-MD5") {
		return ErrChecksum
	}
	return nil
}

// deprecated, must use CompleteUpload2 on aliyun or aws
func (t *Tos) CompleteUpload(ctx context.Context, bucket, object, uploadID string, partList []string) (string, error) {
	uri := t.uri(bucket, object)
	body := bytes.NewBufferString(strings.Join(partList, ","))
	req, _ := http.NewRequest(http.MethodPost, uri+fmt.Sprintf("?uploadID=%s", uploadID), body)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", DecodeErr(res)
	}
	// deprecated, in order to make complete more efficient, complete uploader won't return all file MD5.
	return res.Header.Get("X-Tos-MD5"), nil
}

//
// CompleteUpload2 ...
func (t *Tos) CompleteUpload2(ctx context.Context, bucket, object, uploadID string, parts []UploadPartResult) (string, error) {
	uri := t.uri(bucket, object)
	var partList []string
	for _, part := range parts {
		if part.Etag != "" {
			partList = append(partList, part.PartID+":"+part.Etag)
		} else {
			partList = append(partList, part.PartID)
		}
	}
	body := bytes.NewBufferString(strings.Join(partList, ","))
	req, _ := http.NewRequest(http.MethodPost, uri+fmt.Sprintf("?uploadID=%s", uploadID), body)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", DecodeErr(res)
	}
	// deprecated, in order to make complete more efficient, complete uploader won't return all file MD5.
	return res.Header.Get("X-Tos-MD5"), nil
}

type listRes struct {
	Success int `json:"success"`
	Payload struct {
		UploadID string   `json:"uploadID"`
		PartList []string `json:"partList"`
	} `json:"payload"`
}

// ListParts return a list of all parts have uploaded
func (t *Tos) ListParts(ctx context.Context, bucket, object, uploadID string) ([]string, error) {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodGet, uri+fmt.Sprintf("?uploadID=%s", uploadID), nil)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, DecodeErr(res)
	}
	decoder := json.NewDecoder(res.Body)
	lRes := new(listRes)
	err = decoder.Decode(lRes)
	if err != nil {
		return nil, err
	}
	return lRes.Payload.PartList, nil
}

// AbortUpload abort an upload session with the uploadID
func (t *Tos) AbortUpload(ctx context.Context, bucket, object, uploadID string) error {
	uri := t.uri(bucket, object)
	req, _ := http.NewRequest(http.MethodDelete, uri+fmt.Sprintf("?uploadID=%s", uploadID), nil)
	req.Header.Add(TosAccessHeader, t.c.KeyMap[bucket])
	res, err := t.doReq(ctx, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return DecodeErr(res)
	}
	return nil
}
