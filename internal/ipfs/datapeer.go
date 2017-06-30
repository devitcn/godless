package ipfs

import (
	"io"
	gohttp "net/http"
	"time"

	ipfs "github.com/ipfs/go-ipfs-api"

	"github.com/johnny-morrice/godless/api"
	"github.com/johnny-morrice/godless/internal/http"
	"github.com/johnny-morrice/godless/log"
)

type WebServiceClient struct {
	Url         string
	Http        *gohttp.Client
	PingTimeout time.Duration
	Shell       *ipfs.Shell
	pinger      *ipfs.Shell
}

func (client *WebServiceClient) Connect() error {
	if client.PingTimeout == 0 {
		client.PingTimeout = __DEFAULT_PING_TIMEOUT
	}

	if client.Http == nil {
		log.Info("Using default HTTP client")
		client.Http = http.DefaultBackendClient()
	}

	log.Info("Connecting to IPFS API...")
	pingClient := http.DefaultBackendClient()
	pingClient.Timeout = client.PingTimeout
	client.Shell = ipfs.NewShellWithClient(client.Url, client.Http)
	client.pinger = ipfs.NewShellWithClient(client.Url, pingClient)

	return nil
}

func (client *WebServiceClient) IsUp() bool {
	return client.pinger.IsUp()
}

func (client *WebServiceClient) Disconnect() error {
	return nil
}

func (client WebServiceClient) Cat(path string) (io.ReadCloser, error) {
	return client.Shell.Cat(path)
}

func (client WebServiceClient) Add(r io.Reader) (string, error) {
	return client.Shell.Add(r)
}

func (client WebServiceClient) PubSubPublish(topic, data string) error {
	return client.Shell.PubSubPublish(topic, data)
}

type subscription struct {
	sub *ipfs.PubSubSubscription
}

func (sub subscription) Next() (api.PubSubRecord, error) {
	rec, err := sub.sub.Next()

	if err != nil {
		return nil, err
	}

	return record{rec: rec}, nil
}

type record struct {
	rec ipfs.PubSubRecord
}

func (rec record) From() string {
	return string(rec.rec.From())
}

func (rec record) Data() []byte {
	return rec.rec.Data()
}

func (rec record) SeqNo() int64 {
	return rec.rec.SeqNo()
}

func (rec record) TopicIDs() []string {
	return rec.rec.TopicIDs()
}

func (client WebServiceClient) PubSubSubscribe(topic string) (api.PubSubSubscription, error) {
	sub, err := client.Shell.PubSubSubscribe(topic)

	if err != nil {
		return nil, err
	}

	return subscription{sub: sub}, nil
}

const __DEFAULT_PING_TIMEOUT = time.Second * 5
