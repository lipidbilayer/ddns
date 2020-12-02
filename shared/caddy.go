package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Caddy struct {
	caddyAdminHost string
	domain         string
	client         *http.Client
}

type Response struct {
	*http.Response
	Body []byte
}

func (r *Response) handleHTTPError() error {
	if r.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("Bad status %s", r.Status)
	}
	return nil
}

type CaddyHostRoute struct {
	ID     string   `json:"@id,omitempty"`
	Handle []Handle `json:"handle,omitempty"`
	Match  []Match  `json:"match,omitempty"`
}

type Handle struct {
	Handler   string      `json:"handler,omitempty"`
	Upstreams []Upstreams `json:"upstreams,omitempty"`
}

type Upstreams struct {
	Dial string `json:"dial,omitempty"`
}

type Match struct {
	Host []string `json:"host,omitempty"`
}

func NewCaddy(config *Config) *Caddy {
	return &Caddy{
		caddyAdminHost: config.CaddyAdminHost,
		domain:         config.Domain,
		client:         &http.Client{},
	}
}

func (c *Caddy) callAPI(url, httpMethod string, request interface{}) (*Response, error) {
	body, err := jsonMarshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = cerr
		}
	}()

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	r := &Response{
		Response: resp,
		Body:     buf.Bytes(),
	}

	if err = r.handleHTTPError(); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Caddy) SetReverseProxy(host *Host) error {

	handle := Handle{
		Handler:   "reverse_proxy",
		Upstreams: c.setUpstream(host),
	}

	match := Match{
		Host: []string{fmt.Sprintf("%s%s", host.Hostname, c.domain)},
	}
	request := &CaddyHostRoute{
		ID:     host.Hostname,
		Handle: []Handle{handle},
		Match:  []Match{match},
	}

	url := fmt.Sprintf("http://%s/config/apps/http/servers/srv0/routes", c.caddyAdminHost)
	requestMethod := http.MethodPost
	_, err := c.callAPI(url, requestMethod, request)
	if err != nil {
		return err
	}
	return nil
}

func (c *Caddy) UpdateReverseProxy(host *Host) error {
	request := c.setUpstream(host)
	url := fmt.Sprintf("http://%s/id/%s/handle/0/upstreams/", c.caddyAdminHost, host.Hostname)
	requestMethod := http.MethodPatch
	_, err := c.callAPI(url, requestMethod, request)
	if err != nil {
		return err
	}
	return nil
}

func (c *Caddy) setUpstream(host *Host) []Upstreams {
	dial := fmt.Sprintf("%s:%s", host.Ip, host.Port)
	upstreams := Upstreams{Dial: dial}
	return []Upstreams{upstreams}
}

func jsonMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}
