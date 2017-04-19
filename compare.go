package sketchmerge


import (
	"fmt"
	"os"
	"flag"
	"path/filepath"
	"strings"
	"encoding/json"
	"strconv"
	_ "github.com/NodePrime/jsonpath"
	"github.com/mohae/deepcopy"
	_ "github.com/pquerna/ffjson/ffjson"
	"io/ioutil"
	"bytes"
	"time"
	"log"
	_ "sync"
	"reflect"
)

// Structure of sketch folder
type SketchFileStruct struct {
	fileSet map[string] interface{}
	name string
}
type IVoid interface {}

// Sketch file structure comparison
type FileActionType uint8

//File structure merge actions
const (
	MERGE = iota
	DELETE
	ADD
)

//Merge type for actions
type MergeActionType uint8

//File merge operations
type FileMerge struct {
	FileKey string `json:"file_key"`
	FileExt string `json:"file_ext"`
	IsDirectory bool `json:"is_directory"`
	Action FileActionType `json:"file_copy_action"`
	FileDiff JsonStructureCompare `json:"file_diff,omitempty"`
}

//File structure merge actions (all)
type FileStructureMerge struct {
	MergeActions []FileMerge `json:"merge_actions"`
}


//Difference of two json documents in jsonpath notations
type JsonStructureCompare struct {
	//differences for doc1 vs doc2 in jsonpath request
	Doc1Diffs map[string]interface{} `json:"src_to_dst_diff,omitempty"`

	//differences for doc2 vs doc1 in jsonpath request
	Doc2Diffs map[string]interface{} `json:"dst_to_src_diff,omitempty"`

	//object relocation
	Doc1ObjRelocate map[string]interface{} `json:"src_obj_relocate,omitempty"`

	//object relocation
	Doc2ObjRelocate map[string]interface{} `json:"dst_obj_relocate,omitempty"`

	//key element for arrays elements to check their order
	ObjectKeyName string `json:"seq_key,omitempty"`

	//Dependent objects for src document
	DepDoc1 * DependentObjects `json:"dep_src,omitempty"`

	//Dependent objects for dst document
	DepDoc2 * DependentObjects `json:"dep_dst,omitempty"`

}

//Getting file structure of two dirs
func ExtractSketchDirStruct(baseDir string, newDir string) (SketchFileStruct, SketchFileStruct) {
	var baseFileStruct SketchFileStruct
	var newFileStruct SketchFileStruct

	baseFileStruct.fileSet = make(map[string]interface{})
	newFileStruct.fileSet = make(map[string]interface{})
	baseFileStruct.name = baseDir
	newFileStruct.name = newDir

	err1 := filepath.Walk(baseDir + string(os.PathSeparator) , func(path string, f os.FileInfo, err error) error {
		name := strings.TrimPrefix(path, baseFileStruct.name + string(os.PathSeparator))
		if name != "" {
			baseFileStruct.fileSet[name] = f
		}
		return nil
	})

	err2 := filepath.Walk(newDir + string(os.PathSeparator), func(path string, f os.FileInfo, err error) error {
		name := strings.TrimPrefix(path, newFileStruct.name + string(os.PathSeparator))
		if name != "" {
			newFileStruct.fileSet[name] = f
		}
		return nil
	})

	_=err1
	_=err2
	//fmt.Printf("Errors %v %v\n", err1, err2)

	return baseFileStruct, newFileStruct
}

//Creates file structure changes description
func (fs*FileStructureMerge) FileSetChange(baseSet SketchFileStruct, newSet SketchFileStruct)  {
	for key, item := range baseSet.fileSet {
		mergeAction := new(FileMerge)
		mergeAction.FileExt = filepath.Ext(key)
		mergeAction.FileKey = strings.TrimSuffix(key, mergeAction.FileExt)

		_, ok := newSet.fileSet[key];

		if ok {
			mergeAction.Action = MERGE
		} else {
			mergeAction.Action = DELETE
		}
		delete(newSet.fileSet, key)

		mergeAction.IsDirectory = item.(os.FileInfo).IsDir()

		fs.MergeActions = append(fs.MergeActions, *mergeAction)
	}

	for key := range newSet.fileSet {
		mergeAction := new(FileMerge)
		mergeAction.FileKey = key
		mergeAction.Action = ADD
		fs.MergeActions = append(fs.MergeActions, *mergeAction)
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func (jsc * JsonStructureCompare) AddDependentObjects(docTreeMap map[string]interface{}, dep * DependentObjects, jsonpath string)  {
	for key, item := range docTreeMap {
		dep.AddDependentObject(nil, key, item, jsonpath)
	}
}

//Compare properties of dictionary node
func (jsc * JsonStructureCompare) CompareProperties(doc1TreeMap map[string]interface{}, doc2TreeMap map[string]interface{}, pathDoc1 string, pathDoc2 string)  (string, string, bool) {
	//defer timeTrack(time.Now(), "CompareProperties " + path)

	doc1ObjectKeyValue := doc1TreeMap[jsc.ObjectKeyName];
	doc2ObjectKeyValue := doc2TreeMap[jsc.ObjectKeyName];

	//fmt.Printf("keys: %v %v\n", doc1ObjectKeyValue, doc2ObjectKeyValue)

	if doc1ObjectKeyValue != nil && doc2ObjectKeyValue != nil {
		if doc1ObjectKeyValue != doc2ObjectKeyValue || pathDoc1 != pathDoc2 {
			jsc.addDoc1ObjectRelocated(doc1ObjectKeyValue.(string), pathDoc1, "CompareProperties");
			jsc.addDoc2ObjectRelocated(doc2ObjectKeyValue.(string), pathDoc2, "CompareProperties");
		}
	}

	//go thru all properties of doc1
	hasDiff := false
	for key, item := range doc1TreeMap {

		if subtree, ok := doc2TreeMap[key]; ok {
			//if it has a difference append to difference map
			if __jsonpath1, __jsonpath2 ,ok := jsc.CompareDocuments(&item, &subtree, pathDoc1  + `["` + key + `"]`, pathDoc2  + `["` + key + `"]`); !ok {
				jsc.addDoc1Diff(__jsonpath1, __jsonpath2, "CompareProperties")
				jsc.addDoc2Diff(__jsonpath2, __jsonpath1, "CompareProperties")
				hasDiff = true
			}
		} else {
			jsc.addDoc2Diff("-" + pathDoc1 + `["` + key + `"]`,"", "CompareProperties")
			jsc.addDoc1Diff("+" + pathDoc1 + `["` + key + `"]`, pathDoc2, "CompareProperties")
			hasDiff = true
		}


	}

	if hasDiff {
		for key, item := range doc1TreeMap {
			if key != jsc.ObjectKeyName {
				jsc.DepDoc1.AddDependentObject(doc1ObjectKeyValue, key, item, pathDoc1)
			}
		}
	}


	hasDiff = false
	//collect only properties not doc1
	for key, _:= range doc2TreeMap {
		if _, ok := doc1TreeMap[key]; !ok {
			jsc.addDoc1Diff("-" + pathDoc2 + `["` + key + `"]`,"","CompareProperties")
			jsc.addDoc2Diff("+" + pathDoc2 + `["` + key + `"]`, pathDoc1, "CompareProperties")
			hasDiff = true
		}
	}

	if hasDiff {
		for key, item := range doc2TreeMap {
			if key != jsc.ObjectKeyName {
				jsc.DepDoc2.AddDependentObject(doc2ObjectKeyValue, key, item, pathDoc2)
			}
		}
	}

	return pathDoc1, pathDoc2, true
}

//Compare array sequence of json node for objectKeyName
func CompareSequence(objectKeyName string, doc1TreeArray []interface{}, doc2TreeArray []interface{}) (map[int]int, map[int]int) {
	//defer timeTrack(time.Now(), "CompareSequence" + path)
	doc1Changes := make(map[int]int, len(doc1TreeArray))
	doc2Changes := make(map[int]int, len(doc2TreeArray))
	// create map of indexes for each object
	keysDoc1 := make(map[string]int, len(doc1TreeArray))
	keysDoc2 := make(map[string]int, len(doc2TreeArray))

	//put doc1 indeces to map by given key
	for index, item := range doc1TreeArray {
		if itemTreeMap, isItemMap := item.(map[string]interface{}); isItemMap {
			if objectId, ok := itemTreeMap[objectKeyName]; ok {
				keysDoc1[objectId.(string)] = index
			}
		}

	}

	//put doc2 indeces to map by given key
	for index, item := range doc2TreeArray {
		if itemTreeMap, isItemMap := item.(map[string]interface{}); isItemMap {
			if objectId, ok := itemTreeMap[objectKeyName]; ok {
				keysDoc2[objectId.(string)] = index
			}
		}

	}

	//build index change map for doc1
	for key, idxDoc1 := range keysDoc1 {
		if idxDoc2, ok := keysDoc2[key]; ok {
			//NOTE: indeces could be different
			doc1Changes[idxDoc1] = idxDoc2
		} else {
			doc1Changes[idxDoc1] = -1
		}
	}

	//build index change map for doc2
	for key, idxDoc2 := range keysDoc2 {
		if idxDoc1, ok := keysDoc1[key]; ok {
			//NOTE: indeces could be different
			doc2Changes[idxDoc2] = idxDoc1
		} else {
			doc2Changes[idxDoc2] = -1
		}

		//log.Println("doc2Changes:" + strconv.Itoa(len(keysDoc2)) +":" + strconv.Itoa(idxDoc2) +":"+ strconv.Itoa(doc2Changes[idxDoc2]) )
	}


	return doc1Changes, doc2Changes
}


func (jsc * JsonStructureCompare) addDoc1Diff(jsonpathDoc1 string, jsonpathDoc2 interface{}, from string) {
	//log.Printf("doc1Diff: %v %v %v\n", from, jsonpathDoc1, jsonpathDoc2)
	jsc.Doc1Diffs[jsonpathDoc1] = jsonpathDoc2
}

func (jsc * JsonStructureCompare) addDoc2Diff(jsonpathDoc1 string, jsonpathDoc2 interface{}, from string) {
	//log.Printf("doc2Diff: %v %v %v\n", from, jsonpathDoc1, jsonpathDoc2)
	jsc.Doc2Diffs[jsonpathDoc1] = jsonpathDoc2
}

func (jsc * JsonStructureCompare) addDoc1SeqDiff(jsonpathDoc1 string, jsonpathDoc2 interface{}, from string) {
	//log.Printf("doc1SeqDiff: %v %v %v\n", from, jsonpathDoc1, jsonpathDoc2)
	jsc.Doc1Diffs[jsonpathDoc1] = jsonpathDoc2
}

func (jsc * JsonStructureCompare) addDoc2SeqDiff(jsonpathDoc1 string, jsonpathDoc2 interface{}, from string) {
	//log.Printf("doc2SeqDiff: %v %v %v\n", from, jsonpathDoc1, jsonpathDoc2)
	jsc.Doc2Diffs[jsonpathDoc1] = jsonpathDoc2
}

func (jsc * JsonStructureCompare) addDoc1ObjectRelocated(objectKeyValue string, jsonpathDoc interface{}, from string) {
	//log.Printf("Doc1ObjRelocate: %v %v %v\n", from, objectKeyValue, jsonpathDoc)
	jsc.Doc1ObjRelocate[objectKeyValue] = jsonpathDoc
}

func (jsc * JsonStructureCompare) addDoc2ObjectRelocated(objectKeyValue string, jsonpathDoc interface{}, from string) {
	//log.Printf("Doc2ObjRelocate: %v %v %v\n", from, objectKeyValue, jsonpathDoc)
	jsc.Doc2ObjRelocate[objectKeyValue] = jsonpathDoc
}

func (jsc * JsonStructureCompare) removeDoc1ObjectRelocated(objectKeyValue string, from string) {
	delete(jsc.Doc1ObjRelocate, objectKeyValue)
}

func (jsc * JsonStructureCompare) removeDoc2ObjectRelocated(objectKeyValue string, from string) {
	delete(jsc.Doc2ObjRelocate, objectKeyValue)
}

//Compare each element in array node
func (jsc * JsonStructureCompare) CompareSlices(doc1TreeArray []interface{}, doc2TreeArray []interface{}, pathDoc1 string, pathDoc2 string) (string, string, bool) {
	//defer timeTrack(time.Now(), "CompareSlices " + path)
	doc1Changes, doc2Changes := CompareSequence(jsc.ObjectKeyName, doc1TreeArray, doc2TreeArray)

	doc1ChangesCopy := deepcopy.Copy(doc1Changes).(map[int]int)
	doc2ChangesCopy := deepcopy.Copy(doc2Changes).(map[int]int)

	//go thru array associations with the same objectKeyName for doc1
	for idxDoc1, idxDoc2 := range doc1Changes {
		jsonpathDoc1 := strings.Join([]string{pathDoc1, "[", strconv.Itoa(idxDoc1), "]"}, "")
		jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", strconv.Itoa(idxDoc2), "]"}, "")
		if idxDoc1 == idxDoc2 {
			//remove similar indeces
			delete(doc1ChangesCopy, idxDoc1)
		}
		if idxDoc2 == -1 {
			//if there is no such element in doc2 array
			jsc.addDoc2Diff("-" + jsonpathDoc1, "","CompareSlices")
			jsc.addDoc1Diff("+" + jsonpathDoc1, pathDoc2, "CompareSlices")
			jsc.DepDoc2.AddDependentPath( "-" + jsonpathDoc1, "^" + pathDoc1, "^" + pathDoc2)
			jsc.CompareDocuments(&(doc1TreeArray[idxDoc1]), nil, jsonpathDoc1, "");
		} else if __jsonpath1, __jsonpath2, ok := jsc.CompareDocuments(&(doc1TreeArray[idxDoc1]), &(doc2TreeArray[idxDoc2]), jsonpathDoc1, jsonpathDoc2); !ok {
			jsc.addDoc1Diff(__jsonpath1, __jsonpath2, "CompareSlices")
			jsc.addDoc2Diff(__jsonpath2, __jsonpath1, "CompareSlices")
		}
	}

	//go thru array associations with the same objectKeyName for doc2
	for idxDoc2, idxDoc1 := range doc2Changes {
		if idxDoc2 == idxDoc1 {
			//remove similar indeces
			delete(doc2ChangesCopy, idxDoc2)
		}
		jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", strconv.Itoa(idxDoc2), "]"}, "")

		if idxDoc1 == -1 {
			//if there is no such element in doc1 array
			jsc.addDoc1Diff("-" + jsonpathDoc2, "", "CompareSlices")
			jsc.addDoc2Diff("+" + jsonpathDoc2, pathDoc1, "CompareSlices")
			jsc.DepDoc1.AddDependentPath( "-" + jsonpathDoc2, "^" + pathDoc2, "^" + pathDoc1)
			jsc.CompareDocuments(nil, &(doc2TreeArray[idxDoc2]), "", jsonpathDoc2);


		}
	}

	if len(doc1Changes) == 0 && len(doc2Changes) == 0 {
		if !reflect.DeepEqual(doc1TreeArray, doc2TreeArray) {
			jsc.addDoc1Diff(pathDoc1, pathDoc2, "CompareSlices")
			jsc.addDoc2Diff(pathDoc2, pathDoc1, "CompareSlices")

			for idxDoc1 := range doc1TreeArray {
				jsonpathDoc1 := strings.Join([]string{pathDoc1, "[", strconv.Itoa(idxDoc1), "]"}, "")
				jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", strconv.Itoa(idxDoc1), "]"}, "")
				if idxDoc1 >= len(doc2TreeArray) {
					jsc.addDoc2Diff("-" + jsonpathDoc1, "","CompareSlices")
					jsc.addDoc1Diff("+" + jsonpathDoc1, pathDoc2, "CompareSlices")
					continue
				}

				if __jsonpath1, __jsonpath2, ok := jsc.CompareDocuments(&(doc1TreeArray[idxDoc1]), &(doc2TreeArray[idxDoc1]), jsonpathDoc1, jsonpathDoc2); !ok {
					jsc.addDoc1Diff(__jsonpath1, __jsonpath2, "CompareSlices")
					jsc.addDoc2Diff(__jsonpath2, __jsonpath1, "CompareSlices")
				}
			}

			if len(doc2TreeArray) > len(doc1TreeArray) {
				idxStart := len(doc1TreeArray)
				idxEnd := len(doc2TreeArray)

				for idxDoc2 := idxStart; idxDoc2 < idxEnd; idxDoc2++ {
					jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", strconv.Itoa(idxDoc2), "]"}, "")
					if idxDoc2 >= len(doc1TreeArray) {
						jsc.addDoc1Diff("-"+jsonpathDoc2, "", "CompareSlices")
						jsc.addDoc2Diff("+"+jsonpathDoc2, pathDoc1, "CompareSlices")


						continue
					}
				}
			}
		}
	}

	//set index change maps into differences map only for changed indeces
	if len(doc1ChangesCopy) > 0 || len(doc2ChangesCopy) > 0{
		jsc.addDoc1SeqDiff("^" + pathDoc1, "^" + pathDoc2, "CompareSequence")
		jsc.addDoc2SeqDiff("^" + pathDoc2, "^" + pathDoc1, "CompareSequence")
	}


	return pathDoc1, pathDoc2, true
}

func (jsc * JsonStructureCompare) CompareDocuments(doc1 *interface{}, doc2 *interface{}, pathDoc1 string, pathDoc2 string) (string, string, bool) {
	//defer timeTrack(time.Now(), "CompareDocuments " + path)
	//try to convert to json type doc1
	if doc1 == nil {
		doc2TreeMap, isDoc2Map := (*doc2).(map[string]interface{})
		if isDoc2Map {
			//if second is dictionary add dependent object
			jsc.AddDependentObjects(doc2TreeMap, jsc.DepDoc2, pathDoc2)
			return pathDoc1, pathDoc2, false;
		}
		return pathDoc1, pathDoc2, true;
	}

	if doc2 == nil {
		doc1TreeMap, isDoc1Map := (*doc1).(map[string]interface{})
		if isDoc1Map {
			//if first is dictionary add dependent object
			jsc.AddDependentObjects(doc1TreeMap, jsc.DepDoc1, pathDoc1)
			return pathDoc1, pathDoc2, false;
		}
		return pathDoc1, pathDoc2, true;
	}

	doc1TreeMap, isDoc1Map := (*doc1).(map[string]interface{})
	doc1TreeArray, isDoc1Array := (*doc1).([]interface{})

	//try to convert to json type doc2
	doc2TreeMap, isDoc2Map := (*doc2).(map[string]interface{})
	doc2TreeArray, isDoc2Array := (*doc2).([]interface{})

	if isDoc1Array && isDoc2Array {
		//if both are arrays compare arrays
		return jsc.CompareSlices(doc1TreeArray, doc2TreeArray, pathDoc1, pathDoc2)
	} else if isDoc1Map && isDoc2Map {
		//if both documents are dictionaries compare their properties
		return jsc.CompareProperties(doc1TreeMap, doc2TreeMap, pathDoc1, pathDoc2)
	} else if isDoc1Map && !isDoc2Map {
		//if first is dictionary add dependent object
		jsc.AddDependentObjects(doc1TreeMap, jsc.DepDoc1, pathDoc1)
	} else if !isDoc1Map && isDoc2Map {
		//if second is dictionary add dependent object
		jsc.AddDependentObjects(doc2TreeMap, jsc.DepDoc2, pathDoc2)
	} else if !isDoc1Array && !isDoc1Map && !isDoc2Array && !isDoc2Map {
		//if values are not maps or arrays compare them by default
		//fmt.Printf("keys: %v %v\n", *doc1, *doc2)
		if *doc1 != *doc2 {
			return pathDoc1 /*+ "+"*/, pathDoc2 /*+ "+"*/ /*+ fmt.Sprintf("%s", (*doc1), (*doc2))*/, false
		}
	} else {
		//types of elements are different
		return pathDoc1, pathDoc2, false;
	}

	return pathDoc1, pathDoc2, true;
}

func (jsc * JsonStructureCompare) Compare(doc1TreeMap map[string]interface{}, doc2TreeMap map[string]interface{}, path string) {
	defer timeTrack(time.Now(), "Compare" + path)
	jsc.CompareProperties(doc1TreeMap, doc2TreeMap, path, path)
}

func NewJsonStructureCompare() *JsonStructureCompare {
	return &JsonStructureCompare{make(map[string]interface{}),
				     make(map[string]interface{}),
						make(map[string]interface{}),
						make(map[string]interface{}),
						"do_objectID",
							&DependentObjects{make(map[string]interface{}), make(map[string]interface{})},
							&DependentObjects{make(map[string]interface{}), make(map[string]interface{})}}
}

func Test(doc1File string, doc2File string) (map[string]interface{}, map[string]interface{}) {

	fileDoc1, eDoc1 := ioutil.ReadFile(doc1File)
	if eDoc1 != nil {
		fmt.Printf("Doc1 File error: %v\n", eDoc1)
		os.Exit(1)
	}

	fileDoc2, eDoc2 := ioutil.ReadFile(doc2File)
	if eDoc2 != nil {
		fmt.Printf("Doc2 File error: %v\n", eDoc2)
		os.Exit(1)
	}

	var result1 map[string]interface{}
	var decoder1 = json.NewDecoder(bytes.NewReader(fileDoc1))
	decoder1.UseNumber()

	if err := decoder1.Decode(&result1); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	//mergeInfo, _ := json.MarshalIndent(result1, "", "  ")

	//fmt.Println(string(mergeInfo))


	var result2 map[string]interface{}
	var decoder2 = json.NewDecoder(bytes.NewReader(fileDoc2))
	decoder2.UseNumber()

	if err := decoder2.Decode(&result2); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	//mergeInfo2, _ := json.MarshalIndent(result2, "", "  ")

	//fmt.Println(string(mergeInfo2))

	jsCompare := NewJsonStructureCompare()
	jsCompare.Compare(result1, result2, "$")

	return result1, result2
}

func testCLI() {

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("    sketchvcs file1 file2 ...\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}


	baseFileStruct, newFileStruct := ExtractSketchDirStruct(flag.Arg(0), flag.Arg(1))


	fsMerge := new(FileStructureMerge)
	fsMerge.FileSetChange(baseFileStruct, newFileStruct)

	for _, fm := range fsMerge.MergeActions {
		fileExt := filepath.Ext(fm.FileKey)
		if fileExt == "json" {

		}
	}

	mergeInfo, _ := json.MarshalIndent(fsMerge, "", "  ")

	fmt.Println(string(mergeInfo))


	Test(flag.Arg(0) + "/document.json"/*"/pages/0651DBAC-C79E-4619-AFC3-A8B80B93E01D.json"*/, flag.Arg(1) + "/document.json"/*"/pages/0651DBAC-C79E-4619-AFC3-A8B80B93E01D.json"*/)

}

