package mock_godless

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	"github.com/johnny-morrice/godless/api"
	"github.com/johnny-morrice/godless/crdt"
	"github.com/johnny-morrice/godless/internal/ipfs"
	"github.com/johnny-morrice/godless/internal/testutil"
)

func TestIpfsRemoteStoreConnectSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	mock.EXPECT().Connect().Return(nil)

	err := store.Connect()

	testutil.AssertNil(t, err)
}

func TestIpfsRemoteStoreConnectFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	mock.EXPECT().Connect().Return(expectedError())

	err := store.Connect()

	testutil.AssertNonNil(t, err)
}

func TestIpfsRemoteStoreAddNamespaceSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const nsAddrText string = "NS Addr"

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	ns := makeNamespaceForIPFS()

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Add(gomock.Any()).Return(nsAddrText, nil)

	addr, err := store.AddNamespace(ns)

	testutil.AssertNil(t, err)

	testutil.AssertEquals(t, "Unexpected namespace path", nsAddrText, string(addr))
}

func TestIpfsRemoteStoreAddNamespaceFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	ns := makeNamespaceForIPFS()

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Add(gomock.Any()).Return("", expectedError())

	addr, err := store.AddNamespace(ns)

	testutil.AssertNonNil(t, err)
	testutil.Assert(t, "Expected nil path", crdt.IsNilPath(addr))
}

func TestIpfsRemoteStoreAddIndexSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const indexAddrText string = "Index Addr"

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	index := makeIndexForIPFS()

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Add(gomock.Any()).Return(indexAddrText, nil)

	addr, err := store.AddIndex(index)

	testutil.AssertNil(t, err)

	testutil.AssertEquals(t, "Unexpected index path", indexAddrText, string(addr))
}

func TestIpfsRemoteStoreAddIndexFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const indexAddrText string = "Index Addr"

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	index := makeIndexForIPFS()

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Add(gomock.Any()).Return("", expectedError())

	addr, err := store.AddIndex(index)

	testutil.AssertNonNil(t, err)
	testutil.Assert(t, "Expected nil path", crdt.IsNilPath(addr))
}

func TestIpfsRemoteStoreCatNamespaceSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const namespaceAddr = "Namespace addr"

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	expected := makeNamespaceForIPFS()
	reader := makeNamespaceReaderForIPFS()

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Cat(namespaceAddr).Return(reader, nil)

	actual, err := store.CatNamespace(namespaceAddr)

	testutil.AssertNil(t, err)
	testutil.Assert(t, "Unexpected namespace", expected.Equals(actual))
}

func TestIpfsRemoteStoreCatNamespaceFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const namespaceAddr = "Namespace addr"

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Cat(namespaceAddr).Return(nil, expectedError())

	namespace, err := store.CatNamespace(namespaceAddr)

	testutil.AssertNonNil(t, err)
	testutil.Assert(t, "Expected zero namespace", namespace.IsEmpty())
}

func TestIpfsRemoteStoreCatIndexSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const indexAddr = "Index addr"

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	expected := makeIndexForIPFS()
	reader := makeIndexReaderForIPFS()

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Cat(indexAddr).Return(reader, nil)

	actual, err := store.CatIndex(indexAddr)

	testutil.AssertNil(t, err)
	testutil.Assert(t, "Unexpected index", expected.Equals(actual))
}

func TestIpfsRemoteStoreCatIndexFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const indexAddr = "Index addr"

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	mock.EXPECT().IsUp().Return(true).AnyTimes()
	mock.EXPECT().Cat(indexAddr).Return(nil, expectedError())

	index, err := store.CatIndex(indexAddr)

	testutil.AssertNonNil(t, err)
	testutil.Assert(t, "Expected zero index", index.IsEmpty())
}

func TestIpfsRemoteStoreSubscribeAddrStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	mockSub := NewMockPubSubSubscription(ctrl)
	mockRecord := NewMockPubSubRecord(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)
	defer store.Disconnect()

	const topic = "Hello"
	const readCount = 10

	expectedLink := crdt.UnsignedLink("Dude")
	linkText, err := crdt.SerializeLink(expectedLink)
	panicOnBadInit(err)
	linkBytes := []byte(linkText)

	mock.EXPECT().IsUp().Return(true)
	mock.EXPECT().Disconnect().Return(nil)
	mock.EXPECT().PubSubSubscribe(topic).Return(mockSub, nil)
	mockSub.EXPECT().Next().Return(mockRecord, nil).MinTimes(readCount)
	mockRecord.EXPECT().Data().Return(linkBytes).MinTimes(readCount)
	mockRecord.EXPECT().From().MinTimes(readCount)

	linkch, errch := store.SubscribeAddrStream(topic)

	testutil.AssertNonNil(t, linkch)
	testutil.AssertNonNil(t, errch)

	for i := 0; i < readCount; i++ {
		actualLink := <-linkch
		testutil.AssertEquals(t, "Unexpected Link", expectedLink.Path(), actualLink.Path())
	}

}

func TestIpfsRemoteStoreSubscribeAddrStreamRestart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	mockSub := NewMockPubSubSubscription(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)
	defer store.Disconnect()

	const topic = "Hello"
	const timeoutDuration = time.Millisecond * 100

	mock.EXPECT().Disconnect().Return(nil)
	mock.EXPECT().IsUp().Return(true)
	mock.EXPECT().PubSubSubscribe(topic).Return(mockSub, nil).AnyTimes()
	mockSub.EXPECT().Next().Return(nil, expectedError()).AnyTimes()

	linkch, errch := store.SubscribeAddrStream(topic)

	testutil.AssertNonNil(t, linkch)
	testutil.AssertNonNil(t, errch)

	timeout := time.NewTimer(timeoutDuration)

	select {
	case link := <-linkch:
		t.Error("Unexpected link:", link)
	case <-timeout.C:
		break
	}
}

func TestIpfsRemoteStoreSubscribeAddrStreamFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)
	defer store.Disconnect()

	const topic = "Hello"
	const timeoutDuration = time.Millisecond * 100

	mock.EXPECT().Disconnect().Return(nil)
	mock.EXPECT().IsUp().Return(true)
	mock.EXPECT().PubSubSubscribe(topic).Return(nil, expectedError()).AnyTimes()

	linkch, errch := store.SubscribeAddrStream(topic)

	testutil.AssertNonNil(t, linkch)
	testutil.AssertNonNil(t, errch)

	timeout := time.NewTimer(timeoutDuration)

	select {
	case link := <-linkch:
		t.Error("Unexpected link:", link)
	case <-timeout.C:
		break
	}
}

func TestIpfsRemoteStorePublishAddrSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	link := crdt.UnsignedLink("hi")
	linkText, linkErr := crdt.SerializeLink(link)

	panicOnBadInit(linkErr)

	topics := []api.PubSubTopic{"wow", "awesome"}

	for _, t := range topics {
		mock.EXPECT().PubSubPublish(string(t), string(linkText)).Return(nil)
	}

	mock.EXPECT().IsUp().Return(true)

	err := store.PublishAddr(link, topics)

	testutil.AssertNil(t, err)
}

func TestIpfsRemoteStorePublishAddrFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	link := crdt.UnsignedLink("hi")
	linkText, linkErr := crdt.SerializeLink(link)

	panicOnBadInit(linkErr)

	topics := []api.PubSubTopic{"wow", "awesome"}

	for _, t := range topics {
		mock.EXPECT().PubSubPublish(string(t), string(linkText)).Return(expectedError())
	}

	mock.EXPECT().IsUp().Return(true)

	err := store.PublishAddr(link, topics)

	testutil.AssertNil(t, err)
}

func TestIpfsRemoteStoreDisconnectSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	mock.EXPECT().Disconnect().Return(nil)

	err := store.Disconnect()

	testutil.AssertNil(t, err)
}

func TestIpfsRemoteStoreDisconnectFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDataPeer(ctrl)
	store := ipfs.MakeIpfsRemoteStore(mock)

	mock.EXPECT().Disconnect().Return(expectedError())

	err := store.Disconnect()
	testutil.AssertNonNil(t, err)
}

func makeNamespaceReaderForIPFS() io.ReadCloser {
	ns := makeNamespaceForIPFS()
	buff := &bytes.Buffer{}
	invalid, err := crdt.EncodeNamespace(ns, buff)

	panicOnInvalidNamespace(invalid)
	panicOnBadInit(err)

	return ioutil.NopCloser(buff)
}

func makeIndexReaderForIPFS() io.ReadCloser {
	index := makeIndexForIPFS()
	buff := &bytes.Buffer{}
	invalid, err := crdt.EncodeIndex(index, buff)

	panicOnInvalidIndex(invalid)
	panicOnBadInit(err)
	return ioutil.NopCloser(buff)
}

func makeIndexForIPFS() crdt.Index {
	return crdt.MakeIndex(map[crdt.TableName]crdt.Link{
		"Hi": crdt.UnsignedLink("Dude"),
	})
}

func makeNamespaceForIPFS() crdt.Namespace {
	return crdt.MakeNamespace(map[crdt.TableName]crdt.Table{
		"Hi": crdt.MakeTable(map[crdt.RowName]crdt.Row{
			"Hello": crdt.MakeRow(map[crdt.EntryName]crdt.Entry{
				"Dude": crdt.MakeEntry([]crdt.Point{crdt.UnsignedPoint("Wow")}),
			}),
		}),
	})
}

func expectedError() error {
	return errors.New("Expected error")
}
