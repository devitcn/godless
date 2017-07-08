package api

//go:generate mockgen -package mock_godless -destination ../mock/mock_api.go -imports lib=github.com/johnny-morrice/api -self_package lib github.com/johnny-morrice/godless/api Core,RemoteStore,RemoteNamespace,NamespaceSearcher,DataPeer,PubSubSubscription,PubSubRecord

import (
	"bytes"

	"github.com/pkg/errors"

	"github.com/johnny-morrice/godless/crdt"
	"github.com/johnny-morrice/godless/query"
)

type Core interface {
	RunQuery(*query.Query, Command)
	Reflect(ReflectionType, Command)
	Replicate([]crdt.Link, Command)
	WriteMemoryImage() error
	Close()
}

type Service interface {
	CloserService
	RequestService
}

type RequestService interface {
	Call(Request) (<-chan Response, error)
}

type CloserService interface {
	CloseAPI()
}

type Request struct {
	Type       MessageType
	Reflection ReflectionType
	Query      *query.Query
	Replicate  []crdt.Link
}

type Responder interface {
	RunQuery() Response
}

type ResponderLambda func() Response

func (arf ResponderLambda) RunQuery() Response {
	return arf()
}

type ReflectionType uint16

const (
	REFLECT_NOOP = ReflectionType(iota)
	REFLECT_HEAD_PATH
	REFLECT_DUMP_NAMESPACE
	REFLECT_INDEX
)

type MessageType uint8

const (
	API_MESSAGE_NOOP = MessageType(iota)
	API_QUERY
	API_REFLECT
	API_REPLICATE
)

type coreCommand interface {
	Run(Core, Command)
}

type coreReplicator struct {
	links []crdt.Link
}

func (replicator coreReplicator) Run(kvn Core, kvq Command) {
	kvn.Replicate(replicator.links, kvq)
}

type coreQueryRunner struct {
	query *query.Query
}

func (kqr coreQueryRunner) Run(kvn Core, kvq Command) {
	kvn.RunQuery(kqr.query, kvq)
}

type coreReflectRunner struct {
	reflection ReflectionType
}

func (krr coreReflectRunner) Run(kvn Core, kvq Command) {
	kvn.Reflect(krr.reflection, kvq)
}

type Command struct {
	runner   coreCommand
	Request  Request
	Response chan Response
}

func makeApiQuery(request Request, runner coreCommand) Command {
	return Command{
		Request:  request,
		runner:   runner,
		Response: make(chan Response),
	}
}

func MakeQueryCommand(request Request) Command {
	return makeApiQuery(request, coreQueryRunner{query: request.Query})
}

func MakeReflectCommand(request Request) Command {
	return makeApiQuery(request, coreReflectRunner{reflection: request.Reflection})
}

func MakeReplicateCommand(request Request) Command {
	return makeApiQuery(request, coreReplicator{links: request.Replicate})
}

func (kvq Command) WriteResponse(val Response) {
	kvq.Response <- val
	close(kvq.Response)
}

func (kvq Command) Error(err error) {
	kvq.WriteResponse(Response{Err: err})
}

func (kvq Command) Run(kvn Core) {
	kvq.runner.Run(kvn, kvq)
}

type Response struct {
	Msg       string
	Err       error
	Type      MessageType
	Path      crdt.IPFSPath
	Namespace crdt.Namespace
	Index     crdt.Index
}

func (resp Response) IsEmpty() bool {
	return resp.Equals(Response{})
}

func (resp Response) AsText() (string, error) {
	const failMsg = "AsText failed"

	w := &bytes.Buffer{}
	err := EncodeAPIResponseText(resp, w)

	if err != nil {
		return "", errors.Wrap(err, failMsg)
	}

	return w.String(), nil
}

func (resp Response) Equals(other Response) bool {
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
var RESPONSE_OK Response = Response{Msg: RESPONSE_OK_MSG}
var RESPONSE_FAIL Response = Response{Msg: RESPONSE_FAIL_MSG}
var RESPONSE_QUERY Response = Response{Msg: RESPONSE_OK_MSG, Type: API_QUERY}
var RESPONSE_REPLICATE Response = Response{Msg: RESPONSE_OK_MSG, Type: API_REPLICATE}
var RESPONSE_REFLECT Response = Response{Msg: RESPONSE_OK_MSG, Type: API_REFLECT}
