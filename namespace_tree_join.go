package godless

import "github.com/pkg/errors"

type NamespaceTreeJoin struct {
	noSelectVisitor
	noDebugVisitor
	errorCollectVisitor
	Namespace NamespaceTree
	tableKey  TableName
	table     Table
}

func MakeNamespaceTreeJoin(ns NamespaceTree) *NamespaceTreeJoin {
	return &NamespaceTreeJoin{Namespace: ns}
}

func (visitor *NamespaceTreeJoin) RunQuery() APIResponse {
	fail := RESPONSE_FAIL

	err := visitor.Error()
	if err != nil {
		fail.Err = visitor.Error()
		return fail
	}

	if visitor.tableKey == "" {
		panic("Expected table key")
	}

	err = visitor.Namespace.JoinTable(visitor.tableKey, visitor.table)

	if err != nil {
		fail.Err = errors.Wrap(err, "NamespaceTreeJoin failed")
		return fail
	}

	return RESPONSE_OK
}

func (visitor *NamespaceTreeJoin) VisitOpCode(opCode QueryOpCode) {
	if opCode != JOIN {
		visitor.collectError(errors.New("Expected JOIN OpCode"))
	}
}

func (visitor *NamespaceTreeJoin) VisitTableKey(tableKey TableName) {
	if visitor.Error() != nil {
		return
	}

	visitor.tableKey = tableKey
}

func (visitor *NamespaceTreeJoin) VisitJoin(*QueryJoin) {
}

func (visitor *NamespaceTreeJoin) VisitRowJoin(position int, rowJoin *QueryRowJoin) {
	if visitor.Error() != nil {
		return
	}

	row := Row{}

	for k, entryValue := range rowJoin.Entries {
		entry := MakeEntry([]Point{entryValue})
		row = row.JoinEntry(k, entry)
	}

	joined := visitor.table.JoinRow(rowJoin.RowKey, row)

	visitor.table = joined
}
