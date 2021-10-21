package dag

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

type dagBinaryType uint8

const (
	DAGBinaryTypeDAG    dagBinaryType = iota
	DAGBinaryTypeVertex dagBinaryType = iota
	DAGBinaryTypeEdge   dagBinaryType = iota
)

const (
	DAGDigestTypeLeaf   = iota
	DAGDigestTypeBranch = iota
)

type DAG struct {
	Nodes []DAGVertex
	Edges []DAGEdge
	sinks []uint32
}

func NewDAG() *DAG {
	return &DAG{}
}

func NewDAGFromBinary(b []byte) (*DAG, error) {
	l := len(b)
	lErr := errors.New("DAG: Invalid binary " + hex.EncodeToString(b) + " because of \"invalid length\"")
	d := DAG{}
	i := 0

	if l < i+1 {
		return nil, lErr
	}
	bType := b[i]
	i += 1

	if bType != byte(DAGBinaryTypeDAG) {
		return nil, errors.New("DAG: Invalid binary " + hex.EncodeToString(b) + " because of \"invalid type, expected " + strconv.FormatUint(uint64(DAGBinaryTypeDAG), 10) + " got " + strconv.FormatUint(uint64(bType), 10) + "\"")
	}

	if l < i+4 {
		return nil, lErr
	}
	nodesLength := binary.BigEndian.Uint32(b[i : i+4])
	i += 4

	for j := 0; j < int(nodesLength); j++ {
		if l < i+4 {
			return nil, lErr
		}
		nBinLength := binary.BigEndian.Uint32(b[i : i+4])
		i += 4

		if l < i+int(nBinLength) {
			return nil, lErr
		}
		nBin := b[i : i+int(nBinLength)]
		i += int(nBinLength)

		node, err := NewDAGVertexFromBinary(nBin)
		if err != nil {
			return nil, err
		}

		d.Add(*node)
	}

	if l < i+4 {
		return nil, lErr
	}
	edgesLength := binary.BigEndian.Uint32(b[i : i+4])
	i += 4

	for j := 0; j < int(edgesLength); j++ {
		if l < i+4 {
			return nil, lErr
		}
		eBinLength := binary.BigEndian.Uint32(b[i : i+4])
		i += 4

		if l < i+int(eBinLength) {
			return nil, lErr
		}
		eBin := b[i : i+int(eBinLength)]
		i += int(eBinLength)

		edge, err := NewDAGEdgeFromBinary(eBin)
		if err != nil {
			return nil, err
		}

		d.AddEdge(*edge)
	}

	return &d, nil
}

func (dag *DAG) Binary() []byte {
	var buf bytes.Buffer

	buf.Write([]byte{byte(DAGBinaryTypeDAG)})

	nodesLength := len(dag.Nodes)
	bNodesLength := make([]byte, 4)
	binary.BigEndian.PutUint32(bNodesLength, uint32(nodesLength))
	buf.Write(bNodesLength)

	for _, node := range dag.Nodes {
		nBin := node.Binary()

		nBinLen := make([]byte, 4)
		binary.BigEndian.PutUint32(nBinLen, uint32(len(nBin)))
		buf.Write(nBinLen)

		buf.Write(node.Binary())
	}

	edgesLength := len(dag.Edges)
	bEdgesLength := make([]byte, 4)
	binary.BigEndian.PutUint32(bEdgesLength, uint32(edgesLength))
	buf.Write(bEdgesLength)

	for _, edge := range dag.Edges {
		eBin := edge.Binary()

		eBinLen := make([]byte, 4)
		binary.BigEndian.PutUint32(eBinLen, uint32(len(eBin)))
		buf.Write(eBinLen)

		buf.Write(edge.Binary())
	}

	return buf.Bytes()
}

func (dag *DAG) Clone() *DAG {
	d, _ := NewDAGFromBinary(dag.Binary())

	return d
}

func (dag *DAG) Add(node DAGVertex) {
	for i := range dag.Nodes {
		if dag.Nodes[i].ID == node.ID {
			return
		}
	}

	dag.Nodes = append(dag.Nodes, node)
	dag.sinks = append(dag.sinks, node.ID)
}

func (dag *DAG) AddReduced(node DAGVertex) {
	for _, sink := range dag.sinks {
		dag.AddEdge(DAGEdge{
			Child:  node.ID,
			Parent: sink,
		})
	}

	dag.Add(node)
}

func (dag *DAG) AddEdge(edge DAGEdge) error {
	for _, e := range dag.Edges {
		if e.Child == edge.Child && e.Parent == edge.Parent {
			return nil
		}
	}

	dag.Edges = append(dag.Edges, edge)

	_, err := dag.Dependencies(edge.Child, nil)
	if err != nil {
		dag.Edges = dag.Edges[:len(dag.Edges)-1]

		return err
	}

	for i, sink := range dag.sinks {
		if sink == edge.Parent {
			dag.sinks = append(dag.sinks[:i], dag.sinks[i+1:]...)

			break
		}
	}

	return nil
}

func (dag *DAG) ParentEdges(id uint32) []DAGEdge {
	edges := []DAGEdge{}

	for _, e := range dag.Edges {
		if e.Child == id {
			edges = append(edges, e)
		}
	}

	return edges
}

func (dag *DAG) Dependencies(id uint32, seen []uint32) ([]uint32, error) {
	for i := range seen {
		if id == seen[i] {
			seenStr := make([]string, len(seen))
			for i, s := range seen {
				seenStr[i] = strconv.FormatUint(uint64(s), 10)
			}

			return nil, errors.New("DAG: Cyclic reference detected " + strconv.FormatUint(uint64(id), 10) + " in [" + strings.Join(seenStr, ", ") + "]")
		}
	}

	parentEdges := dag.ParentEdges(id)
	parents := make([]uint32, len(parentEdges))
	for i := range parentEdges {
		parents[i] = parentEdges[i].Parent
	}
	sort.Slice(parents, func(i int, j int) bool { return parents[i] < parents[j] })

	for _, p := range parents {
		pDeps, err := dag.Dependencies(p, nil)
		if err != nil {
			return nil, err
		}

		parents = append(parents, pDeps...)
	}

	return parents, nil
}

func (dag *DAG) Digest(id uint32, hash func([]byte) []byte) ([]byte, error) {
	if id == 0 {
		parents := dag.sinks
		sort.Slice(parents, func(i int, j int) bool { return parents[i] < parents[j] })

		data := []byte{}
		for _, parent := range parents {
			parentDigest, err := dag.Digest(parent, hash)
			if err != nil {
				return nil, err
			}

			data = append(data, hash(append([]byte{byte(DAGDigestTypeBranch)}, parentDigest...))...)
		}

		return hash(data), nil
	}

	for _, node := range dag.Nodes {
		if node.ID == id {
			leaf := append([]byte{byte(DAGDigestTypeLeaf)}, hash(node.Binary())...)

			parentEdges := dag.ParentEdges(id)
			parents := make([]uint32, len(parentEdges))
			for i := range parentEdges {
				parents[i] = parentEdges[i].Parent
			}
			sort.Slice(parents, func(i int, j int) bool { return parents[i] < parents[j] })

			if len(parents) == 0 {
				return hash(leaf), nil
			}

			var branches bytes.Buffer
			for _, parent := range parents {
				parentDigest, err := dag.Digest(parent, hash)
				if err != nil {
					return nil, err
				}

				branches.Write([]byte{byte(DAGDigestTypeBranch)})
				branches.Write(parentDigest)
			}

			return hash(append(leaf, branches.Bytes()...)), nil
		}
	}

	nodesStr := make([]string, len(dag.Nodes))
	for i, node := range dag.Nodes {
		nodesStr[i] = strconv.FormatUint(uint64(node.ID), 10)
	}

	return nil, errors.New("DAG: Vertex + " + strconv.FormatUint(uint64(id), 10) + " not found in in [" + strings.Join(nodesStr, ", ") + "]")
}

type DAGEdge struct {
	Child  uint32
	Parent uint32
}

func NewDAGEdge(child uint32, parent uint32) *DAGEdge {
	return &DAGEdge{
		Child:  child,
		Parent: parent,
	}
}

func NewDAGEdgeFromBinary(b []byte) (*DAGEdge, error) {
	l := len(b)
	lErr := errors.New("DAG: Invalid binary " + hex.EncodeToString(b) + " because of \"invalid length\"")
	i := 0

	if l < i+1 {
		return nil, lErr
	}
	bType := b[i]
	i += 1

	if bType != byte(DAGBinaryTypeEdge) {
		return nil, errors.New("DAG: Invalid binary " + hex.EncodeToString(b) + " because of \"invalid type, expected " + strconv.FormatUint(uint64(DAGBinaryTypeEdge), 10) + " got " + strconv.FormatUint(uint64(bType), 10) + "\"")
	}

	if l < i+4 {
		return nil, lErr
	}
	bChild := binary.BigEndian.Uint32(b[i : i+4])
	i += 4

	if l < i+4 {
		return nil, lErr
	}
	bParent := binary.BigEndian.Uint32(b[i : i+4])
	i += 4

	return &DAGEdge{
		Child:  bChild,
		Parent: bParent,
	}, nil
}

func (edge *DAGEdge) Binary() []byte {
	var buf bytes.Buffer

	buf.Write([]byte{byte(DAGBinaryTypeEdge)})

	childBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(childBytes, edge.Child)
	buf.Write(childBytes)

	parentBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(parentBytes, edge.Parent)
	buf.Write(parentBytes)

	return buf.Bytes()
}

type DAGVertex struct {
	ID   uint32
	Data []byte
}

func NewDAGVertex(data []byte) *DAGVertex {
	return &DAGVertex{
		ID:   rand.Uint32(),
		Data: data,
	}
}

func NewDAGVertexFromBinary(b []byte) (*DAGVertex, error) {
	l := len(b)
	lErr := errors.New("DAG: Invalid binary " + hex.EncodeToString(b) + " because of \"invalid length\"")
	i := 0

	if l < i+1 {
		return nil, lErr
	}
	bType := b[i]
	i += 1

	if bType != byte(DAGBinaryTypeVertex) {
		return nil, errors.New("DAG: Invalid binary " + hex.EncodeToString(b) + " because of \"invalid type, expected " + strconv.FormatUint(uint64(DAGBinaryTypeVertex), 10) + " got " + strconv.FormatUint(uint64(bType), 10) + "\"")
	}

	if l < i+4 {
		return nil, lErr
	}
	bID := binary.BigEndian.Uint32(b[i : i+4])
	i += 4

	if l < i+4 {
		return nil, lErr
	}
	bDataLength := binary.BigEndian.Uint32(b[i : i+4])
	i += 4

	if l < i+int(bDataLength) {
		return nil, lErr
	}
	bData := b[i : i+int(bDataLength)]
	i += int(bDataLength)

	return &DAGVertex{
		ID:   bID,
		Data: bData,
	}, nil
}

func (node *DAGVertex) Binary() []byte {
	var buf bytes.Buffer

	buf.Write([]byte{byte(DAGBinaryTypeVertex)})

	idBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(idBytes, node.ID)
	buf.Write(idBytes)

	dataLength := make([]byte, 4)
	binary.BigEndian.PutUint32(dataLength, uint32(len(node.Data)))
	buf.Write(dataLength)

	buf.Write(node.Data)

	return buf.Bytes()
}

func DigestHashSHA256(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}
