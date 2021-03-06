package cache

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/johnny-morrice/godless/internal/testutil"
)

func TestBoltCache(t *testing.T) {
	f := createTempFile()
	defer f.Close()

	options := BoltOptions{
		FilePath: f.Name(),
	}

	boltFactory, err := MakeBoltFactory(options)

	panicOnBadInit(err)

	cache, err := boltFactory.MakeCache()

	panicOnBadInit(err)

	testCacheGetSet(t, cache)
}

func TestBoltCacheConcurrency(t *testing.T) {
	f := createTempFile()
	defer f.Close()

	options := BoltOptions{
		FilePath: f.Name(),
	}

	boltFactory, err := MakeBoltFactory(options)

	panicOnBadInit(err)

	cache, err := boltFactory.MakeCache()

	panicOnBadInit(err)

	count := __CONCURRENCY_LEVEL / 16
	wg := &sync.WaitGroup{}

	wg.Add(3)

	go func() {
		testHeadConcurrency(t, cache, count)
		wg.Done()
	}()

	go func() {
		testIndexConcurrency(t, cache, count)
		wg.Done()
	}()

	go func() {
		testNamespaceConcurrency(t, cache, count)
		wg.Done()
	}()

	const timeout = time.Second * 30
	testutil.WaitGroupTimeout(t, wg, timeout)
}

func TestBoltCacheExpire(t *testing.T) {
	f := createTempFile()
	defer f.Close()

	const buffsize = 10
	const count = 2 * buffsize

	options := BoltOptions{
		FilePath:     f.Name(),
		MaxCacheSize: buffsize,
	}

	boltFactory, err := MakeBoltFactory(options)

	panicOnBadInit(err)

	cache, err := boltFactory.MakeCache()

	panicOnBadInit(err)

	testNamespaceExpire(t, cache, count, buffsize)
	testIndexExpire(t, cache, count, buffsize)
}

func createTempFile() *os.File {
	file, err := ioutil.TempFile("/tmp", "godless_bolt_test")

	panicOnBadInit(err)

	return file
}

func panicOnBadInit(err error) {
	if err != nil {
		panic(err)
	}
}

func TestBoltMemoryImage(t *testing.T) {
	f := createTempFile()
	defer f.Close()

	options := BoltOptions{
		FilePath: f.Name(),
	}

	boltFactory, err := MakeBoltFactory(options)

	panicOnBadInit(err)

	memimg, err := boltFactory.MakeMemoryImage()

	panicOnBadInit(err)

	testMemoryImage(t, memimg)
	const count = __CONCURRENCY_LEVEL / 16
	testMemoryImageConcurrency(t, memimg, count)
}
