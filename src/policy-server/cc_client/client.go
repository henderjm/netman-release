package cc_client

import (
	"net/http"

	"code.cloudfoundry.org/lager"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	Host       string
	Name       string
	Secret     string
	HTTPClient httpClient
	Logger     lager.Logger
}

func (c *Client) GetAllAppGUIDs(token string) ([]string, error) {
	//resp, err := c.HTTPClient.Do(request)
	return []string{}, nil
}
