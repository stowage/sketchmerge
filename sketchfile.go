package sketchmerge

import (
	"os"
	"os/user"
	_"fmt"
	"log"
	"github.com/twinj/uuid"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"path/filepath"
	"strings"
	_"io"
	"fmt"
	"time"
	"path"
)

type SketchLayerInfo struct {
	LayerName string
	LayerID string
	ArtboardName string
	ArtboardID string
	PageName string
	PageID string
	NiceDescriptionShort string
	NiceDescription string
}

type Difference interface {
	SetDiff(action ApplyAction, src string, dst string, niceDescShort string, niceDesc string)
}

type MainDiff struct {
	Description map[string]interface{} `json:"description,omitempty"`
	Diff map[string]interface{} `json:"diff,omitempty"`
	Difference `json:"-"`
	
}

type SketchLayerDiff struct {
	Name string `json:"name,omitempty"`
	MainDiff
}

type SketchArtboardDiff struct {
	Name string `json:"name,omitempty"`
	LayerDiff map[string]interface{} `json:"layer_diff,omitempty"`
	MainDiff
}

type SketchPageDiff struct {
	Name string `json:"name,omitempty"`
	ArtboardDiff map[string]interface{} `json:"artboard_diff,omitempty"`
	MainDiff
}

type SketchDiff struct {
	PageDiff map[string]interface{} `json:"page_diff,omitempty"`
	MainDiff
}


func (sd* MainDiff) SetDiff(action ApplyAction, src string, dst string, niceDescShortText string, niceDescText string) {
	/*actions := sd.Description["action"]

	if actions == nil {
		sd.Description["action"] = make([]string, 0)
	}

	niceDescShort := sd.Description["nice_description_short"]

	if niceDescShort == nil {
		sd.Description["nice_description_short"] = make([]string, 0)
	}

	niceDesc := sd.Description["nice_description"]

	if niceDesc == nil {
		sd.Description["nice_description"] = make([]string, 0)
	}


	strAction := ""

	switch action {
	case ValueAdd:
		strAction = "ValueAdd"
	case ValueChange:
		strAction = "ValueChange"
	case ValueDelete:
		strAction = "ValueDelete"
	case SequenceChange:
		strAction = "SequenceChange"
	}

	sd.Description["action"] = append(sd.Description["action"].([]string), strAction)
	sd.Description["nice_description_short"] = append(sd.Description["nice_description_short"].([]string), niceDescShortText)
	sd.Description["nice_description"] = append(sd.Description["nice_description"].([]string), niceDescText)
	*/

	sd.Diff[src] = dst


}

func prepareWorkingDir(hasToCreate bool) (string, error) {
	usr, err := user.Current()
	if err != nil {
		log.Fatal( err )
		return "", err
	}
	workingDir := usr.HomeDir + string(os.PathSeparator) + ".versions" + string(os.PathSeparator) + uuid.NewV4().String()
	var errmk error

	if hasToCreate {
		errmk = os.MkdirAll(workingDir, os.ModePerm)
	}
	return workingDir, errmk
}

func removeWorkingDir(workingDir string, hasToSimulate bool)  {
	if !hasToSimulate {
		os.RemoveAll(workingDir)
	}
}

func readJSON(docFile string) (map[string]interface{}, error) {
	fileDoc1, eDoc1 := ioutil.ReadFile(docFile)
	if eDoc1 != nil {
		return nil, eDoc1
	}

	var result1 map[string]interface{}
	var decoder1 = json.NewDecoder(bytes.NewReader(fileDoc1))
	decoder1.UseNumber()

	if err := decoder1.Decode(&result1); err != nil {
		return nil, err
	}

	return result1, nil
}



func CompareJSON(doc1File string, doc2File string) (*JsonStructureCompare, error) {

	jsCompare := NewJsonStructureCompare()

	var result1 map[string]interface{}
	var result2 map[string]interface{}

	if _, err := os.Stat(doc1File); os.IsNotExist(err) {
		result1 = make(map[string]interface{})
		//return jsCompare, nil
	} else {
		var err1 error
		result1, err1 = readJSON(doc1File)

		if err1 != nil {
			return nil, err1
		}
	}

	if _, err := os.Stat(doc2File); os.IsNotExist(err) {
		result2 = make(map[string]interface{})
		//return jsCompare, nil
	} else {
		var err2 error
		result2, err2 = readJSON(doc2File)

		if err2 != nil {
			return nil, err2
		}
	}


	jsCompare.Compare(result1, result2, "$")



	return jsCompare, nil
}



//This method is part of nice json process
func (li * SketchLayerInfo) SetDifference(action ApplyAction, diff SketchDiff, diffSrc string, diffDst string) {

	var page interface{}
	var artboard interface{}
	var layer interface{}

	//Actual difference could be Page, Artboard or Layer
	//set it to existing difference by default
	var actual Difference = &diff

	//if PageID is recognized
	if li.PageID != "" {
		page = diff.PageDiff[li.PageID]

		//if page is not exists then create it
		if page == nil {
			page = SketchPageDiff{Name: li.PageName, ArtboardDiff: make(map[string]interface{}), MainDiff: MainDiff{ Diff: make(map[string]interface{}), Description: make(map[string]interface{})}}
			diff.PageDiff[li.PageID] = page
		}
		_page := page.(SketchPageDiff)

		//set actual difference is Page
		actual = &_page
	}

	//only if we are inside page try to recognize artboard
	if page != nil && li.ArtboardID != "" {
		artboard = page.(SketchPageDiff).ArtboardDiff[li.ArtboardID]

		if artboard == nil {
			artboard = SketchArtboardDiff{Name: li.ArtboardName,  LayerDiff: make(map[string]interface{}), MainDiff:MainDiff{ Diff: make(map[string]interface{}), Description: make(map[string]interface{})} }
			page.(SketchPageDiff).ArtboardDiff[li.ArtboardID] = artboard
		}
		_artboard := artboard.(SketchArtboardDiff)

		//set actual differnce to artboard
		actual = &_artboard
	}

	//if it is artboard
	if artboard != nil && li.LayerID != "" {
		layer = artboard.(SketchArtboardDiff).LayerDiff[li.LayerID]

		if layer == nil {
			layer = SketchLayerDiff{Name: li.LayerName, MainDiff:MainDiff{ Diff: make(map[string]interface{}), Description: make(map[string]interface{})}}
			artboard.(SketchArtboardDiff).LayerDiff[li.LayerID] = layer
		}
		_layer := layer.(SketchLayerDiff)

		actual = &_layer
	}


	actual.SetDiff(action, diffSrc, diffDst, li.NiceDescriptionShort, li.NiceDescription)

}

func ReadKeyValue(doc1 map[string]interface{}, doc2 map[string]interface{}, key string) (SketchLayerInfo, ApplyAction) {
	var pageID = ""
	var pageName = ""

	var artboardID = ""
	var artboardName = ""

	var niceDesc = ""
	var niceDescShort = ""

	var layerID = ""
	var layerName string = ""
	var layerPath string = ""

	//Parse jsonpath in key
	srcSel, srcact, _ := Parse(key)
	doc := doc1

	//Delete action usually belongs to destination document which is doc2
	if /*item == "" &&*/ srcact == ValueDelete {
		doc = doc2
	}

	//Walk thru parsed path key
	_, lastNode, err := srcSel.ApplyWithEvent(doc, func(v interface{}, prevNode Node, node Node) bool {

		if prevNode == nil {
			//This is root element which is page
			layer := v.(map[string]interface{})
			if layer != nil {
				lname := layer["name"]
				lid := layer["do_objectID"]
				if lname == nil || lid == nil {
					return true
				}

				pageName = lname.(string)
				pageID = lid.(string)
				layerPath = pageName

			}
		} else if prevNode.GetKey() == "layers" {
			//This is any layer inside page
			layer := v.(map[string]interface{})
			if layer != nil {
				lname := layer["name"]
				lid := layer["do_objectID"]
				//sid := layer["symbolID"]
				if lname == nil || lid == nil {
					return true
				}

				//if artboard is recognized
				if layer["_class"] == "symbolMaster" {
					artboardName = lname.(string)
					artboardID = lid.(string)
					//artboardID = sid.(string)
					layerPath += "/" + artboardName
				} else if layer["_class"] == "artboard" {
					artboardName = lname.(string)
					artboardID = lid.(string)
					layerPath += "/" + artboardName
				} else  {
					layerName = lname.(string)
					layerID = lid.(string)
					layerPath += "/" + layerName
				}
			}

		}
		return true;
	})

	if err!=nil {
		log.Printf("Error occurired while building nice diff: %v", err)
	}

	if pageID != "" && artboardID != "" && layerID != "" {
		niceDescShort, niceDesc = GetNiceTextForLayer(srcact, layerName, pageName, artboardName, layerPath)
	} else if layerID != "" {
		niceDescShort, niceDesc = GetNiceTextForUnknownLayer(srcact, layerName, layerPath)
	} else if artboardID != "" {
		niceDescShort, niceDesc = GetNiceTextForArtboard(srcact, artboardName, pageName)
	} else if pageID != "" {
		niceDescShort, niceDesc = GetNiceTextForPage(srcact, pageName)
	} else {
		niceDescShort, niceDesc = GetNiceTextForUnknown(srcact, fmt.Sprintf("%v", lastNode.GetKey()))
	}

	diff := SketchLayerInfo{layerName, layerID,
				artboardName, artboardID,
				pageName, pageID,
				niceDescShort, niceDesc}

	return diff, srcact
}


//Builds nice json for pages only
func ProduceNiceDiff(fileAction FileActionType, fileName string, doc1 map[string]interface{}, doc2 map[string]interface{}, diff map[string]interface{}, depPaths1 map[string]interface{}, depPaths2 map[string]interface{}) map[string]interface{}  {

	defer TimeTrack(time.Now(), "ProduceNiceDiff " + fileName)
	if diff==nil {
		return nil
	}

	niceDiff := make(map[string]interface{})

	//Build difference maps
	skDiff := SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), Description: make(map[string]interface{})}}

	//Go thru all differences jsonpath's in actual file
	for key, item := range diff {

		diff, srcact := ReadKeyValue(doc1, doc2, key)

		diff.SetDifference(srcact, skDiff, key, item.(string))

		mathingDiffs := make(map[string]interface{})

		FindMatchingDiffs(SOURCE, fileName, key, depPaths1, depPaths2, mathingDiffs)

		for mKey, mItem := range mathingDiffs {
			diff.SetDifference(srcact, skDiff, mKey, mItem.(string))
		}

		if fileAction == ADD || fileAction == DELETE {
			fileActionPath := BuildFileAction(fileAction, fileName)
			diff.SetDifference(srcact, skDiff, fileActionPath, fileActionPath)
			diff.SetDifference(srcact, skDiff, "~document.json~$[\"pages\"]", "~document.json~$[\"pages\"]")
		}

		log.Printf("action: %v %v", key, item)
	}

	if len(diff) > 0 {
		niceDiff["nice_diff"] = skDiff
	}

	return niceDiff

}

func WriteToFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0755 )
}


func ProcessFileDiff(sketchFileV1 string, sketchFileV2 string, isNice bool) ([]byte, error) {

	isSrcDir := false
	isDstDir := false

	sketchFileV1Info, errv1 := os.Stat(sketchFileV1)

	if errv1 != nil {
		return nil, errv1
	}

	isSrcDir = sketchFileV1Info.IsDir()

	sketchFileV2Info, errv2 := os.Stat(sketchFileV2)

	if errv2 != nil {
		return nil, errv2
	}

	isDstDir = sketchFileV2Info.IsDir()

	workingDirV1, err1 := prepareWorkingDir(!isSrcDir)
	if err1!=nil {
		return nil, err1
	}
	defer removeWorkingDir(workingDirV1, isSrcDir)

	if isSrcDir {
		workingDirV1 = sketchFileV1
	}

	workingDirV2, err2 := prepareWorkingDir(!isDstDir)
	if  err2!=nil {
		return nil, err2
	}
	defer removeWorkingDir(workingDirV2, isDstDir)

	if isDstDir {
		workingDirV2 = sketchFileV2
	}

	if !isSrcDir {
		if err := Unzip(sketchFileV1, workingDirV1); err != nil {
			return nil, err
		}
	}

	if !isDstDir {
		if err := Unzip(sketchFileV2, workingDirV2); err != nil {
			return nil, err
		}
	}

	baseFileStruct, newFileStruct := ExtractSketchDirStruct(workingDirV1, workingDirV2)


	fsMerge := new(FileStructureMerge)
	fsMerge.FileSetChange(newFileStruct, baseFileStruct)

	for i := range fsMerge.MergeActions {
		fileName := fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
		if filepath.Ext(strings.ToLower(fileName)) == ".json" {
			result, err := CompareJSON(workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt,  workingDirV2 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)
			if err != nil {
				return nil, err
			}
			fsMerge.MergeActions[i].FileDiff = *result

			if fsMerge.MergeActions[i].Action != MERGE {
				result.FileDependendObject(fsMerge.MergeActions[i].Action, SOURCE, fsMerge.MergeActions[i].FileKey, fileName)
				result.FileDependendObject(fsMerge.MergeActions[i].Action, DESTINATION, fsMerge.MergeActions[i].FileKey, fileName)
			}
		}
	}


	if _, _, err := ProceedDependencies(workingDirV1, workingDirV2, fsMerge.MergeActions); err!=nil {
		return nil, err
	}

	if !isNice {

		for i := range fsMerge.MergeActions {
			fileName := fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
			for key, _ := range fsMerge.MergeActions[i].FileDiff.Doc1Diffs {
				matchingDiffs := make(map[string]interface{})

				FindMatchingDiffs(SOURCE, fileName, key, fsMerge.MergeActions[i].FileDiff.DepDoc1.DepPath, fsMerge.MergeActions[i].FileDiff.DepDoc2.DepPath, matchingDiffs)

				for mKey, mItem := range matchingDiffs {
					fsMerge.MergeActions[i].FileDiff.Doc1Diffs[mKey] = mItem
				}

			}
			if fsMerge.MergeActions[i].FileDiff.Doc1Diffs == nil {
				continue
			}
			fileAction := fsMerge.MergeActions[i].Action


			if fileAction == ADD || fileAction == DELETE {
				fileActionPath := BuildFileAction(fileAction, fileName)
				fsMerge.MergeActions[i].FileDiff.Doc1Diffs[fileActionPath] = fileActionPath
				fsMerge.MergeActions[i].FileDiff.Doc1Diffs["~document.json~$[\"pages\"]"] = "~document.json~$[\"pages\"]"
			}

		}
		mergeInfo, _ := json.MarshalIndent(fsMerge, "", "  ")

		return mergeInfo, nil
	} else {

		for i := range fsMerge.MergeActions {
			if filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)) == ".json" {
				doc1File := workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
				doc2File := workingDirV2 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt

				var result1 map[string]interface{}
				var result2 map[string]interface{}

				if _, err := os.Stat(doc1File); os.IsNotExist(err) {
					result1 = make(map[string]interface{})
				} else {
					var err1 error
					result1, err1 = readJSON(doc1File)

					if err1 != nil {
						return nil, err1
					}
				}


				if _, err := os.Stat(doc2File); os.IsNotExist(err) {
					result2 = make(map[string]interface{})
				} else {
					var err2 error
					result2, err2 = readJSON(doc2File)

					if err2 != nil {
						return nil, err2
					}
				}

				jsCompare := fsMerge.MergeActions[i].FileDiff

				fileName := fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
				fileAction := fsMerge.MergeActions[i].Action
				fsMerge.MergeActions[i].FileDiff.Doc1Diffs = ProduceNiceDiff(fileAction, fileName, result1, result2, jsCompare.Doc1Diffs, jsCompare.DepDoc1.DepPath, jsCompare.DepDoc2.DepPath)

				if fileAction == ADD {
					fileAction = DELETE
				} else if fileAction == DELETE {
					fileAction = ADD
				}
				fsMerge.MergeActions[i].FileDiff.Doc2Diffs = ProduceNiceDiff(fileAction, fileName, result2, result1, jsCompare.Doc2Diffs, jsCompare.DepDoc2.DepPath, jsCompare.DepDoc1.DepPath)
			} else {
				fsMerge.MergeActions[i].Action = ADD
			}
		}


		mergeInfo, _ := json.MarshalIndent(fsMerge, "", "  ")

		return mergeInfo, nil
	}
}

//Performs 2-way merge using docDiff json paths
func merge(workingDirV1 string, workingDirV2 string, fileName string, objectKeyName string, docDiffs map[string]interface{} ) error {

	srcFilePath := workingDirV1 + string(os.PathSeparator) + fileName
	dstFilePath := workingDirV2 + string(os.PathSeparator) + fileName

	//get files jsons
	jsonDoc1, jsonDoc2, err := decodeMergeFiles(srcFilePath, dstFilePath)

	if err != nil {
		return err
	}

	//Create merge documets structure
	mergeDoc := MergeDocuments{jsonDoc1, jsonDoc2}

	//We will perform delete operations after isertions to avoid
	//actions on the same index
	deleteActions := make(map[string]string)

	//All sequence changes should be performed after all changes
	seqDiff := make(map[string]string)

	//Sorting diffs by deepness of item location
	//this is required if we are adding subelemnts, because we can not add sub element without adding
	//because we can not add sub element without adding parent
	sortedActions := GetSortedDiffs(docDiffs, fileName)

	for i := range sortedActions {

		dep := sortedActions[i].(DependentObj)
		key := dep.JsonPath
		var item interface{} = dep.Ref

		//if item is empty string its a delete operation
		if item == "" {
			deleteActions[key] = ""
		} else if strings.HasPrefix(key, "^") {
			seqDiff[key] = item.(string)
		} else {
			//Merge changes of values first
			mergeDoc.MergeByJSONPath(key, item.(string), DeleteMarked)
		}
	}

	//Perform all deletions
	//First iteration will only mark to delete
	for key, _ := range deleteActions {
		mergeDoc.MergeByJSONPath("", key, MarkElementToDelete)
	}

	//second iteration will delete
	//TODO: optimize second call
	for key, _ := range deleteActions {
		mergeDoc.MergeByJSONPath("", key, DeleteMarked)
	}

	//Perform sorting
	for key, item := range seqDiff {
		mergeDoc.MergeSequenceByJSONPath(objectKeyName, key, item)
	}

	//Marshal result
	data, err := json.Marshal(mergeDoc.DstDocument)

	if err != nil {
		return err
	}

	//to final sketch file
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

func createMergeAction(fileKey, fileAction string) FileMerge {

	fileExt := filepath.Ext(fileKey)
	fileName := strings.TrimSuffix(fileKey, fileExt)

	mergeAction := FileMerge{ FileExt: fileExt, FileKey: fileName, IsDirectory: false, FileDiff: CreateJsonStructureCompare()}

	if strings.HasPrefix(fileAction, "A") {
		mergeAction.Action = ADD
	} else if strings.HasPrefix(fileAction, "D") {
		mergeAction.Action = DELETE
	}

	return mergeAction
}

//if strings.HasPrefix(key, "R") {
//
//newKey, newItem := ReversAction( key, item.(string))
//
//key = newKey
//item = newItem
//delete(mergeJSON.MergeActions[i].FileDiff.Doc1Diffs, originalKey)
//mergeJSON.MergeActions[i].FileDiff.Doc1Diffs[key] = item.(string)
//originalKey = key
//
//continue
//}

func buildFileActions(workingDirV1 string, workingDirV2 string, mergeJSON FileStructureMerge) ([]FileMerge) {

	newMergeActions := make([]FileMerge, 0)

	mergeMap := make(map[string]interface{})

	for i := range mergeJSON.MergeActions {


		for key, item := range mergeJSON.MergeActions[i].FileDiff.Doc1Diffs {

			if strings.HasPrefix(key, "R") {
				//newKey, newItem := ReversAction( key, item.(string))
				//
				//key = newKey
				//item = newItem
				//
				//if strings.HasPrefix(key,"D") {
				//	continue
				//}
				continue
			}

			fileName := mergeJSON.MergeActions[i].FileKey + mergeJSON.MergeActions[i].FileExt
			fileKey := ReadFileKey(key)
			fileAction := ReadFileAction(key)

			//fmt.Println("key: " + fileName + " " + fileKey)
			if fileKey != "" {
				fileName = fileKey
			}


			fileMerge, ok := mergeMap[fileName].(FileMerge)
			if !ok {


				fileMerge = createMergeAction(fileName, fileAction)
				mergeMap[fileName] = fileMerge

			}

			if strings.HasPrefix(key,"A") {
				fileMerge.Action = ADD
				mergeMap[fileName] = fileMerge
			} else if strings.HasPrefix(key,"D") {
				fileMerge.Action = DELETE
				mergeMap[fileName] = fileMerge
			}

			fileMerge.FileDiff.Doc1Diffs[FlatJsonPath(key, false)] = FlatJsonPath(item.(string), false)
		}


	}

	for _, value := range mergeMap {
		newMergeActions = append(newMergeActions, value.(FileMerge))
	}

	return newMergeActions
}

func mergeActions(workingDirV1 string, workingDirV2 string, mergeJSON FileStructureMerge) error {


	mergeActions := buildFileActions(workingDirV1, workingDirV2, mergeJSON)

	info1, _ := json.MarshalIndent(mergeActions, "", "  ")
	fmt.Printf("%v\n", string(info1))

	//For debug purpose
	//for i := range mergeActions {
	//	if filepath.Ext(strings.ToLower(mergeActions[i].FileKey+mergeActions[i].FileExt)) == ".json" {
	//		fileName := mergeActions[i].FileKey + mergeActions[i].FileExt
	//
	//		srcFilePath := workingDirV1 + string(os.PathSeparator) + fileName
	//		dstFilePath := workingDirV2 + string(os.PathSeparator) + fileName
	//		result1, result2, _ := decodeMergeFiles(srcFilePath, dstFilePath)
	//
	//		fileAction := mergeActions[i].Action
	//		mergeActions[i].FileDiff.Doc1Diffs = ProduceNiceDiff(fileAction, fileName, result2, result1, mergeActions[i].FileDiff.Doc1Diffs , make(map[string]interface{}), make(map[string]interface{}))
	//
	//		info1, _ := json.MarshalIndent(mergeActions, "", "  ")
	//		fmt.Printf("%v\n", string(info1))
	//	}
	//}
	//

	for i := range mergeActions {

		fileName := mergeActions[i].FileKey + mergeActions[i].FileExt

		if !mergeActions[i].IsDirectory && mergeActions[i].Action == ADD {
			updateFile(workingDirV1, workingDirV2, fileName)
		}

		if mergeActions[i].FileDiff.Doc1Diffs == nil {
			continue
		}

		if err := merge(workingDirV1, workingDirV2,
			fileName,
			mergeActions[i].FileDiff.ObjectKeyName,
			mergeActions[i].FileDiff.Doc1Diffs); err!=nil {
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

