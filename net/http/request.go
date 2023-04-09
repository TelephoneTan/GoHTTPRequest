package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/TelephoneTan/GoHTTPRequest/net"
	"github.com/TelephoneTan/GoHTTPRequest/net/http/header"
	"github.com/TelephoneTan/GoHTTPRequest/net/http/method"
	"github.com/TelephoneTan/GoHTTPRequest/net/mime"
	"github.com/TelephoneTan/GoHTTPRequest/util"
	"github.com/TelephoneTan/GoPromise/async/promise"
	"github.com/TelephoneTan/GoPromise/async/task"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	defaultIsQuickTest    = false
	defaultFollowRedirect = true
	defaultConnectTimeout = Duration(2 * time.Second)
	defaultReadTimeout    = Duration(20 * time.Second)
	defaultWriteTimeout   = Duration(20 * time.Second)
	quickTestTimeout      = Duration(500 * time.Millisecond)
)

type Stream struct {
	Reader io.Reader
	Done   func()
}

type Result[T any] struct {
	Request Request
	Result  T
}

type ctxPack struct {
	ctx    context.Context
	cancel func()
}

type Binary []byte

func (b *Binary) UnmarshalJSON(bs []byte) error {
	var s64 *string
	err := json.Unmarshal(bs, &s64)
	if err != nil {
		return err
	}
	if s64 == nil {
		*b = nil
		return nil
	} else {
		bin, err := base64.StdEncoding.DecodeString(*s64)
		if err != nil {
			return err
		}
		*b = bin
		return nil
	}
}

func (b *Binary) MarshalJSON() ([]byte, error) {
	if *b == nil {
		return json.Marshal(nil)
	} else {
		return json.Marshal(base64.StdEncoding.EncodeToString(*b))
	}
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(bs []byte) error {
	var ms *int64
	err := json.Unmarshal(bs, &ms)
	if err != nil {
		return err
	}
	if ms == nil {
		return nil
	} else {
		*d = Duration(*ms)
		return nil
	}
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*d).Milliseconds())
}

type ResponseBinary struct {
	bin Binary
}

func (r *ResponseBinary) UnmarshalJSON(bs []byte) error {
	return r.bin.UnmarshalJSON(bs)
}

func (r *ResponseBinary) MarshalJSON() ([]byte, error) {
	return r.bin.MarshalJSON()
}

type HeaderMap map[string][]string

func (h *HeaderMap) UnmarshalJSON(bs []byte) error {
	var x *[][]any
	err := json.Unmarshal(bs, x)
	if err != nil {
		return err
	}
	if x == nil {
		*h = nil
		return nil
	} else {
		*h = map[string][]string{}
		for _, kv := range *x {
			var k string
			var v []string
			if len(kv) > 0 {
				k, _ = kv[0].(string)
				if len(kv) > 1 {
					v, _ = kv[1].([]string)
				}
			}
			if v == nil {
				v = []string{}
			}
			(*h)[k] = v
		}
		return nil
	}
}

func (h *HeaderMap) MarshalJSON() ([]byte, error) {
	var x [][]any
	for k, v := range *h {
		x = append(x, []any{k, v})
	}
	return json.Marshal(x)
}

type _Request struct {
	RequestSemaphore         promise.Semaphore `json:"-"`
	Method                   *method.Method
	URL                      string
	CustomizedHeaderList     [][]string
	RequestBinary            Binary
	RequestForm              [][]string
	RequestString            string
	RequestFile              *os.File  `json:"-"`
	RequestBody              io.Reader `json:"-"`
	RequestContentType       *mime.Type
	RequestContentTypeHeader string
	Timeout                  *Duration
	ConnectTimeout           *Duration
	ReadTimeout              *Duration
	WriteTimeout             *Duration
	IsQuickTest              *bool
	FollowRedirect           *bool
	CookieJar                FlexibleCookieJar `json:"-"`
	AutoSendCookies          *bool
	AutoReceiveCookies       *bool
	Proxy                    *net.Proxy
	//
	StatusCode         int
	StatusMessage      string
	ResponseHeaderList [][]string
	ResponseHeaderMap  HeaderMap
	//
	ResponseBinary ResponseBinary `json:"responseBinary"`
	//
	contentLength int64
	getBody       func() (io.ReadCloser, error)
	//
	transport *http.Transport
	client    *http.Client
	//
	context *atomic.Pointer[ctxPack]
	//
	stream       task.Once[Result[Stream]]
	send         task.Once[Result[any]]
	byteSlice    task.Once[Result[[]byte]]
	string       task.Once[Result[string]]
	json         task.Once[Result[any]]
	htmlDocument task.Once[Result[*html.Node]]
}

type Request = *_Request

var transportPool = sync.Pool{New: func() any { return &http.Transport{} }}
var clientPool = sync.Pool{New: func() any { return &http.Client{} }}

func (r Request) calContentType() string {
	if r.RequestContentType != nil {
		return string(*r.RequestContentType)
	}
	return r.RequestContentTypeHeader
}

func (r Request) generateRequestBody() io.Reader {
	if len(r.RequestForm) > 0 {
		sb := strings.Builder{}
		for i, kv := range r.RequestForm {
			if len(kv) > 0 {
				if i > 0 {
					sb.WriteString("&")
				}
				sb.WriteString(url.QueryEscape(kv[0]))
				if len(kv) > 1 {
					sb.WriteString("=")
					sb.WriteString(url.QueryEscape(kv[1]))
				}
			}
		}
		if sb.Len() > 0 {
			r.RequestString = sb.String()
			r.RequestContentType = &mime.XWWWFormURLEncoded
		}
	}
	if r.RequestString != "" {
		r.RequestBody = strings.NewReader(r.RequestString)
		if r.calContentType() == "" {
			r.RequestContentType = &mime.TextPlainUTF8
		}
	}
	if len(r.RequestBinary) > 0 {
		r.RequestBody = bytes.NewReader(r.RequestBinary)
		if r.calContentType() == "" {
			r.RequestContentTypeHeader = http.DetectContentType(r.RequestBinary)
		}
	}
	if r.RequestFile != nil {
		fi, err := r.RequestFile.Stat()
		if err != nil {
			panic(err)
		}
		r.contentLength = fi.Size()
		r.getBody = func() (io.ReadCloser, error) {
			return os.Open(r.RequestFile.Name())
		}
		r.RequestBody = r.RequestFile
		if r.calContentType() == "" {
			r.RequestContentType = &mime.ApplicationOctetStream
		}
	}
	if r.RequestBody != nil {
		if r.calContentType() == "" {
			r.RequestContentType = &mime.ApplicationOctetStream
		}
	}
	return r.RequestBody
}

func (r Request) generateRequestMethod() string {
	if r.Method == nil {
		r.Method = &method.GET
	}
	return string(*r.Method)
}

func (r Request) applyRequestHeaders(request *http.Request) {
	ct := r.calContentType()
	if ct != "" {
		request.Header.Set(header.ContentType, ct)
	}
	if r.contentLength > 0 {
		request.Header.Set(header.ContentLength, strconv.FormatInt(r.contentLength, 10))
	}
	for _, kv := range r.CustomizedHeaderList {
		if len(kv) > 0 {
			k := kv[0]
			var v string
			if len(kv) > 1 {
				v = kv[1]
			}
			request.Header.Add(k, v)
		}
	}
}

func (r Request) generateFollowRedirect() bool {
	if r.FollowRedirect == nil {
		r.FollowRedirect = &defaultFollowRedirect
	}
	return *r.FollowRedirect
}

func (r Request) generateTimeout() {
	if r.ConnectTimeout == nil {
		r.ConnectTimeout = &defaultConnectTimeout
	}
	if r.ReadTimeout == nil {
		r.ReadTimeout = &defaultReadTimeout
	}
	if r.WriteTimeout == nil {
		r.WriteTimeout = &defaultWriteTimeout
	}
	if r.Timeout == nil {
		timeout := *r.ConnectTimeout + *r.ReadTimeout + *r.WriteTimeout
		r.Timeout = &timeout
	}
	if r.IsQuickTest == nil {
		r.IsQuickTest = &defaultIsQuickTest
	}
	if *r.IsQuickTest {
		r.ConnectTimeout = &quickTestTimeout
		r.ReadTimeout = &quickTestTimeout
		r.WriteTimeout = &quickTestTimeout
		timeout := *r.ConnectTimeout + *r.ReadTimeout + *r.WriteTimeout
		r.Timeout = &timeout
	}
}

func (r Request) generateCookieJar() FlexibleCookieJar {
	if r.CookieJar != nil {
		if r.AutoSendCookies != nil && r.AutoReceiveCookies != nil {
			r.CookieJar = r.CookieJar.WithReadWrite(*r.AutoSendCookies, *r.AutoReceiveCookies)
		} else if r.AutoSendCookies != nil {
			r.CookieJar = r.CookieJar.WithRead(*r.AutoSendCookies)
		} else if r.AutoReceiveCookies != nil {
			r.CookieJar = r.CookieJar.WithWrite(*r.AutoReceiveCookies)
		}
	}
	return r.CookieJar
}

func (r Request) generateTransport(request *http.Request) *http.Transport {
	r.generateTimeout()
	r.transport = transportPool.Get().(*http.Transport)
	r.transport.TLSHandshakeTimeout = time.Duration(*r.ConnectTimeout)
	var u *url.URL
	if r.Proxy != nil {
		u = r.Proxy.URL()
	}
	if u == nil {
		if eu, err := http.ProxyFromEnvironment(request); eu != nil && err == nil {
			u = eu
		}
	}
	if u != nil {
		r.transport.Proxy = http.ProxyURL(u)
	} else {
		r.transport.Proxy = nil
	}
	return r.transport
}

func (r Request) generateClient(request *http.Request) *http.Client {
	r.generateTransport(request)
	r.generateFollowRedirect()
	r.generateCookieJar()
	r.client = clientPool.Get().(*http.Client)
	r.client.Transport = r.transport
	r.client.Timeout = time.Duration(*r.Timeout)
	r.client.Jar = r.CookieJar
	if !*r.FollowRedirect {
		r.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		r.client.CheckRedirect = nil
	}
	return r.client
}

func (r Request) Cancel() bool {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return r.context.CompareAndSwap(nil, &ctxPack{ctx: ctx, cancel: cancel})
}

func (r Request) getContext() *ctxPack {
	ctx, cancel := context.WithCancel(context.Background())
	newCTX := &ctxPack{ctx: ctx, cancel: cancel}
	if r.context.CompareAndSwap(nil, newCTX) {
		return newCTX
	} else {
		return r.context.Load()
	}
}

func (r Request) GetHeader(name string) []string {
	for n, vs := range r.ResponseHeaderMap {
		if strings.EqualFold(n, name) {
			return vs
		}
	}
	return nil
}

func (r Request) stringTask(defaultCharset string) task.Once[Result[string]] {
	return task.NewOnceTask(promise.Job[Result[string]]{
		Do: func(rs promise.Resolver[Result[string]], re promise.Rejector) {
			rs.ResolvePromise(promise.Then(r.byteSlice.Do(), promise.FulfilledListener[Result[[]byte], Result[string]]{
				OnFulfilled: func(bsRes Result[[]byte]) any {
					var ct string
					if headers := bsRes.Request.GetHeader(header.ContentType); len(headers) > 0 {
						ct = headers[0]
					}
					if ct == "" {
						ct = "text/plain;charset=" + defaultCharset
					}
					reader, err := charset.NewReader(bytes.NewReader(bsRes.Result), ct)
					if err != nil {
						panic(err)
					}
					bb := bytes.Buffer{}
					_, err = bb.ReadFrom(reader)
					if err != nil {
						panic(err)
					}
					return Result[string]{
						Request: r,
						Result:  bb.String(),
					}
				},
			}))
		},
	})
}

func (r Request) htmlTask(defaultCharset string) task.Once[Result[*html.Node]] {
	return task.NewOnceTask(promise.Job[Result[*html.Node]]{
		Do: func(rs promise.Resolver[Result[*html.Node]], re promise.Rejector) {
			var strPromise promise.Promise[Result[string]]
			if defaultCharset == "" {
				strPromise = r.String()
			} else {
				strPromise = r.StringWithCharset(defaultCharset)
			}
			rs.ResolvePromise(promise.Then(strPromise, promise.FulfilledListener[Result[string], Result[*html.Node]]{
				OnFulfilled: func(strRes Result[string]) any {
					node, err := html.Parse(strings.NewReader(strRes.Result))
					if err != nil {
						panic(err)
					}
					return Result[*html.Node]{
						Request: r,
						Result:  node,
					}
				},
			}))
		},
	})
}

func (r Request) init() Request {
	r.context = &atomic.Pointer[ctxPack]{}
	r.stream = task.NewOnceTask(promise.Job[Result[Stream]]{
		Do: func(rs promise.Resolver[Result[Stream]], re promise.Rejector) {
			ok := false
			ctx := r.getContext()
			cancelContext := func() {
				ctx.cancel()
			}
			defer func() {
				if !ok {
					cancelContext()
				}
			}()
			//
			request, err := http.NewRequestWithContext(ctx.ctx, r.generateRequestMethod(), r.URL, r.generateRequestBody())
			if err != nil {
				panic(err)
			}
			if r.getBody != nil {
				request.GetBody = r.getBody
			}
			//
			r.applyRequestHeaders(request)
			//
			response, err := r.generateClient(request).Do(request)
			recycleClient := func() {
				clientPool.Put(r.client)
			}
			recycleTransport := func() {
				transportPool.Put(r.transport)
			}
			defer func() {
				if !ok {
					recycleTransport()
				}
			}()
			defer func() {
				if !ok {
					recycleClient()
				}
			}()
			closeBody := func() {
				_ = response.Body.Close()
			}
			if err != nil {
				panic(err)
			} else {
				defer func() {
					if !ok {
						closeBody()
					}
				}()
			}
			//
			r.StatusCode = response.StatusCode
			r.StatusMessage = response.Status
			//
			r.ResponseHeaderMap = map[string][]string{}
			if response.Header != nil {
				r.ResponseHeaderMap = HeaderMap(response.Header)
			}
			//
			for k, vs := range r.ResponseHeaderMap {
				for _, v := range vs {
					r.ResponseHeaderList = append(r.ResponseHeaderList, []string{k, v})
				}
			}
			//
			rs.ResolveValue(Result[Stream]{
				Request: r,
				Result: Stream{
					Reader: response.Body,
					Done: func() {
						defer recycleClient()
						defer recycleTransport()
						defer cancelContext()
						defer closeBody()
						_, _ = io.Copy(io.Discard, response.Body)
					},
				},
			})
			ok = true
		},
	})
	r.send = task.NewOnceTask(promise.Job[Result[any]]{
		Do: func(rs promise.Resolver[Result[any]], re promise.Rejector) {
			rs.ResolvePromise(promise.Then(r.stream.Do(), promise.FulfilledListener[Result[Stream], Result[any]]{
				OnFulfilled: func(streamRes Result[Stream]) any {
					defer streamRes.Result.Done()
					return Result[any]{
						Request: r,
						Result:  r,
					}
				},
			}))
		},
	})
	r.byteSlice = task.NewOnceTask(promise.Job[Result[[]byte]]{
		Do: func(rs promise.Resolver[Result[[]byte]], re promise.Rejector) {
			rs.ResolvePromise(promise.Then(r.stream.Do(), promise.FulfilledListener[Result[Stream], Result[[]byte]]{
				OnFulfilled: func(streamRes Result[Stream]) any {
					defer streamRes.Result.Done()
					var err error
					r.ResponseBinary.bin, err = io.ReadAll(streamRes.Result.Reader)
					if err != nil {
						panic(err)
					}
					if r.ResponseBinary.bin == nil {
						r.ResponseBinary.bin = []byte{}
					}
					return Result[[]byte]{
						Request: r,
						Result:  r.ResponseBinary.bin,
					}
				},
			}))
		},
	})
	r.string = r.stringTask("utf-8")
	r.json = task.NewOnceTask(promise.Job[Result[any]]{
		Do: func(rs promise.Resolver[Result[any]], re promise.Rejector) {
			rs.ResolvePromise(promise.Then(r.byteSlice.Do(), promise.FulfilledListener[Result[[]byte], Result[any]]{
				OnFulfilled: func(bsRes Result[[]byte]) any {
					var res any
					err := json.Unmarshal(bsRes.Result, &res)
					if err != nil {
						panic(err)
					}
					return Result[any]{
						Request: r,
						Result:  res,
					}
				},
			}))
		},
	})
	r.htmlDocument = r.htmlTask("")
	return r
}

func NewRequest(init ...func(Request)) Request {
	return util.New(new(_Request).init(), init...)
}

func (r Request) Clone() Request {
	return util.Copy(*r, func(clone Request) {
		clone.init()
		if clone.CustomizedHeaderList != nil {
			clone.CustomizedHeaderList = append([][]string{}, clone.CustomizedHeaderList...)
			for i, kv := range clone.CustomizedHeaderList {
				if kv != nil {
					clone.CustomizedHeaderList[i] = append([]string{}, kv...)
				}
			}
		}
		if clone.RequestForm != nil {
			clone.RequestForm = append([][]string{}, clone.RequestForm...)
			for i, kv := range clone.RequestForm {
				if kv != nil {
					clone.RequestForm[i] = append([]string{}, kv...)
				}
			}
		}
		if clone.RequestBinary != nil {
			clone.RequestBinary = append([]byte{}, clone.RequestBinary...)
		}
	})
}

func (r Request) Serialize() string {
	bs, _ := json.Marshal(r)
	return string(bs)
}

func (r Request) Deserialize(s string) Request {
	_ = json.Unmarshal([]byte(s), r)
	return r
}

func (r Request) Stream() promise.Promise[Result[Stream]] {
	return r.stream.Do()
}

func (r Request) Send() promise.Promise[Result[any]] {
	return r.send.Do()
}

func (r Request) ByteSlice() promise.Promise[Result[[]byte]] {
	return r.byteSlice.Do()
}

func (r Request) String() promise.Promise[Result[string]] {
	return r.string.Do()
}

func (r Request) StringWithCharset(charset string) promise.Promise[Result[string]] {
	return r.stringTask(charset).Do()
}

func (r Request) Json() promise.Promise[Result[any]] {
	return r.json.Do()
}

func (r Request) HTMLDocument() promise.Promise[Result[*html.Node]] {
	return r.htmlDocument.Do()
}

func (r Request) HTMLDocumentWithCharset(charset string) promise.Promise[Result[*html.Node]] {
	return r.htmlTask(charset).Do()
}
