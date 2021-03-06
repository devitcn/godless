package http

import (
	"bytes"
	"fmt"
	"io"
	gohttp "net/http"
	"time"

	"github.com/johnny-morrice/godless/api"
	"github.com/johnny-morrice/godless/log"
	"github.com/pkg/errors"
)

type ClientOptions struct {
	Endpoints
	ServerAddr string
	Http       *gohttp.Client
	Validator  api.RequestValidator
}

type client struct {
	ClientOptions
}

func MakeClient(options ClientOptions) (api.Client, error) {
	client := &client{ClientOptions: options}

	if client.Validator == nil {
		client.Validator = api.StandardRequestValidator()
	}

	if client.ServerAddr == "" {
		return nil, errors.New("Expected ServerAddr")
	}

	client.UseDefaultEndpoints()

	if client.Http == nil {
		client.Http = defaultHttpClient()
	}

	return client, nil
}

func (client *client) Send(request api.Request) (api.Response, error) {
	err := request.Validate(client.Validator)

	if err != nil {
		log.Debug("Query validation error: %v", err)
		return api.RESPONSE_FAIL, errors.Wrap(err, fmt.Sprintf("Cowardly refusing to send invalid Request: %v", request))
	}

	buff := &bytes.Buffer{}
	err = api.EncodeRequest(request, buff)

	if err != nil {
		return api.RESPONSE_FAIL, errors.Wrap(err, "SendQuery failed")
	}

	return client.Post(client.CommandEndpoint, MIME_PROTO, buff)
}

func (client *client) Post(path, bodyType string, body io.Reader) (api.Response, error) {
	addr := client.ServerAddr + path
	log.Info("HTTP POST to %s", addr)

	resp, err := client.Http.Post(addr, bodyType, body)

	if err != nil {
		return api.RESPONSE_FAIL, errors.Wrap(err, "HTTP POST failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != WEB_API_SUCCESS && resp.StatusCode != WEB_API_ERROR {
		return api.RESPONSE_FAIL, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	apiresp, err := client.decodeHttpResponse(resp)

	if err != nil {
		return apiresp, errors.Wrap(err, "Error decoding API response")
	}

	if apiresp.Err == nil {
		return apiresp, nil
	} else {
		return apiresp, errors.Wrap(apiresp.Err, "API returned error")
	}
}

func (client *client) decodeHttpResponse(resp *gohttp.Response) (api.Response, error) {
	if HasContentType(resp.Header, MIME_PROTO) {
		return api.DecodeResponse(resp.Body)
	} else if HasContentType(resp.Header, MIME_PROTO_TEXT) {
		return api.DecodeResponseText(resp.Body)
	} else {
		return api.RESPONSE_FAIL, incorrectContentType(resp.Header)
	}
}

var __frontendClient *gohttp.Client

func defaultHttpClient() *gohttp.Client {
	if __frontendClient == nil {
		__frontendClient = &gohttp.Client{
			Timeout: time.Duration(__FRONTEND_TIMEOUT),
		}
	}

	return __frontendClient
}

const __FRONTEND_TIMEOUT = 1 * time.Minute
