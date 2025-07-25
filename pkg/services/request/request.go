package request

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

type Response struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// RequestService defines the interface for making HTTP requests
type RequestService interface {
	Post(ctx context.Context, url string, body any, response any) (Response, error)
	Get(ctx context.Context, url string, response any) (Response, error)
}

// requestService implements the RequestService interface
type requestService struct {
	// client *resty.Client
	baseURL string
	client  *fasthttp.Client
	*options
}

type options struct {
	afterSuccessRequestFunc afterSuccessRequest
	afterRequestFunc        afterRequest
}

type afterSuccessRequest func(any, any)
type afterRequest func(success bool, duration time.Duration)

// WithAfterSuccessRequestFunc sets a function to be called after each request
func WithAfterSuccessRequestFunc(fn afterSuccessRequest) func(*options) {
	return func(opts *options) {
		opts.afterSuccessRequestFunc = fn
	}
}

// WithAfterRequestFunc sets a function to be called after each request on error
func WithAfterRequestFunc(fn afterRequest) func(*options) {
	return func(opts *options) {
		opts.afterRequestFunc = fn
	}
}

// NewRequestService creates a new instance of requestService
func NewRequestService(baseUrl string, opts ...func(*options)) RequestService {

	rs := &requestService{
		baseURL: baseUrl,
		options: &options{},
	}

	for _, opt := range opts {
		opt(rs.options)
	}
	return rs
}

// Post sends a POST request to the specified URI with the given body
func (r *requestService) Post(ctx context.Context, uri string, body any, response any) (Response, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.Header.SetMethod(http.MethodPost)
	req.Header.Set("content-type", "application/json")
	req.SetRequestURI(r.baseURL + uri)
	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	req.SetBody(payload)
	start := time.Now()
	err = fasthttp.Do(req, resp)
	if err != nil {
		return Response{}, err
	}
	bodyResp := resp.Body()

	end := time.Since(start)

	ret := Response{
		StatusCode: resp.StatusCode(),
		Message:    string(bodyResp),
	}

	success := isSuccess(ret.StatusCode)
	if r.options.afterRequestFunc != nil {
		r.options.afterRequestFunc(success, end)
	}

	if !success {
		return ret, errors.New("request failed: " + ret.Message)
	}
	if response != nil {
		if err := json.Unmarshal(bodyResp, response); err != nil {
			return Response{}, err
		}
	}
	if r.options.afterSuccessRequestFunc != nil {
		r.options.afterSuccessRequestFunc(body, response)
	}
	return ret, nil
}

// Get sends a GET request to the specified URL
func (r *requestService) Get(ctx context.Context, uri string, response any) (Response, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.Header.SetMethod(http.MethodGet)
	req.Header.Set("accept", "application/json")
	req.SetRequestURI(r.baseURL + uri)

	err := fasthttp.Do(req, resp)
	if err != nil {
		return Response{}, err
	}

	bodyResp := resp.Body()
	ret := Response{
		StatusCode: resp.StatusCode(),
		Message:    string(bodyResp),
	}

	if !isSuccess(ret.StatusCode) {
		return ret, nil
	}
	if response != nil {
		if err = json.Unmarshal(bodyResp, response); err != nil {
			return Response{}, err
		}
	}

	return ret, nil
}

func isSuccess(statusCode int) bool {
	s := statusCode / 100
	return s == 2
}
