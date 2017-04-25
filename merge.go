package sketchmerge

import (
	"errors"
	"io"
	_"fmt"
	_ "reflect"
	_ "sort"
	"strconv"
	"strings"
	"log"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"os"
	"path/filepath"
	"github.com/mohae/deepcopy"
	"path"

)

type NodeEvent func(interface{}, Node, Node) bool

type Selection interface {
	Apply(v interface{}) (interface{}, Node, error)
	ApplyWithEvent(v interface{}, e NodeEvent) (interface{}, Node, error)
	GetKey() (interface{})
	GetCurrentPath() string
}

type Node interface {
	Selection
	SetNext(v Node)
	SetPrev(v Node)
	SetLast(v Node)
	GetNext() Node
	GetPrev() Node
	GetLast() Node
	GetFileName() string
 	GetFileAction() (FileActionType)


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
	InvalidMergeAction= errors.New("Invalid merge action")
)

func applyNext(nn Node, prevnn Node, v interface{}, e NodeEvent) (interface{}, Node,  error) {
	if nn == nil {
		return v, prevnn, nil
	}

	return nn.ApplyWithEvent(v, e)
}


type RootNode struct {
	NextNode Node
	PrevNode Node
	LastNode Node
	FileName string
	fileAction FileActionType
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


func (r *RootNode) GetFileName() (string) {
	return r.FileName
}

func (r *RootNode) GetFileAction() (FileActionType) {
	return r.fileAction
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

func (r *RootNode) GetCurrentPath() string {

	return "$"
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

func (m *MapSelection) GetCurrentPath() string {

	return "[\"" + m.Key + "\"]"
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

func (a *ArraySelection) GetCurrentPath() string {

	return "[" + strconv.Itoa(a.Key) + "]"
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

	if s[0] == 'A' {
		r += "A"
		s = s[1:]
	} else if s[0] == 'D' {
		r += "D"
		s = s[1:]
	}

	if s[0] == '~' {
		r += "~"
		s = s[1:]
		n := strings.Index(s, "~")
		r += s[0 : n+1]
		s = s[n+1:]
	}

	if s == "" {
		return r + "$"
	}

	if s[0] == '-' {
		r = "-"
		s = s[1:]
	} else if s[0] == '+' {
		r = "+"
		s = s[1:]
	} else if s[0] == '^' {
		r = "^"
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

func PathLength(s string) int {
	return strings.Count(s, "][")
}

func GetPath(n Node) string {
	var path string
	for n != nil  {
		path = n.GetCurrentPath() + path
		n = n.GetPrev()
	}
	if path == "" {
		path = "$"
	}
	return path
}

func getActionWithoutReverse(s string) string {
	if s == "" {
		return ""
	}

	if s[0] == 'R' {
		s = s[1:]
	}

	return s
}

func ReversAction(s1 string, s2 string) (string, string) {
	s1 = getActionWithoutReverse(s1)
	s2 = getActionWithoutReverse(s2)

	a1 := ReadFileAction(s1)

	s1 = FlatJsonPath(s1, false)
	s2 = FlatJsonPath(s2, false)

	if s1[0] == 'A' {
		sw := s1
		s1 = "D" + a1[1:] + s1
		s2 = sw
	} else if s1[0] == 'D' {
		sw := s1
		s1 = "A" + a1[1:] + s1
		s2 = sw
	} else if s1[0] == '+' {
		s1 = a1 + "-" + s1[1:]
		s2 = ""
	} else if s1[0] == '-' {
		s1 = ""
		s2 = ""
	} else {
		sw := s1
		s1 = a1 + s2
		s2 = sw
	}

	return s1, s2
}

func ReadFileKey(s string) string {

	if s == "" {
		return ""
	}

	r := ""

	if s[0] == 'A' {
		s = s[1:]
	} else if s[0] == 'D' {
		s = s[1:]
	}

	if s[0] == '~' {
		s = s[1:]
		n := strings.Index(s, "~")
		r += s[0 : n]
		return r
	}

	return ""
}

func ReadFileAction(s string) string {

	if s == "" {
		return ""
	}

	r := ""

	if s[0] == 'A' {
		r += "A"
		s = s[1:]
	} else if s[0] == 'D' {
		r += "D"
		s = s[1:]
	}

	if s[0] == '~' {
		r += "~"
		s = s[1:]
		n := strings.Index(s, "~")
		r += s[0 : n + 1]
		return r
	}

	return ""
}

func FlatJsonPath(s string, omitActions bool) string {
	if s == "" {
		return "$"
	}
	r := ""

	if s[0] == 'A' {
		s = s[1:]
	} else if s[0] == 'D' {
		s = s[1:]
	}

	if s[0] == '~' {
		s = s[1:]
		n := strings.Index(s, "~")
		s = s[n+1:]
	}

	if s == "" {
		return "$"
	}

	if s[0] == '-' {
		r = "-"
		s = s[1:]
	} else if s[0] == '+' {
		r = "+"
		s = s[1:]
	} else if s[0] == '^' {
		r = "^"
		s = s[1:]
	}
	if omitActions {
		r = ""
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
	var FileName string
	var fileAction FileActionType = MERGE

	if s[0] == 'A' {
		s = s[1:]
		fileAction = ADD
	} else if s[0] == 'D' {
		s = s[1:]
		fileAction = DELETE
	}

	if s[0] == '~' {
		s = s[1:]
		n := strings.Index(s, "~")
		FileName = s[0 : n]
		s = s[n+1:]
	}

	rt := RootNode{nil, nil, nil, FileName, fileAction}

	if s[0] == '-' {
		s = s[1:]
		action = ValueDelete
	} else if s[0] == '+' {
		s = s[1:]
		action = ValueAdd
	} else if s[0] == '^' {
		s = s[1:]
		action = SequenceChange
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

	if strings.HasPrefix(srcPath, "^") {

		return InvalidMergeAction
	}

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

	if !strings.HasPrefix(srcPath, "^") {

		return InvalidMergeAction
	}

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
			log.Printf("src_path %v %v\n", srcPath, dstPath)
			log.Printf("no doc2 pos %v %v\n", idxDoc1, idxDoc2)
			continue
		}


		if idxDoc1 < len(newslice) {
			newslice[idxDoc1] = slice[idxDoc2]
			slice[idxDoc2] = nil
		} else {
			log.Printf("src_path %v %v\n", srcPath, dstPath)
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

func decodeMergeFiles(doc1File string, doc2File string) (map[string]interface{}, map[string]interface{}, error) {

	fileDoc1, eDoc1 := ioutil.ReadFile(doc1File)
	if eDoc1 != nil {
		return nil, nil, eDoc1
	}

	fileDoc2, eDoc2 := ioutil.ReadFile(doc2File)
	if eDoc2 != nil {
		return nil, nil, eDoc2
	}

	var result1 map[string]interface{}
	var decoder1 = json.NewDecoder(bytes.NewReader(fileDoc1))
	decoder1.UseNumber()

	if err := decoder1.Decode(&result1); err != nil {
		return nil, nil, err
	}

	var result2 map[string]interface{}
	var decoder2 = json.NewDecoder(bytes.NewReader(fileDoc2))
	decoder2.UseNumber()

	if err := decoder2.Decode(&result2); err != nil {
		return nil, nil, err
	}

	return result1, result2, nil
}

func GetSortedDiffs(docDiffs map[string]interface{}, fileName string) []interface{} {
	sortedActions := make([]interface{}, len(docDiffs))

	k := 0

	for key, item := range docDiffs {
		newDep := DependentObj{key, item.(string), fileName}

		if k == 0 {
			sortedActions[0] = newDep
		} else {
			for i := k; i > 0; i-- {

				dep, isDep := sortedActions[i - 1].(DependentObj)
				if isDep {
					if PathLength(key) < PathLength(dep.JsonPath) {
						sortedActions[i - 1] = newDep
						sortedActions[i] = dep
					} else {
						sortedActions[i] = newDep
						break
					}
				}
			}
		}
		k++
	}

	return sortedActions
}

func merge(workingDirV1 string, workingDirV2 string, fileName string, objectKeyName string, docDiffs map[string]interface{} ) error {

	srcFilePath := workingDirV1 + string(os.PathSeparator) + fileName
	dstFilePath := workingDirV2 + string(os.PathSeparator) + fileName


	jsonDoc1, jsonDoc2, err := decodeMergeFiles(srcFilePath, dstFilePath)

	if err != nil {
		return err
	}

	mergeDoc := MergeDocuments{jsonDoc1, jsonDoc2}

	deleteActions := make(map[string]string)
	seqDiff := make(map[string]string)

	sortedActions := GetSortedDiffs(docDiffs, fileName)

	for i := range sortedActions {

		dep := sortedActions[i].(DependentObj)
		key := dep.JsonPath
		var item interface{} = dep.Ref

		if item == "" {
			deleteActions[key] = ""
		} else if strings.HasPrefix(key, "^") {
			seqDiff[key] = item.(string)
		} else {
			mergeDoc.MergeByJSONPath(key, item.(string))
		}
	}

	for key, _ := range deleteActions {
		mergeDoc.MergeByJSONPath("", key)
	}

	for key, item := range seqDiff {
		mergeDoc.MergeSequenceByJSONPath(objectKeyName, key, item)
	}

	data, err := json.Marshal(mergeDoc.DstDocument)

	if err != nil {
		return err
	}

	WriteToFile(dstFilePath, data)

	return nil
}

func updateFile(workingDirV1, workingDirV2, fileKey string) {
	targetFileName := workingDirV2 + string(os.PathSeparator) + fileKey
	baseFileName := path.Base(targetFileName)

	targetDir := strings.TrimSuffix(targetFileName, baseFileName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0777)
	}
	if _, err := os.Stat(targetFileName); os.IsExist(err) {
		err = os.Remove(targetFileName)
		if err != nil {
			log.Println(err)
		}
	}

	CopyFile(workingDirV1 + string(os.PathSeparator) + fileKey, targetFileName)
}

func segregateFileActions(workingDirV1 string, workingDirV2 string, mergeJSON FileStructureMerge)  {

	mergeMap := make(map[string]FileMerge)

	for i := range mergeJSON.MergeActions {
		fileName := mergeJSON.MergeActions[i].FileKey + mergeJSON.MergeActions[i].FileExt
		mergeMap[fileName] = mergeJSON.MergeActions[i]
	}

	for i := range mergeJSON.MergeActions {
		docDiffs := mergeJSON.MergeActions[i].FileDiff.Doc1Diffs

		copyDiffs := deepcopy.Copy(docDiffs).(map[string]interface{})
		for key, item := range copyDiffs {

			originalKey := key

			if strings.HasPrefix(key, "R") {

				newKey, newItem := ReversAction( key, item.(string))

				key = newKey
				item = newItem
				delete(mergeJSON.MergeActions[i].FileDiff.Doc1Diffs, originalKey)
				mergeJSON.MergeActions[i].FileDiff.Doc1Diffs[key] = item.(string)
			}

			if key == "" {
				continue
			}

			fileKey := ReadFileKey(key)
			fileAction := ReadFileAction(key)


			if fileKey != "" {
				delete(mergeJSON.MergeActions[i].FileDiff.Doc1Diffs, originalKey)
				mergeAction := mergeMap[fileKey]

				if strings.HasPrefix(fileAction, "A") {
					mergeAction.Action = ADD

					updateFile(workingDirV1, workingDirV2, fileKey)
					continue
				} else if strings.HasPrefix(fileAction, "D") {
					mergeAction.Action = DELETE
					continue
				}

				if mergeAction.FileDiff.Doc1Diffs == nil {
					mergeAction.FileDiff.Doc1Diffs = make(map[string]interface{})
				}

				mergeAction.FileDiff.Doc1Diffs[FlatJsonPath(key, false)] = FlatJsonPath(item.(string), false)
			}
		}


	}
}

func mergeActions(workingDirV1 string, workingDirV2 string, mergeJSON FileStructureMerge) error {

	segregateFileActions(workingDirV1, workingDirV2, mergeJSON)

	for i := range mergeJSON.MergeActions {

		fileName := mergeJSON.MergeActions[i].FileKey + mergeJSON.MergeActions[i].FileExt

		if !mergeJSON.MergeActions[i].IsDirectory && mergeJSON.MergeActions[i].Action == ADD {
			updateFile(workingDirV1, workingDirV2, fileName)
		}

		if mergeJSON.MergeActions[i].FileDiff.Doc1Diffs == nil {
			continue
		}

		if err := merge(workingDirV1, workingDirV2,
			fileName,
			mergeJSON.MergeActions[i].FileDiff.ObjectKeyName,
			mergeJSON.MergeActions[i].FileDiff.Doc1Diffs); err!=nil {
			continue
		}

	}
	return nil
}

func ProcessFileMerge(mergeFileName string, sketchFileV1 string, sketchFileV2 string, outputDir string) error {

	isSrcDir := false
	isDstDir := false

	sketchFileV1Info, errv1 := os.Stat(sketchFileV1)

	if errv1 != nil {
		return errv1
	}

	isSrcDir = sketchFileV1Info.IsDir()

	sketchFileV2Info, errv2 := os.Stat(sketchFileV2)

	if errv2 != nil {
		return errv2
	}

	isDstDir = sketchFileV2Info.IsDir()

	workingDirV1, err1 := prepareWorkingDir(!isSrcDir)
	if err1!=nil {
		return err1
	}
	defer removeWorkingDir(workingDirV1, isSrcDir)

	if isSrcDir {
		workingDirV1 = sketchFileV1
	}

	workingDirV2, err2 := prepareWorkingDir(!isDstDir)
	if  err2!=nil {
		return err2
	}
	defer removeWorkingDir(workingDirV2, isDstDir)

	if isDstDir {
		workingDirV2 = sketchFileV2
	}

	if !isSrcDir {
		if err := Unzip(sketchFileV1, workingDirV1); err != nil {
			return err
		}
	}

	if !isDstDir {
		if err := Unzip(sketchFileV2, workingDirV2); err != nil {
			return err
		}
	}

	mergeFile, err := ioutil.ReadFile(mergeFileName)
	if err != nil {
		return err
	}

	var mergeJSON FileStructureMerge
	var decoder = json.NewDecoder(bytes.NewReader(mergeFile))
	decoder.UseNumber()

	if err := decoder.Decode(&mergeJSON); err != nil {
		return  err
	}

	if err := mergeActions(workingDirV1, workingDirV2, mergeJSON); err != nil  {
		return err
	}

	if !isDstDir {
		sketchFile := outputDir + string(os.PathSeparator) + strings.TrimPrefix(sketchFileV2, filepath.Dir(sketchFileV2))
		//similar to zip -y -r -q -8 testVCS2.sketch ./pages/ ./previews/ document.json meta.json user.json
		Zipit(workingDirV2, sketchFile)
	}

	return nil

}
