package prom

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type gardenerManagedPrometheusClient struct {
	client *http.Client
	uri    string
}

func (mpc *gardenerManagedPrometheusClient) Query(query string, queryTime time.Time) ([]byte, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("time", queryTime.Format(time.RFC3339))
	res, err := mpc.client.Get(mpc.uri + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if statusCode := res.StatusCode; statusCode > 299 {
		return resBody, fmt.Errorf("response failed with status code %d", statusCode)
	}
	if err != nil {
		return nil, err
	}
	return resBody, nil
}

// BasicAuthTransport is a custom transport that adds Basic Authentication to the HTTP request
type BasicAuthTransport struct {
	Username string
	Password string
	Wrapped  http.RoundTripper
}

func (bat *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(bat.Username, bat.Password)
	return bat.Wrapped.RoundTrip(req)
}

// NewGardenerManagedPrometheusClient returns an HTTP client for talking to
// the Gardener Managed Service for Prometheus.
func NewGardenerManagedPrometheusClient() (Client, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	// Set Basic Authentication
	client.Transport = &BasicAuthTransport{
		Username: "admin",
		Password: "jabWZf5AxsjbHyP5y1IzZC4tfwBDUmaG",
		Wrapped:  http.DefaultTransport,
	}
	return &gardenerManagedPrometheusClient{
		client: client,
		uri:    "https://p-i030268--perf-test.ingress.aws-ha.seed.dev.k8s.ondemand.com/api/v1/query",
	}, nil
}

var _ Client = &gardenerManagedPrometheusClient{}
