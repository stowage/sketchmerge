package sketchmerge

import (
	"errors"
	"io"
	_"fmt"
	_ "encoding/json"
	_ "reflect"
	_ "sort"
	"strconv"
	"strings"
	"log"
)

type NodeEvent func(interface{}, Node, Node) bool

type Selection interface {
	Apply(v interface{}) (interface{}, Node, error)
	ApplyWithEvent(v interface{}, e NodeEvent) (interface{}, Node, error)
	GetKey() (interface{})
}

type Node interface {
	Selection
	SetNext(v Node)
	SetPrev(v Node)
	SetLast(v Node)
	GetNext() Node
	GetPrev() Node
	GetLast() Node

}

type MergeDocuments struct{
	SrcDocument map[string]interface{}
	DstDocument map[string]interface{}
}

const (
	ValueChange = iota
	ValueDelete
	ValueAdd
	SequenceChange
)

type ApplyAction uint8

var (
	MapTypeError      = errors.New("Expected Type to be a Map.")
	ArrayTypeError    = errors.New("Expected Type to be an Array.")
	SyntaxError       = errors.New("Bad Syntax.")
	NotFound          = errors.New("Not Found")
	IndexOutOfBounds  = errors.New("Out of Bounds")
)

func applyNext(nn Node, prevnn Node, v interface{}, e NodeEvent) (interface{}, Node,  error) {
	if nn == nil {
		return v, prevnn, nil
	}

	return nn.ApplyWithEvent(v, e)
}

func insert(slice []interface{}, index int , value interface{}) ([]interface{}) {

	slice = slice[0 : len(slice)+1]
	copy(slice[index+1:], slice[index:])
	slice[index] = value
	return slice
}

type RootNode struct {
	NextNode Node
	PrevNode Node
	LastNode Node
}

func (r *RootNode) SetNext(n Node) {
	r.NextNode = n
}

func (r *RootNode) SetPrev(n Node) {
	r.PrevNode = n
}

func (r *RootNode) GetPrev() (Node) {
	return r.PrevNode
}

func (r *RootNode) GetLast() (Node) {
	return r.LastNode
}

func (r *RootNode) GetNext() (Node) {
	return r.NextNode
}

func (r *RootNode) SetLast(n Node) {
	r.LastNode = n
}

func (r *RootNode) Apply(v interface{}) ( interface{}, Node, error) {
	return r.ApplyWithEvent(v, nil)
}

func (r *RootNode) ApplyWithEvent(v interface{}, e NodeEvent) ( interface{}, Node, error) {
	if e != nil && !e(v, nil, r) {
		return v, r, nil
	}

	return applyNext(r.NextNode, r, v, e)
}

func (r *RootNode) GetKey() interface{} {
	return nil
}

type MapSelection struct {
	Key string
	RootNode
}
func (m *MapSelection) Apply(v interface{}) ( interface{}, Node, error) {
	return m.ApplyWithEvent(v, nil)
}

func (m *MapSelection) ApplyWithEvent(v interface{}, e NodeEvent) ( interface{}, Node, error) {

	mv, ok := v.(map[string]interface{})
	if !ok {
		return v, m, MapTypeError
	}
	nv, ok := mv[m.Key]
	if !ok {
		return nil, m, NotFound
	}
	if e != nil && !e(nv, m.PrevNode, m) {
		return nv, m, nil
	}
	return applyNext(m.NextNode, m, nv, e)
}

func (m *MapSelection) GetKey() interface{} {

	return m.Key
}


type ArraySelection struct {
	Key int
	RootNode
}
func (a *ArraySelection) Apply(v interface{}) (interface{}, Node, error) {
	return a.ApplyWithEvent(v, nil)
}

func (a *ArraySelection) ApplyWithEvent(v interface{}, e NodeEvent) (interface{}, Node, error) {
	arv, ok := v.([]interface{})
	if !ok {
		return v, a, ArrayTypeError
	}
	// Check to see if the value is in bounds for the array.
	if a.Key < 0 || a.Key >= len(arv) {
		return nil, a, IndexOutOfBounds

	}

	if e != nil && !e(arv[a.Key], a.PrevNode, a) {
		return arv[a.Key], a, nil
	}

	return applyNext(a.NextNode, a, arv[a.Key], e)
}

func (a *ArraySelection) GetKey() interface{} {
	return a.Key
}

func minNotNeg1(a int, bs ...int) int {
	m := a
	for _, b := range bs {
		if a == -1 || (b != -1 && b < m) {
			m = b
		}
	}
	return m
}

func normalize(s string) string {
	if s == "" {
		return "$"
	}
	r := ""

	if s[0] == '-' {
		r = "-"
		s = s[1:]
	} else if s[0] == '+' {
		r = "+"
		s = s[1:]
	}

	r += "$"

	if s[0] == '$' {
		s = s[1:]
	}

	for len(s) > 0 {

		// Grab the bracketed entries
		for len(s) > 0 && s[0] == '[' {
			n := strings.Index(s, "]")
			r += s[0 : n+1]
			s = s[n+1:]
		}
		if len(s) <= 0 {
			break
		}

		n := minNotNeg1(strings.Index(s, "["))
		if n == 0 {
			continue
		}
		if n != -1 {
			r += `["` + s[:n] + `"]`
			s = s[n:]
		} else {
			r += `["` + s + `"]`
			s = ""
		}

	}
	return r
}

func getNode(s string) (Node, string, error) {
	var rs string
	if len(s) == 0 {
		return nil, s, io.EOF
	}
	n := strings.Index(s, "]")
	if n == -1 {
		return nil, s, SyntaxError
	}
	if len(s) > n {
		rs = s[n+1:]
	}
	switch s[:2] {
	case "[\"":
		//fmt.Printf("parse map %v\n", s[2 : n-1])
		return &MapSelection{Key: s[2 : n-1]}, rs, nil
	default: // Assume it's a array index otherwise.
		i, err := strconv.Atoi(s[1:n])
		if err != nil {
			return nil, rs, SyntaxError
		}
		//fmt.Printf("parse array %v\n", i)
		return &ArraySelection{Key: i}, rs, nil
	}
}


func Parse(s string) (Node, ApplyAction, error) {

	var nn Node
	var err error
	var action ApplyAction = ValueChange
	s = normalize(s)
	rt := RootNode{nil, nil, nil}

	if s[0] == '-' {
		s = s[1:]
		action = ValueDelete
	} else if s[0] == '+' {
		s = s[1:]
		action = ValueAdd
	}

	s = s[1:]
	var c Node
	c = &rt
	for len(s) > 0 {
		nn, s, err = getNode(s)
		if err != nil {
			return nil, action,  err
		}
		c.SetNext(nn)
		nn.SetPrev(c)
		c = nn
		//fmt.Printf("node %v\n", nn.GetKey())
	}
	rt.SetLast(c)
	return &rt, action, nil
}

func (md * MergeDocuments) setArrayElement(srcNode Node, dstNode Node) error {


	src, _,srcerr := srcNode.Apply( md.SrcDocument)
	dst, lastDstNode, dsterr := dstNode.Apply( md.DstDocument)

	if srcerr != nil {
		return srcerr
	}
	if dsterr != nil {
		return dsterr
	}

	prevNode := lastDstNode.GetPrev();
	prevNode.SetNext(nil)

	if prevNode == nil {
		return NotFound
	}

	_ = dst

	fordst, _, err := dstNode.Apply( md.DstDocument)

	if err != nil {
		return err
	}

	fordst.([]interface{})[lastDstNode.GetKey().(int)] = src

	return nil
}

func (md * MergeDocuments) addArrayElement(srcNode Node, dstNode Node) error {
	src, _, srcerr := srcNode.Apply(md.SrcDocument)
	dst, lastDstNode, dsterr := dstNode.Apply(md.DstDocument)

	if srcerr != nil {
		return srcerr
	}
	if dsterr != nil {
		return dsterr
	}

	prevNode := lastDstNode.GetPrev();
	prevNode.SetNext(nil)

	if prevNode == nil {
		return NotFound
	}
	fordst, _, err := dstNode.Apply(md.DstDocument)

	if err != nil {
		return err
	}

	fordst.(map[string]interface{})[lastDstNode.GetKey().(string)] = append(dst.([]interface{}), src)

	return nil
}

func (md * MergeDocuments) deleteArrayElement(dstNode Node) error {
	_, lastDstNode, err := dstNode.Apply(md.DstDocument)

	if err != nil {
		return err
	}

	prevNode := lastDstNode.GetPrev()
	prevNode.SetNext(nil)

	if prevNode == nil {
		return NotFound
	}

	fordst, arrLastNode, err := dstNode.Apply(md.DstDocument)

	if err != nil {
		return err
	}

	prevNode = prevNode.GetPrev()
	prevNode.SetNext(nil)

	if prevNode == nil {
		return NotFound
	}

	findst, _, finerr := dstNode.Apply(md.DstDocument)

	if finerr != nil {
		return finerr
	}

	index := lastDstNode.GetKey().(int)
	finArr := append(fordst.([]interface{})[:index], fordst.([]interface{})[index+1:]...)
	//fmt.Printf("Err: %v %v\n", arrLastNode.GetKey(), findst)
	findst.(map[string]interface{})[arrLastNode.GetKey().(string)] = finArr
	return nil
}

func (md * MergeDocuments) addMapElement(srcNode Node, dstNode Node) error {
	src, lastSrcNode, srcerr := srcNode.Apply(md.SrcDocument)
	dst, _, dsterr := dstNode.Apply(md.DstDocument)

	if srcerr != nil {
		return srcerr
	}

	if dsterr != nil {
		return dsterr
	}

	key := lastSrcNode.GetKey()

	dst.(map[string]interface{})[key.(string)] = src

	return nil
}

func (md * MergeDocuments) setMapElement(srcNode Node, dstNode Node) error {
	src, lastSrcNode, srcerr := srcNode.Apply(md.SrcDocument)
	_, lastDstNode, dsterr := dstNode.Apply(md.DstDocument)

	if srcerr != nil {
		return srcerr
	}

	if dsterr != nil {
		return dsterr
	}
	prevNode := lastDstNode.GetPrev();
	prevNode.SetNext(nil)

	if prevNode == nil {
		return NotFound
	}


	fordst, _, err := dstNode.Apply(md.DstDocument)

	if err != nil {
		return err
	}

	key := lastSrcNode.GetKey()

	fordst.(map[string]interface{})[key.(string)] = src


	return nil
}

func (md * MergeDocuments) deleteMapElement(dstNode Node) error {
	_, lastDstNode, err := dstNode.Apply(md.DstDocument)
	key := lastDstNode.GetKey()

	if err != nil {
		return err
	}

	prevNode := lastDstNode.GetPrev();
	prevNode.SetNext(nil)

	if prevNode == nil {
		return NotFound
	}

	fordst, _, err := dstNode.Apply(md.DstDocument)

	delete(fordst.(map[string]interface{}), key.(string))

	return nil
}



func (md * MergeDocuments) MergeByJSONPath(srcPath string, dstPath string) error {

	srcSel, srcact, srcerr := Parse(srcPath)
	dstSel, dstact, dsterr := Parse(dstPath)

	if srcerr != nil {
		return srcerr
	}

	if dsterr != nil {
		return dsterr
	}

	_, lastSrcNode, srcerr := srcSel.Apply(md.SrcDocument)

	if srcerr != nil {
		return srcerr
	}

	if srcPath == "" {
		srcact = dstact
		_, lastSrcNode, srcerr = dstSel.Apply(md.DstDocument)
	}

	switch selType := lastSrcNode.(type) {
	default:
		_ = selType
		return md.setMapElement(srcSel, dstSel)
	case *MapSelection:
		if srcact == ValueDelete {
			return md.deleteMapElement(dstSel)
		}

		if srcact == ValueAdd {
			return md.addMapElement(srcSel, dstSel)
		}

		return md.setMapElement(srcSel, dstSel)
	case *ArraySelection:
		if srcact == ValueDelete {
			return md.deleteArrayElement(dstSel)
		}

		if srcact == ValueAdd {
			return md.addArrayElement(srcSel, dstSel)
		}

		return md.setArrayElement(srcSel, dstSel)

	}



	return nil
}

func (md * MergeDocuments) MergeSequenceByJSONPath(objectKeyName string, srcPath string, dstPath string) error {

	srcSel, _, srcerr := Parse(srcPath)
	dstSel, _, dsterr := Parse(dstPath)

	if srcerr != nil {
		return srcerr
	}

	if dsterr != nil {
		return dsterr
	}

	fordst, _, fderr := dstSel.Apply(md.DstDocument)
	if fderr != nil {
		return fderr;
	}

	forsrc, _, fserr := srcSel.Apply(md.SrcDocument)
	if fserr != nil {
		return fserr;
	}

	//build id associations by objectID
	doc1Changes, _ := CompareSequence(objectKeyName, forsrc.([]interface{}), fordst.([]interface{}))

	slice := fordst.([]interface{})
	newslice := make([]interface{}, len(slice))

	for idxDoc1, idxDoc2 := range doc1Changes {
		if idxDoc2 == -1 {
			log.Printf("no doc2 pos %v %v\n", idxDoc1, idxDoc2)
			continue
		}


		if idxDoc1 < len(newslice) {
			newslice[idxDoc1] = slice[idxDoc2]
			slice[idxDoc2] = nil
		} else {
			log.Printf("Sequence out of range %v %v\n", idxDoc1, len(slice))
		}
	}

	j := 0
	for i := range newslice {
		if newslice[i] == nil {
			for k:=j ; k<len(slice); k++ {
				if slice[k] != nil {
					newslice[i] = slice[k]
					slice[k] = nil
					j = k + 1
				}
			}
		}
	}

	copy(slice, newslice)

	return nil
}


