package godless

import (
	"bytes"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/pkg/errors"
)

// Generate a *valid* query.
func (query *Query) Generate(rand *rand.Rand, size int) reflect.Value {
	const TABLE_NAME_MAX = 20

	gen := &Query{}
	gen.TableKey = TableName(randLetters(rand, TABLE_NAME_MAX))

	if rand.Float32() > 0.5 {
		gen.OpCode = SELECT
		gen.Select = genQuerySelect(rand, size)
	} else {
		gen.OpCode = JOIN
		gen.Join = genQueryJoin(rand, size)
	}

	return reflect.ValueOf(gen)
}

func genQuerySelect(rand *rand.Rand, size int) QuerySelect {
	return QuerySelect{}
}

func genQueryJoin(rand *rand.Rand, size int) QueryJoin {
	const ROW_SCALE = 1.0
	const ENTRY_SCALE = 0.2
	const MAX_STR_LEN = 10
	rowCount := genCount(rand, size, ROW_SCALE)

	gen := QueryJoin{Rows: make([]QueryRowJoin, rowCount)}

	for i := 0; i < rowCount; i++ {
		gen.Rows[i] = QueryRowJoin{Entries: map[EntryName]Point{}}
		row := &gen.Rows[i]
		row.RowKey = RowName(randKey(rand, MAX_STR_LEN))

		entryCount := genCount(rand, size, ENTRY_SCALE)
		for i := 0; i < entryCount; i++ {
			entry := randKey(rand, MAX_STR_LEN)
			point := randPoint(rand, MAX_STR_LEN)
			row.Entries[EntryName(entry)] = Point(point)
		}
	}

	return gen
}

func randKey(rand *rand.Rand, max int) string {
	return randLetters(rand, max)
}

func randPoint(rand *rand.Rand, max int) string {
	const MIN_POINT_LENGTH = 0
	const pointSyms = ALPHABET + DIGITS + SYMBOLS
	const injectScale = 0.1
	injectCount := genCount(rand, max, injectScale)
	point := randStr(rand, pointSyms, MIN_POINT_LENGTH, max-injectCount)

	for i := 0; i < injectCount; i++ {
		// position := rand.Intn(len(point))
		// inject := randEscape(rand)
		// point = insert(point, inject, position)
	}
	return point
}

func insert(old, ins string, pos int) string {
	before := old[:pos]
	after := old[pos:]
	return before + ins + after
}

func randEscape(rand *rand.Rand) string {
	const chars = "\\nt\""
	return "\\" + randStr(rand, chars, 1, 2)
}

func TestParseQuery(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
		return
	}

	config := &quick.Config{
		MaxCount: PARSE_REPEAT_COUNT,
	}

	err := quick.Check(queryParseOk, config)

	if err != nil {
		t.Error("Unexpected error:", trim(err))
	}
}

func queryParseOk(expected *Query) bool {
	source := prettyQueryString(expected)
	logdbg("Pretty Printed: \"%v\"", source)

	actual, err := CompileQuery(source)

	if err != nil {
		panic(errors.Wrap(err, "Parse error"))
	}

	same := expected.Equals(actual)

	if !same {
		logDiff(source, prettyQueryString(actual))
	}

	return same
}

func logDiff(old, new string) {
	oldParts := strings.Split(old, "")
	newParts := strings.Split(new, "")

	minSize := imin(len(oldParts), len(newParts))

	for i := 0; i < minSize; i++ {
		oldChar := oldParts[i]
		newChar := newParts[i]

		if oldChar != newChar {
			fragmentStart := i - 10
			if fragmentStart < 0 {
				fragmentStart = 0
			}

			fragmentEnd := i + 100

			oldFragment := old[fragmentStart:fragmentEnd]
			newFragment := new[fragmentStart:fragmentEnd]

			logerr("First difference at %v", i)
			logerr("Old was: '%v'", oldFragment)
			logerr("New was: '%v'", newFragment)
		}
	}

}

func prettyQueryString(query *Query) string {
	buff := &bytes.Buffer{}
	err := query.PrettyPrint(buff)

	if err != nil {
		panic(err)
	}

	return buff.String()
}

func queryEncodeOk(expected *Query) bool {
	actual := querySerializationPass(expected)
	return expected.Equals(actual)
}

func querySerializationPass(expected *Query) *Query {
	buff := &bytes.Buffer{}
	err := EncodeQuery(expected, buff)

	if err != nil {
		panic(err)
	}

	var actual *Query
	actual, err = DecodeQuery(buff)

	if err != nil {
		panic(err)
	}

	return actual
}

const PARSE_REPEAT_COUNT = 50
