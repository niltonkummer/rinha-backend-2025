package request

import (
	"context"
	"errors"
	"time"

	"github.com/goccy/go-json"
	"resty.dev/v3"
)

type Response struct {
	StatusCode int    `json:"statusCode"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// RequestService defines the interface for making HTTP requests
type RequestService interface {
	Post(ctx context.Context, url string, body any, response any) (*Response, error)
	Get(ctx context.Context, url string, response any) (*Response, error)
	SetTimeout(timeout time.Duration)
}

// requestService implements the RequestService interface
type requestService struct {
	client *resty.Client
	*options
}

type options struct {
	afterSuccessRequestFunc afterSuccessRequest
}

type afterSuccessRequest func(any, any)

// WithAfterRequestFunc sets a function to be called after each request
func WithAfterRequestFunc(fn afterSuccessRequest) func(*options) {
	return func(opts *options) {
		opts.afterSuccessRequestFunc = fn
	}
}

// NewRequestService creates a new instance of requestService
func NewRequestService(baseUrl string, opts ...func(*options)) RequestService {
	client := resty.NewWithTransportSettings(
		// Set the transport settings to use a custom HTTP client
		&resty.TransportSettings{
			DialerKeepAlive: time.Minute * 3,
		}).
		SetRetryCount(1).
		SetTimeout(time.Second * 5).
		SetBaseURL(baseUrl)

	rs := &requestService{
		client:  client,
		options: &options{},
	}

	for _, opt := range opts {
		opt(rs.options)
	}
	return rs
}

func (r *requestService) SetTimeout(timeout time.Duration) {
	r.client.SetTimeout(timeout)
}

// Post sends a POST request to the specified URL with the given body
func (r *requestService) Post(ctx context.Context, url string, body any, response any) (*Response, error) {
	resp, err := r.client.R().
		SetContentType("application/json").
		WithContext(ctx).
		SetBody(body).
		Post(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ret := &Response{
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}

	if !resp.IsSuccess() {
		return ret, errors.New("request failed: " + resp.Status())
	}
	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return nil, err
		}
	}
	if r.options.afterSuccessRequestFunc != nil {
		r.options.afterSuccessRequestFunc(body, response)
	}
	return ret, nil
}

// Get sends a GET request to the specified URL
func (r *requestService) Get(ctx context.Context, url string, response any) (*Response, error) {
	request := r.client.R()
	resp, err := request.
		WithContext(ctx).
		SetExpectResponseContentType("application/json").
		Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ret := &Response{
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}

	if !resp.IsSuccess() {
		return ret, nil
	}
	if response != nil {
		if err = json.NewDecoder(resp.Body).Decode(response); err != nil {
			return nil, err
		}
	}

	return ret, nil
}
