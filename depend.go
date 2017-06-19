// Copyright 2017 Sergey Fedoseev. All rights reserved.
// This module contains functions helping to find dependencies

package sketchmerge

import (
	"regexp"
	"strings"
	"log"
	"time"
	"path/filepath"
	"os"
	_"encoding/json"
	_"fmt"
)
const GuidFormat = "^[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$"

const RefImagesFormat = "^images"  + string(os.PathSeparator) + "[a-z0-9]{40}$"

const RefPagesFormat = "^pages" + string(os.PathSeparator) +"[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$"

var formats = []*regexp.Regexp { BuildReg(GuidFormat), BuildReg(RefImagesFormat), BuildReg(RefPagesFormat) }


func BuildReg(regstr string) (*regexp.Regexp) {

	reg, err := regexp.Compile(regstr)
	if err != nil {
		log.Printf("Error in regexp: %v %v\n", regstr, err)
	}
	return reg
}

const  (
	SOURCE = iota
	DESTINATION
)

type DocumentType uint8

type DependentObj struct {
	JsonPath string `json:"path,omitempty"`
	Ref string `json:"ref,omitempty"`
	FileKey string `json:"file_key,omitempty"`
}

type DependentMerge struct {
	JsonPathSrc string `json:"path_src,omitempty"`
	JsonPathDst string `json:"path_dst,omitempty"`
	Ref string `json:"ref,omitempty"`
}

type DependentObjects struct {
	docType DocumentType
	DepObj map[string]interface{} `json:"dep_obj,omitempty"`
	DepPath map[string]interface{} `json:"dep_path,omitempty"`
}

func _timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)

}

func DetectID(value interface{}) bool {

	switch valueType := value.(type) {
	default:
		_=valueType
		return false
	case string:
		for index := range formats {
			reg := formats[index]

			if reg.MatchString(strings.ToLower(value.(string))) {
				return true
			}
		}
	}
	return false
}

func IsSketchID(value interface{}) bool {
	//defer _timeTrack(time.Now(), "IsSketchID")
	if value == nil {
		return false
	}
	return DetectID(value)
}

func (dep* DependentObjects) AddDependentObject(key string, value interface{}, jsonpath string) bool {

	var depKey string
	var depMap map[string]interface{}

	//In case we need to take GUID values in key elements uncomment this
	//isKeyObjectID := IsSketchID(key)
	isValueObjectID := IsSketchID(value)

	//In case we need to take GUID values in key elements uncomment this
	//if isKeyObjectID {
	//	depKey = key
	//	depMap = dep.DepObj
	//
	//} else
	if isValueObjectID {
		depKey = value.(string)
		depMap = dep.DepObj

	} else {
		depKey = jsonpath
		jsonpath = ""
		depMap = dep.DepPath

	}

	if strings.HasPrefix(depKey, "pages" + string(os.PathSeparator)) {
		depKey = strings.TrimPrefix(depKey, "pages" + string(os.PathSeparator))
	}

	//In case we need to take GUID values in key elements uncomment this
	//if isKeyObjectID {
	//	depItem := depMap[depKey]
	//	if depItem == nil {
	//		depItem = make([]interface{}, 0)
	//	}
	//	depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath})
	//}

	if isValueObjectID {
		depItem := depMap[depKey]
		if depItem == nil {
			depItem = make([]interface{}, 0)
		}
		depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath})

		return true
	}

	return false

}

func (dep* DependentObjects) AddDependent(depKey string, jsonpath1 string, jsonpath2 string, fileKey string)  {

	depMap := dep.DepObj

	depItem := depMap[depKey]
	if depItem == nil {
		depItem = make([]interface{}, 0)
	}
	depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath2, Ref:jsonpath1, FileKey:fileKey})


}


func (dep* DependentObjects) AddDependentPath(key string, value string, jsonpath string)  {


	var depKey string = key
	var depMap map[string]interface{} = dep.DepPath

	depItem := depMap[depKey]
	if depItem == nil {
		depItem = make([]interface{}, 0)
	}
	depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath, Ref:value})



}


//Build dependencies map from jsonpath by finding ids
func (dep* DependentObjects) ResolveDependencies(fileKey string, filepath string, jsonpath1 string, jsonpath2 string, doc map[string]interface{}) error {
	//defer TimeTrack(time.Now(), "ResolveDependencies" + jsonpath1)
	srcSel, _, err1 := Parse(jsonpath1)
	//fmt.Printf("jsonpath: %v\n", jsonpath1)
	if err1 != nil {
		return err1

	}

	if jsonpath2 == "" {
		return nil
	}

	fileJsonPath1 := ""

	if filepath != "" {

		fileJsonPath1 = "~" + filepath + "~" + jsonpath1
	} else {
		fileJsonPath1 = jsonpath1
	}

	fileJsonPath2 := ""

	if filepath != "" {

		fileJsonPath2 = "~" + filepath + "~" + jsonpath2
	} else {
		fileJsonPath2 = jsonpath2
	}

	_, _, err := srcSel.ApplyWithEvent(doc, func(v interface{}, prevNode Node, node Node) bool {
		layer, isLayer := v.(map[string]interface{})
		if isLayer {

			sid := layer["symbolID"]
			if sid != nil {
				dep.AddDependent(sid.(string), fileJsonPath2, fileJsonPath1, fileKey)
			}

			//_ref := layer["_ref"]
			//if _ref != nil {
			//	if IsSketchID(_ref) {
			//		if strings.HasPrefix(_ref.(string), "pages" + string(os.PathSeparator)) {
			//			_ref = strings.TrimPrefix(_ref.(string), "pages" + string(os.PathSeparator))
			//		}
			//		dep.AddDependent(_ref.(string), fileJsonPath2, fileJsonPath1, fileKey)
			//	}
			//}

			lid := layer["do_objectID"]



			var objectID string

			if lid == nil {
				key := node.GetKey()
				if IsSketchID(key) {
					objectID = key.(string)
					dep.AddDependent(objectID, fileJsonPath2, fileJsonPath1, fileKey)
				}
				return true
			}


			//fmt.Printf("do_objId: %v %v\n", filepath, lid)
			objectID = lid.(string)

			dep.AddDependent(objectID, fileJsonPath2, fileJsonPath1, fileKey)


		}

		return true;
	})

	if err != nil {
		return err
	}





	return nil
}



//build dependence map objectID->jsonpaths
func (dep * DependentObjects) buildDependencePaths(docType DocumentType, workingPathV1 string, workingPathV2 string, mergeActions []FileMerge, dep2 * DependentObjects) (map[string]interface{},error) {

	defer TimeTrack(time.Now(), "buildDependencePaths " + workingPathV1)

	fileMap1 := make(map[string]interface{})
	//Go thru all files
	for i := range mergeActions {

		//build files associates for addDependencies method
		fileMap1[mergeActions[i].FileKey] = mergeActions[i]
		//only if it's json file
		if filepath.Ext(strings.ToLower(mergeActions[i].FileKey + mergeActions[i].FileExt)) == ".json" {

			fullFilePath :=  mergeActions[i].FileKey + mergeActions[i].FileExt

			mergeActionCode := mergeActions[i].Action
			fileActionPrefix := ""

			//defer TimeTrack(time.Now(), "buildDependencePaths for each " + fullFilePath)
			if docType == DESTINATION {
				if mergeActionCode == ADD {
					mergeActionCode = DELETE
				} else if mergeActionCode == DELETE {
					mergeActionCode = ADD
				}
			}

			switch mergeActionCode {
			case ADD:
				fileActionPrefix = "A"
			case DELETE:
				fileActionPrefix = "D"
			}

			//if its a new file
			if mergeActionCode != MERGE {
				if strings.HasPrefix(mergeActions[i].FileKey, "pages" + string(os.PathSeparator)) {
					objectID := strings.TrimPrefix(mergeActions[i].FileKey, "pages" + string(os.PathSeparator))
					jsonFilePath := "~" + fullFilePath + "~$"
					dep.AddDependent(objectID, fileActionPrefix+jsonFilePath, fileActionPrefix+jsonFilePath, mergeActions[i].FileKey)

				} else {
					jsonFilePath := fileActionPrefix + "~" + fullFilePath + "~$"
					dep.AddDependent(mergeActions[i].FileKey, jsonFilePath, jsonFilePath, mergeActions[i].FileKey)
				}
			// if we need to delete this file
			} else {

				//defer TimeTrack(time.Now(), "reading buildDependencePaths for each " + fullFilePath)

				docDiffs := mergeActions[i].FileDiff.Doc1Diffs

				if docType == DESTINATION {
					docDiffs = mergeActions[i].FileDiff.Doc2Diffs
				}

				if docDiffs == nil {
					continue
				}

				if _, err := os.Stat(workingPathV1 + string(os.PathSeparator) + fullFilePath); os.IsNotExist(err) {
					continue
				}

				result1, err1 := readJSON(workingPathV1 + string(os.PathSeparator) + fullFilePath)

				if err1 != nil {
					return nil, err1
				}

				if _, err := os.Stat(workingPathV2 + string(os.PathSeparator) + fullFilePath); os.IsNotExist(err) {
					continue
				}

				result2, err2 := readJSON(workingPathV2 + string(os.PathSeparator) + fullFilePath)

				if err2 != nil {
					return nil, err2
				}

				for key, item := range docDiffs {

					if strings.HasPrefix(key, "-") {
						if err := dep2.ResolveDependencies(mergeActions[i].FileKey, fullFilePath, key, item.(string), result2); err != nil {
							return nil, err
						}
					} else {
						if err := dep.ResolveDependencies(mergeActions[i].FileKey, fullFilePath, key, item.(string), result1); err != nil {
							return nil, err
						}
					}
				}
			}
		} else {
			fullFilePath :=  mergeActions[i].FileKey + mergeActions[i].FileExt
			mergeActionCode := mergeActions[i].Action
			fileActionPrefix := ""

			if docType == DESTINATION {
				if mergeActionCode == ADD {
					mergeActionCode = DELETE
				} else if mergeActionCode == DELETE {
					mergeActionCode = ADD
				}
			}

			switch mergeActionCode {
			case ADD, MERGE:
				fileActionPrefix = "A"
			case DELETE:
				fileActionPrefix = "D"
			}

			//if its a binary
			if strings.HasPrefix(mergeActions[i].FileKey, "pages" + string(os.PathSeparator)) {
				objectID := strings.TrimPrefix(mergeActions[i].FileKey, "pages" + string(os.PathSeparator))
				jsonFilePath := "~" + fullFilePath + "~$"
				dep.AddDependent(objectID, fileActionPrefix+jsonFilePath, fileActionPrefix+jsonFilePath, mergeActions[i].FileKey)

			} else {
				jsonFilePath := fileActionPrefix + "~" + fullFilePath + "~$"
				dep.AddDependent(mergeActions[i].FileKey, jsonFilePath, jsonFilePath, mergeActions[i].FileKey)
			}


		}
	}

	return fileMap1, nil
}

//Convert dependent objects to depencies jsonpaths
//depObj - global dependencies in all documents
//docDep - dependencies to external or internal object only for one document
func addDependencies(docType DocumentType, fileKey string, depObj * DependentObjects, docDep * DependentObjects, fileMap map[string]interface{}, stopFileKey map[string]bool) (*DependentObjects) {
	if docDep == nil {
		docDep = &DependentObjects{ docType, make(map[string]interface{}),make(map[string]interface{})}
	}
	//Find all global dependencies for references in this document
	//We need wthis in order to build dependend json paths
	for key, value := range docDep.DepObj { //Go thru all dependent object id's in a document

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

					//Add dependent path
					//if strings.HasPrefix(paths[k].(DependentObj).JsonPath, "A") || !strings.HasPrefix(depPaths[j].(DependentObj).FileKey, "pages") {
					docDep.AddDependentPath(paths[k].(DependentObj).JsonPath, depPaths[j].(DependentObj).Ref, depPaths[j].(DependentObj).JsonPath)
					//}

					//get file details from associated map build by buildDependencePaths in order to get particular dependent objects for file
					fileMerge, isFileMerge := fileMap[depPaths[j].(DependentObj).FileKey].(FileMerge)

					docDiffs := fileMerge.FileDiff.DepDoc1

					if docType == DESTINATION {
						docDiffs = fileMerge.FileDiff.DepDoc2
					}

					if isFileMerge && docDiffs != nil {

						//Avoid endless recursions by keeping all files keys in map
						if !stopFileKey[fileMerge.FileKey] {

							//Find dependencies recursively
							docSubDep := &DependentObjects{ docType, docDiffs.DepObj, make(map[string]interface{})}
							stopFileKey[fileKey] = true

							subDep := addDependencies(docType, depPaths[j].(DependentObj).FileKey, depObj, docSubDep, fileMap, stopFileKey)

							//Add jsonpaths to current dependency
							for subKey, subPath := range subDep.DepPath {
								subDepPath, isPath := subPath.([]interface{})
								if isPath {
									for i := range subDepPath {
										var newFileKey= "~" + fileMerge.FileKey + fileMerge.FileExt + "~" + subKey
										//keep prefix if jsonpath contain fileName
										if strings.HasPrefix(subKey, "~") ||
											strings.HasPrefix(subKey, "A") ||
											strings.HasPrefix(subKey, "D") {
											newFileKey = subKey
										}
										//if strings.HasPrefix(subDepPath[i].(DependentObj).JsonPath, "A") || !strings.HasPrefix(depPaths[j].(DependentObj).FileKey, "pages") {
										docDep.AddDependentPath(newFileKey, subDepPath[i].(DependentObj).Ref, subDepPath[i].(DependentObj).JsonPath)
										//}
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
					//	if strings.HasPrefix(paths[k].(DependentObj).JsonPath, "A") || !strings.HasPrefix(depPaths[j].(DependentObj).FileKey, "pages") {
						docDep.AddDependentPath(paths[k].(DependentObj).JsonPath, ref, jsonpath)
					//	}
					}
				}
			}

		}
	}
	return docDep
}

func ProceedDependencies(workingDirV1 string, workingDirV2 string, fileMerge []FileMerge ) (*DependentObjects, *DependentObjects, *FileMerge, error) {

	depObj1 := DependentObjects{ SOURCE, make(map[string]interface{}), make(map[string]interface{})}
	depObj2 := DependentObjects{ SOURCE, make(map[string]interface{}), make(map[string]interface{})}

	//find all dependent jsonpaths by do_objectID or symbolID or any sketchID
	fileMap1, err1 := depObj1.buildDependencePaths(SOURCE, workingDirV1, workingDirV2, fileMerge, &depObj2)

	if err1!=nil {
		return nil, nil, nil, err1
	}

	fileMap2, err2 := depObj2.buildDependencePaths(DESTINATION, workingDirV2, workingDirV1, fileMerge, &depObj1)
	if err2!=nil {
		return nil, nil, nil, err2
	}

	//info1, _ := json.MarshalIndent(depObj1, "", "  ")
	//fmt.Printf("buildDependencePaths1: %v\n", string(info1))
	//
	//info2, _ := json.MarshalIndent(depObj2, "", "  ")
	//fmt.Printf("buildDependencePaths2:  %v\n", string(info2))

	var fileMergeDoc * FileMerge

	//build dependent jsonpaths for each file
	for i := range fileMerge {
		fileMerge[i].FileDiff.DepDoc1 = addDependencies(SOURCE, fileMerge[i].FileKey, &depObj1, fileMerge[i].FileDiff.DepDoc1, fileMap1, make(map[string]bool))
		fileMerge[i].FileDiff.DepDoc2 = addDependencies(DESTINATION, fileMerge[i].FileKey, &depObj2, fileMerge[i].FileDiff.DepDoc2, fileMap2, make(map[string]bool))

		if fileMerge[i].FileKey == "document" {
			fileMergeDoc = &fileMerge[i]
		}
	}
	return &depObj1, &depObj2, fileMergeDoc, nil;
}

//Find dependent jsonpaths to matchingKey recursively
//depPaths - are jsonpaths build by addDependencies method, store all dependent merge actions into diffs array
//matchingKey is merge jsonpath
func FindMatchingDiffs(docType DocumentType,fileName string, matchingKey string, depPaths1 map[string]interface{}, depPaths2 map[string]interface{}, diffs map[string]interface{}) {

	//TODO: Filter out attributes for dependencies do it different way.
	// We don't need lookup for dependencies for many attributes like frames points and etc, because there is no
	// dependncies for float values
	// could we just check if it's a primitive type? it could very specific fot points or is there any other type
	if strings.Contains(matchingKey, "resizingConstraint") || strings.Contains(matchingKey, "archivedAttributedString") || strings.Contains(matchingKey, "points") || strings.Contains(matchingKey, "frame") {
		return
	}
	//remove file names from action jsonpath
	matchingKeyWithouFile := FlatJsonPath(matchingKey, false)
	//if it's delete action that will affect destination file so changing processing file
	if strings.HasPrefix(matchingKeyWithouFile, "-") && docType == SOURCE {
		//matchingAction := "+" + FlatJsonPath(matchingKey, true)

		FindMatchingDiffs(DESTINATION, fileName, matchingKey, depPaths2, depPaths1, diffs)
		return
	}

	//remove file action specific elements from jsonpath
	flatMatch := FlatJsonPath(matchingKey, true)

	//ignore sequence change dependencies because they are refering to parent elements
	if strings.HasPrefix(matchingKeyWithouFile, "^") {
		return
	}

	//We shouldn't lookup matching element for file add/delete actions because it has $ document root action
	//and we should not lookup all elements
	if strings.HasPrefix(matchingKey, "A") {
		return
	}
	if strings.HasPrefix(matchingKey, "D") {
		return
	}

	//log.Printf("FindMatchingDiffs: %v %v", matchingKey, len(depPaths1))
	for key, item := range depPaths1 {

		//remove all file actions from key jsonpath
		flatKey := FlatJsonPath(key, true)

		//find only dependent paths matching given matchingKey or having it as prefix
		if flatKey == flatMatch || strings.HasPrefix(flatKey, flatMatch) {

			paths, isPaths := item.([]interface{})
			if isPaths {
				//go thru all paths for given matchingKey jsonpath may look as follows:
				//	"~meta.json~$[\"pagesAndArtboards\"][\"C416AD9F-4C8F-49F9-B431-DB6F5B964911\"][\"artboards\"]": [
				//	{
				//	"path": "+$[\"pagesAndArtboards\"][\"C416AD9F-4C8F-49F9-B431-DB6F5B964911\"][\"artboards\"][\"8291C1C6-561D-4A25-B542-79E8FE345D57\"]",
				//	"ref": "$[\"pagesAndArtboards\"][\"C416AD9F-4C8F-49F9-B431-DB6F5B964911\"][\"artboards\"]"
				//	}
				//	]
				for i := range paths {
					newKey := paths[i].(DependentObj).JsonPath
					fileKey := ReadFileAction(key)
					fileNewKey := ReadFileAction(newKey)

					//log.Printf("fileName: %v -> %v\n", fileName, ReadFileKey(newKey) )

					//if it reffers to actual file just ignore

					//TODO: Better way to ignore dependencies in the same layer
					//Ignore actions in the same layer except sorting
					if fileNewKey == "" && !strings.HasPrefix(key,"^") {
						continue
					}
					if fileName == ReadFileKey(newKey) {
						continue
					}

					//if key refers to other file append to action which belongs to this file
					if fileKey != "" && fileNewKey == "" {
						newKey = fileKey + newKey
					}

					//We will mark all reverse actions with R letter and we will invert action
					//while merging
					dstActionPrefix := ""

					//Mark all destination actions to source as reverse, should invert all actions
					// + to - and invert changes
					//this is required in order to delete dependend objects if referencing object has bean deleted
					if docType == DESTINATION {
						dstActionPrefix = "R"
					}

					//if there is similar element just ignore it
					if diffs[dstActionPrefix + newKey] != nil {
						continue
					}

					//we should not look dependencies in other pages/* files because
					//because lost references in that case will be processed correctlly
					//but we should allow additions and deletions
					if !strings.HasPrefix(ReadFileKey(newKey), "pages" + string(os.PathSeparator)) || strings.HasPrefix(newKey, "A") || strings.HasPrefix(newKey, "D"){
						//store new jsonpath pair
						diffs[dstActionPrefix+newKey] = paths[i].(DependentObj).Ref //+ " ← " + key

						//Look up dependencies recursively for newKey
						FindMatchingDiffs(docType, fileName, newKey, depPaths1, depPaths2, diffs)
					}
				}
			}
		}
	}

}

