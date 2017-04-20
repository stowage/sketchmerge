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

	sd.Description["action"] = append(sd.Description["action"].([]string), action)
	sd.Description["nice_description_short"] = append(sd.Description["nice_description_short"].([]string), niceDescShortText)
	sd.Description["nice_description"] = append(sd.Description["nice_description"].([]string), niceDescText)*/

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
	_=strAction
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

	if _, err := os.Stat(doc1File); os.IsNotExist(err) {
		return jsCompare, nil
	}

	result1, err1 := readJSON(doc1File)

	if err1 != nil {
		return nil, err1
	}

	if _, err := os.Stat(doc2File); os.IsNotExist(err) {
		return jsCompare, nil
	}

	result2, err2 := readJSON(doc2File)

	if err2 != nil {
		return nil, err2
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
	if page != nil {
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
	if artboard != nil {
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

//Builds nice json for pages only
func ProduceNiceDiff(fileName string, doc1 map[string]interface{}, doc2 map[string]interface{}, diff map[string]interface{}, depPaths map[string]interface{} ) map[string]interface{}  {

	if diff==nil {
		return nil
	}

	niceDiff := make(map[string]interface{})

	//Build difference maps
	skDiff := SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), Description: make(map[string]interface{})}}

	//Go thru all differences jsonpath's in actual file
	for key, item := range diff {
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
		if item == "" && srcact == ValueDelete {
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
					if lname == nil || lid == nil {
						return true
					}

					//if artboard is recognized
					if layer["_class"] == "artboard" {
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

		mathingDiffs := make(map[string]interface{})

		diff.SetDifference(srcact, skDiff, key, item.(string))

		findMatchingDiffs(fileName, key, depPaths, mathingDiffs)

		for mKey, mItem := range mathingDiffs {
			diff.SetDifference(srcact, skDiff, mKey, mItem.(string))
		}

	}

	if len(diff) > 0 {
		niceDiff["nice_diff"] = skDiff
	}

	return niceDiff

}

func WriteToFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0755 )
}

func findMatchingDiffs(fileName string, matchingKey string, depPaths map[string]interface{}, diffs map[string]interface{}) {

	for key, item := range depPaths {

		flatKey := FlatJsonPath(key, true)
		flatMatch := FlatJsonPath(matchingKey, true)
		if flatKey == flatMatch || strings.HasPrefix(flatKey, flatMatch) {

			if strings.HasPrefix(matchingKey, "^") {
				continue
			}

			paths, isPaths := item.([]interface{})
			if isPaths {
				for i := range paths {
					newKey := paths[i].(DependentObj).JsonPath
					fileKey := ReadFileAction(key)
					fileNewKey := ReadFileAction(newKey)

					log.Printf("fileName: %v -> %v\n", fileName, ReadFileKey(newKey) )

					//if it reffers to actual file just ignore
					if fileName == ReadFileKey(newKey) {
						continue
					}

					//if key refers to other file append to action which belong to this file
					if fileKey != "" && fileNewKey == "" {
						newKey = fileKey + newKey
					}

					//if there is similar element just ignore it
					if diffs[newKey] != nil {
						continue
					}

					//store new jsonpath pair
					diffs[newKey] = paths[i].(DependentObj).Ref //+ " <- " + key

					//Look up dependencies recursively for newKey
					findMatchingDiffs(fileName, newKey, depPaths, diffs)
				}
			}
		}
	}

}

//Convert dependent objects to depencies jsonpaths
func addDependencies(fileKey string, depObj * DependentObjects, docDep * DependentObjects, fileMap map[string]interface{}, stopFileKey map[string]bool) (*DependentObjects) {
	if docDep == nil {
		docDep = &DependentObjects{make(map[string]interface{}),make(map[string]interface{})}
	}
	for key, value := range docDep.DepObj {

		iPaths := depObj.DepObj[key]

		if iPaths == nil {
			continue
		}
		depPaths := iPaths.([]interface{})
		paths := value.([]interface{})
		//Loop thru all dependencies build by buildDependencePaths method
		for k := range paths {

			//Go thru all dependencies in actual file
			for j := range depPaths {

				//Add dependencies if it reffers to other file
				if depPaths[j].(DependentObj).FileKey != fileKey {

					docDep.AddDependentPath(paths[k].(DependentObj).JsonPath, depPaths[j].(DependentObj).Ref, depPaths[j].(DependentObj).JsonPath)

					fileMerge, isFileMerge := fileMap[depPaths[j].(DependentObj).FileKey].(FileMerge)

					if isFileMerge && fileMerge.FileDiff.DepDoc1 != nil {

						//Avoid endless recursions by keeping all files keys in map
						if !stopFileKey[fileMerge.FileKey] {

							//Find dependencies recursively
							docSubDep := &DependentObjects{fileMerge.FileDiff.DepDoc1.DepObj, make(map[string]interface{})}
							stopFileKey[fileKey] = true
							subDep := addDependencies(depPaths[j].(DependentObj).FileKey, depObj, docSubDep, fileMap, stopFileKey)

							//Add jsonpaths to current dependency
							for subKey, subPath := range subDep.DepPath {
								subDepPath, isPath := subPath.([]interface{})
								if isPath {
									for i := range subDepPath {
										var newFileKey = "~" + fileMerge.FileKey + fileMerge.FileExt + "~" + subKey
										if strings.HasPrefix(subKey, "~") {
											newFileKey = subKey
										}
										docDep.AddDependentPath(newFileKey, subDepPath[i].(DependentObj).Ref, subDepPath[i].(DependentObj).JsonPath)
									}
								}
							}
						}

					}

				} else {
					//Add dependencies if it reffers to actual file
					ref := FlatJsonPath(depPaths[j].(DependentObj).Ref, false)
					jsonpath := FlatJsonPath(depPaths[j].(DependentObj).JsonPath, false)
					if jsonpath != "" {
						docDep.AddDependentPath(paths[k].(DependentObj).JsonPath, ref, jsonpath)
					}
				}
			}

		}
	}
	return docDep
}

func ProceedDependencies(workingDirV1 string, workingDirV2 string, fileMerge []FileMerge ) error {

	depObj := DependentObjects{make(map[string]interface{}), make(map[string]interface{})}

	fileMap, err := depObj.buildDependencePaths(workingDirV1, workingDirV2, fileMerge)
	if err!=nil {
		return err
	}

	info, _ := json.MarshalIndent(depObj, "", "  ")
	fmt.Printf("%v\n", string(info))

	_=fileMap
	for i := range fileMerge {
		fileMerge[i].FileDiff.DepDoc1 = addDependencies(fileMerge[i].FileKey, &depObj, fileMerge[i].FileDiff.DepDoc1, fileMap, make(map[string]bool))
	}
	return nil;
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
		if filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)) == ".json" {
			result, err := CompareJSON(workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt,  workingDirV2 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)
			if err != nil {
				return nil, err
			}
			fsMerge.MergeActions[i].FileDiff = *result


		}
	}

	if err := ProceedDependencies(workingDirV1, workingDirV2, fsMerge.MergeActions); err!=nil {
		return nil, err
	}

	if !isNice {
		mergeInfo, _ := json.MarshalIndent(fsMerge, "", "  ")

		return mergeInfo, nil
	} else {
		/*for i := range fsMerge.MergeActions {
			if filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)) == ".json" {
				result, err := CompareJSONNice(workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt,  workingDirV2 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)
				if err != nil {
					return nil, err
				}
				fsMerge.MergeActions[i].FileDiff = *result


			}
		}*/



		for i := range fsMerge.MergeActions {
			if filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)) == ".json" {
				doc1File := workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
				doc2File := workingDirV2 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt

				if _, err := os.Stat(doc1File); os.IsNotExist(err) {
					continue
				}

				result1, err1 := readJSON(doc1File)

				if err1 != nil {
					return nil, err1
				}

				if _, err := os.Stat(doc2File); os.IsNotExist(err) {
					continue
				}

				result2, err2 := readJSON(doc2File)

				if err2 != nil {
					return nil, err2
				}

				jsCompare := fsMerge.MergeActions[i].FileDiff

				fileName := fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
				fsMerge.MergeActions[i].FileDiff.Doc1Diffs = ProduceNiceDiff(fileName, result1, result2, jsCompare.Doc1Diffs, jsCompare.DepDoc1.DepPath)
				fsMerge.MergeActions[i].FileDiff.Doc2Diffs = ProduceNiceDiff(fileName, result2, result1, jsCompare.Doc2Diffs, jsCompare.DepDoc2.DepPath)
			}
		}


		mergeInfo, _ := json.MarshalIndent(fsMerge, "", "  ")

		return mergeInfo, nil
	}
}



