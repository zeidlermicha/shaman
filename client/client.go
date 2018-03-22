package client

import (
	"encoding/json"
	"net/http"

	sham "github.com/nanopack/shaman/core/common"

	"bytes"
	"io"
	"errors"
	"net/url"
	"reflect"
	"github.com/google/go-querystring/query"
	"time"
)

const (
	records = "/records"
)


type ShamanClient struct {
	httpClient *http.Client
	host       string
	token      string
}

type FullOption struct {
	ShowFull bool `url:"full,omitempty"`
}

func NewShamanClient(host string, token string) *ShamanClient {
	return &ShamanClient{
		httpClient: &http.Client{Timeout:time.Second * 10},
		host:       host,
		token:      token,
	}
}

func (c *ShamanClient) GetRecords(options *FullOption) ([]*sham.Resource, error) {
	res := make([]*sham.Resource, 0)
	path, e := addURLQueryOptions(records, options)
	if e != nil {
		return nil, e
	}
	_, err := c.get(path, &res)
	if err != nil {
		return nil, err
	}

	return res, err
}

func (c *ShamanClient) AddRecord(resource *sham.Resource) (*sham.Resource, error) {
	res := &sham.Resource{}
	_, err := c.post(records, resource, res)
	if nil != err {
		return nil, err
	}
	return res, err
}

func (c *ShamanClient) UpdateRecord(resource *sham.Resource) (*sham.Resource, error) {
	res := &sham.Resource{}
	_, err := c.put(records+"/"+resource.Domain, resource, res)
	if nil != err {
		return nil, err
	}
	return res, err
}

func (c *ShamanClient) DeleteRecord(domain string) error {
	msg := &sham.ApiError{}
	_, err := c.delete(records+"/"+domain, nil, msg)

	return err
}

func (c *ShamanClient) NewRequest(method, path string, payload interface{}) (*http.Request, error) {
	urlPath := c.host + path

	body := new(bytes.Buffer)
	if payload != nil {
		err := json.NewEncoder(body).Encode(payload)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, urlPath, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-AUTH-TOKEN", c.token)

	return req, nil
}

func (c *ShamanClient) get(path string, obj interface{}) (*http.Response, error) {
	req, err := c.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req, obj)
}

func (c *ShamanClient) post(path string, payload, obj interface{}) (*http.Response, error) {
	req, err := c.NewRequest("POST", path, payload)
	if err != nil {
		return nil, err
	}

	return c.Do(req, obj)
}

func (c *ShamanClient) put(path string, payload, obj interface{}) (*http.Response, error) {
	req, err := c.NewRequest("PUT", path, payload)
	if err != nil {
		return nil, err
	}

	return c.Do(req, obj)
}

func (c *ShamanClient) delete(path string, payload interface{}, obj interface{}) (*http.Response, error) {
	req, err := c.NewRequest("DELETE", path, payload)
	if err != nil {
		return nil, err
	}

	return c.Do(req, obj)
}

func (c *ShamanClient) Do(req *http.Request, obj interface{}) (*http.Response, error) {

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	e, err := checkResponse(resp)
	if err != nil {
		return resp, err
	}
	if e != nil {
		return resp, errors.New(e.ErrorString)
	}

	if obj != nil {
		if w, ok := obj.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(obj)
		}
	}

	return resp, err
}

func checkResponse(resp *http.Response) (*sham.ApiError, error) {
	if code := resp.StatusCode; 200 <= code && code <= 299 {
		return nil, nil
	}

	errorResponse := &sham.ApiError{}

	err := json.NewDecoder(resp.Body).Decode(errorResponse)
	if err != nil {
		return nil, err
	}

	return errorResponse, nil

}


func addURLQueryOptions(path string, options interface{}) (string, error) {
	opt := reflect.ValueOf(options)


	if opt.Kind() == reflect.Ptr && opt.IsNil() {
		return path, nil
	}

	u, err := url.Parse(path)
	if err != nil {
		return path, err
	}

	qs, err := query.Values(options)
	if err != nil {
		return path, err
	}

	uqs := u.Query()
	for k := range qs {
		uqs.Set(k, qs.Get(k))
	}
	u.RawQuery = uqs.Encode()

	return u.String(), nil
}
