package api

//go:generate mockgen -package mock_godless -destination ../mock/mock_api.go -imports lib=github.com/johnny-morrice/api -self_package lib github.com/johnny-morrice/godless/api RemoteNamespace,RemoteStore,NamespaceTree,NamespaceSearcher,DataPeer,PubSubSubscription,PubSubRecord

import (
	"bytes"

	"github.com/pkg/errors"

	"github.com/johnny-morrice/godless/crdt"
	"github.com/johnny-morrice/godless/query"
)

type APIService interface {
	APICloserService
	APIRequestService
}

type APIRequestService interface {
	Call(APIRequest) (<-chan APIResponse, error)
}

type APICloserService interface {
	CloseAPI()
}

type APIRequest struct {
	Type       APIMessageType
	Reflection APIReflectionType
	Query      *query.Query
	Replicate  []crdt.Link
}

type APIResponder interface {
	RunQuery() APIResponse
}

type APIResponderFunc func() APIResponse

func (arf APIResponderFunc) RunQuery() APIResponse {
	return arf()
}

type APIReflectionType uint16

const (
	REFLECT_NOOP = APIReflectionType(iota)
	REFLECT_HEAD_PATH
	REFLECT_DUMP_NAMESPACE
	REFLECT_INDEX
)

type APIMessageType uint8

const (
	API_MESSAGE_NOOP = APIMessageType(iota)
	API_QUERY
	API_REFLECT
	API_REPLICATE
)

type RemoteNamespace interface {
	RunKvQuery(*query.Query, KvQuery)
	RunKvReflection(APIReflectionType, KvQuery)
	Replicate([]crdt.Link, KvQuery)
	Close()
}

type kvRunner interface {
	Run(RemoteNamespace, KvQuery)
}

type kvReplicator struct {
	links []crdt.Link
}

func (replicator kvReplicator) Run(kvn RemoteNamespace, kvq KvQuery) {
	kvn.Replicate(replicator.links, kvq)
}

type kvQueryRunner struct {
	query *query.Query
}

func (kqr kvQueryRunner) Run(kvn RemoteNamespace, kvq KvQuery) {
	kvn.RunKvQuery(kqr.query, kvq)
}

type kvReflectRunner struct {
	reflection APIReflectionType
}

func (krr kvReflectRunner) Run(kvn RemoteNamespace, kvq KvQuery) {
	kvn.RunKvReflection(krr.reflection, kvq)
}

type KvQuery struct {
	runner   kvRunner
	Request  APIRequest
	Response chan APIResponse
}

func makeApiQuery(request APIRequest, runner kvRunner) KvQuery {
	return KvQuery{
		Request:  request,
		runner:   runner,
		Response: make(chan APIResponse),
	}
}

func MakeKvQuery(request APIRequest) KvQuery {
	return makeApiQuery(request, kvQueryRunner{query: request.Query})
}

func MakeKvReflect(request APIRequest) KvQuery {
	return makeApiQuery(request, kvReflectRunner{reflection: request.Reflection})
}

func MakeKvReplicate(request APIRequest) KvQuery {
	return makeApiQuery(request, kvReplicator{links: request.Replicate})
}

func (kvq KvQuery) WriteResponse(val APIResponse) {
	kvq.Response <- val
	close(kvq.Response)
}

func (kvq KvQuery) Error(err error) {
	kvq.WriteResponse(APIResponse{Err: err})
}

func (kvq KvQuery) Run(kvn RemoteNamespace) {
	kvq.runner.Run(kvn, kvq)
}

type APIResponse struct {
	Msg       string
	Err       error
	Type      APIMessageType
	Path      crdt.IPFSPath
	Namespace crdt.Namespace
	Index     crdt.Index
}

func (resp APIResponse) IsEmpty() bool {
	return resp.Equals(APIResponse{})
}

func (resp APIResponse) AsText() (string, error) {
	const failMsg = "AsText failed"

	w := &bytes.Buffer{}
	err := EncodeAPIResponseText(resp, w)

	if err != nil {
		return "", errors.Wrap(err, failMsg)
	}

	return w.String(), nil
}

func (resp APIResponse) Equals(other APIResponse) bool {
	ok := resp.Msg == other.Msg
	ok = ok && resp.Type == other.Type
	ok = ok && resp.Path == other.Path

	if !ok {
		return false
	}

	if resp.Err != nil {
		if other.Err == nil {
			return false
		} else if resp.Err.Error() != other.Err.Error() {
			return false
		}
	}

	if !resp.Namespace.Equals(other.Namespace) {
		return false
	}

	if !resp.Index.Equals(other.Index) {
		return false
	}

	return true
}

var RESPONSE_FAIL_MSG = "error"
var RESPONSE_OK_MSG = "ok"
var RESPONSE_OK APIResponse = APIResponse{Msg: RESPONSE_OK_MSG}
var RESPONSE_FAIL APIResponse = APIResponse{Msg: RESPONSE_FAIL_MSG}
var RESPONSE_QUERY APIResponse = APIResponse{Msg: RESPONSE_OK_MSG, Type: API_QUERY}
var RESPONSE_REPLICATE APIResponse = APIResponse{Msg: RESPONSE_OK_MSG, Type: API_REPLICATE}
var RESPONSE_REFLECT APIResponse = APIResponse{Msg: RESPONSE_OK_MSG, Type: API_REFLECT}
