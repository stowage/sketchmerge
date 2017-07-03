// Copyright 2017 Sergey Fedoseev. All rights reserved.
// This module contains functions needed to compare json trees mainly for Sketch App
// The main idea is to traverse both trees and build json paths for each tree
// We also take into account changes in array order
// This is not a regular jsonpath, but json paths queries with extensions
// Regular jsonpath:
// 	$["layers"][1] - addresses an element in array
// 	$["layers"][@do_objectID='59E11126-A64E-4325-9832-1F4D625C272B'] - addresses an element in array by finding appropriate key
//	$["layers"][1]["frame"] - addresses property
// Json Path with actions:
//	^$["layers"] - tells that sequence of array has changed
//	+$["layers"][3] - tells that layer at index 3 should be added
//	-$["layers"][4] - tells that layer at index 4 should be deleted
//	~pages/9E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBB.json~$["layers"][0] - refers to first layer in file pages/9E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBB.json
//	A~pages/9E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBB.json~$ - tells that file has to be copied from source to destination
//	D~pages/9E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBB.json~$ - tells that file has to be deleted
//	R+$["layers"][3] - this is reverse action, it tells that layer at index 3 should be deleted (R prefix means reverse action)

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
	"os/exec"
)

type LayerFrame struct {
	x json.Number `json:"x"`
	y json.Number `json:"y"`
	width json.Number `json:"width"`
	height json.Number `json:"height"`
}

// Structure of sketch folder
type SketchFileStruct struct {
	fileSet map[string] interface{}
	name string
}

//Sketch layer information
type SketchLayerInfo struct {
	LayerName string
	LayerID string
	LayerNames []string
	LayerIDs []string
	LayerFrames []*LayerFrame
	LayerPaths []string
	ArtboardName string
	ArtboardID string
	PageName string
	PageID string
	ActualPath string
	ArtboardPath string
	ClassName string
	frame *LayerFrame
	Data map[string]interface{}
}

//Unique id for sketch layer
func (li * SketchLayerInfo) fingerprint(solt string) string {
	return li.LayerID + "/" + li.ArtboardID + "/" + li.PageID + "/" + solt
}

//Interface for setting up/grouping differences
type Difference interface {
	SetDiff(action ApplyAction, actualPath, src , dst, name, className string, loc string) DiffObject
	SetCollision(oid string)
	GetDiff() map[string]interface{}
}

//Main object describing differencies, Changes field contains jsonpaths
type DiffObject struct {

	Description map[string]interface{} `json:"description,omitempty"`
	Changes map[string]interface{} `json:"changes,omitempty"`

}

//Parent structure for all type of differencies
type MainDiff struct {
	DiffInfo map[string]interface{} `json:"info,omitempty"`
	Diff map[string]interface{} `json:"diff,omitempty"`
	Difference `json:"-"`

}

//Sketch layer differencies
type SketchLayerDiff struct {
	Name string `json:"name,omitempty"`
	Data map[string]interface{} `json:"data,omitempty"`
	LayerDiff map[string]interface{} `json:"layer_diff,omitempty"`
	MainDiff
}

//Artboard differencies
type SketchArtboardDiff struct {
	Name string `json:"name,omitempty"`
	Data map[string]interface{} `json:"data,omitempty"`
	LayerDiff map[string]interface{} `json:"layer_diff,omitempty"`
	MainDiff
}

//Page differencies
type SketchPageDiff struct {
	Name string `json:"name,omitempty"`
	ArtboardDiff map[string]interface{} `json:"artboard_diff,omitempty"`
	LayerDiff map[string]interface{} `json:"layer_diff,omitempty"`
	MainDiff
}

//Sketch document differencies
type SketchDiff struct {
	PageDiff map[string]interface{} `json:"page_diff,omitempty"`
	MainDiff
	ShowInfo bool `json:"-"`
	Dump1 * DumpInfo `json:"-"`
	Dump2 * DumpInfo `json:"-"`
}

//Resturns differences structure
func (sd* MainDiff) GetDiff() map[string]interface{}  {
	return sd.Diff
}

//Set reference collision object id (not used)
func (sd* MainDiff) SetCollision(oid string) {
	sd.DiffInfo["CollisionID"] = oid
}

//Set difference info for given layer, artboard or page
func (sd* MainDiff) SetDiff(action ApplyAction, actualPath, src , dst , name, className string, loc string) DiffObject  {

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


	localDiffs, ok := sd.Diff[loc].(map[string]interface{})
	if !ok {
		localDiffs = make(map[string]interface{})

		sd.Diff[loc] = localDiffs

	}

	diffs, ok := localDiffs[actualPath].(DiffObject)
	if !ok {
		diffs = DiffObject{make(map[string]interface{}), make(map[string]interface{})}
		localDiffs[actualPath] = diffs
	}

	changes := diffs.Changes
	descriptions := diffs.Description

	descriptions["name"] = name

	if className != "" {
		descriptions["class"] = className
	}

	//if frame != nil {
	//	frameMap := make(map[string]interface{})
	//	frameMap["x"] = frame.x
	//	frameMap["y"] = frame.y
	//	frameMap["width"] = frame.width
	//	frameMap["height"] = frame.height
	//	descriptions["frame"] = frameMap
	//}

	if preAction, ok := descriptions["action"].(string); ok {
		if preAction == "SequenceChange" {
			descriptions["action"] = strAction
		} else if preAction == "ValueChange" && action != SequenceChange {
			descriptions["action"] = strAction
		} else if preAction == "ValueDelete" && action == ValueAdd {
			descriptions["action"] = "ValueChange"
		} else if preAction == "ValueAdd" && action == ValueDelete {
			descriptions["action"] = "ValueChange"
		}
	} else {
		descriptions["action"] = strAction
	}

	descriptions["action"] = strAction
	changes[src] = dst

	return diffs

}

// Sketch file structure comparison
type FileActionType uint8

//File structure merge actions
const (
	MERGE = iota
	ADD
	DELETE

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
	hasInfo bool
	dump1 * DumpInfo
	dump2 * DumpInfo
	sketchPath * string
	exportPath * string
	sketchFileV1 * string
	sketchFileV2 * string
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
	DepDoc1 * DependentObjects `json:"-"`

	//Dependent objects for dst document
	DepDoc2 * DependentObjects `json:"-"`

}

//Performs comparison of two json files
//doc1File, doc2File are paths
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

func (li * SketchLayerInfo) appendFrame(diffData map[string]interface{},path string, frame * LayerFrame, loc string) {

	if li.Data != nil {

		if _, ok := diffData[loc]; !ok {
			diffData[loc] = make(map[string]interface{})
		}

		locData := diffData[loc].(map[string]interface{})

		if _, ok := locData[li.ActualPath]; !ok {
			locData[path] = make(map[string]interface{})
		}

		data := locData[path].(map[string]interface{})
		if frame != nil {
			frameMap := make(map[string]interface{})
			frameMap["x"] = frame.x
			frameMap["y"] = frame.y
			frameMap["width"] = frame.width
			frameMap["height"] = frame.height
			data["frame"] = frameMap
		}
	}
}

func mergeContentData(existingData map[string]interface{}, newData map[string]interface{}) map[string]interface{} {
	isEmpty := true
	isSingleLeveled := true
	firstKey := ""
	secondKey := ""

	if existingData != nil {
		isEmpty = false
	}

	if !isEmpty {
		 if data, hasData := newData["diff"].(map[string]interface{}); hasData {
			 for key, value := range data {
				 firstKey = key
				 if prop, ok := value.(map[string]interface{}); ok {
					 for key, _ := range prop {
						 secondKey = key
						 isSingleLeveled = false
						 break
					 }
				 }
				 break
			 }

			 diff, ok := existingData["diff"].(map[string]interface{})
			 if !ok {
				 existingData["diff"] = newData["diff"]
				 return existingData
			 }

			 if isSingleLeveled {
				 diff[firstKey] = data[firstKey]
			 } else {
				 subData, hasSubData := diff[firstKey].(map[string]interface{})
				 if hasSubData {
					 subData[secondKey] = data[firstKey].(map[string]interface{})[secondKey]
				 } else {
					 diff[firstKey] = data[firstKey]
				 }

			 }

			 return existingData
		 }


	}

	return newData
}

func (li * SketchLayerInfo) appendItems(diffData map[string]interface{}, loc string, srclayer, dstlayer interface{}) {

	if li.Data != nil {
		if len(li.Data) == 0 {
			return
		}

		if srclayer != nil && dstlayer != nil {
			if _, ok := diffData["info"]; !ok {
				diffData["info"] = make(map[string]interface{})
			}
			info := diffData["info"].(map[string]interface{})
			info["original"] = dstlayer
			info[loc] = srclayer
		}

		if _, ok := diffData[loc]; !ok {
			diffData[loc] = make(map[string]interface{})
		}

		locData := diffData[loc].(map[string]interface{})


		if _, ok := locData[li.ActualPath]; !ok {
			locData[li.ActualPath] = make(map[string]interface{})
		}

		data := locData[li.ActualPath].(map[string]interface{})

		for path, item := range li.Data {
			//data[path] = item
			if strings.HasPrefix(path, li.ActualPath) {
				mergeContentData(data, item.(map[string]interface{}))
			} else {
				data[path] = item
			}
		}
	}
}

//This method is part of nice json process
func (li * SketchLayerInfo) SetDifference(action ApplyAction, diff SketchDiff, diffSrc string, diffDst string, loc string, srclayer, dstlayer interface{}) (string, Difference) {

	var page interface{}
	var artboard interface{}
	var layer interface{}

	//Actual difference could be Page, Artboard or Layer
	//set it to existing difference by default
	var actual Difference = &diff
	var actualID string
	var actualName string
	var className string
	var frame * LayerFrame

	//if PageID is recognized
	if li.PageID != "" {
		page = diff.PageDiff[li.PageID]

		//if page is not exists then create it
		if page == nil {
			page = SketchPageDiff{Name: li.PageName, ArtboardDiff: make(map[string]interface{}), LayerDiff: make(map[string]interface{}), MainDiff: MainDiff{ Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{}) }}
			diff.PageDiff[li.PageID] = page
		}
		_page := page.(SketchPageDiff)

		//set actual difference is Page
		actual = &_page
		actualID = li.PageID
		actualName = li.PageName
		className = li.ClassName
		frame = li.frame
	}

	//only if we are inside page try to recognize artboard
	if page != nil && li.ArtboardID != "" {
		artboard = page.(SketchPageDiff).ArtboardDiff[li.ArtboardID]

		if artboard == nil {
			artboard = SketchArtboardDiff{Name: li.ArtboardName, LayerDiff: make(map[string]interface{}), MainDiff:MainDiff{ Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, Data:make(map[string]interface{}) }
			page.(SketchPageDiff).ArtboardDiff[li.ArtboardID] = artboard
			data := artboard.(SketchArtboardDiff).Data
			li.appendFrame(data, li.ArtboardPath, li.frame, loc)
		}


		_artboard := artboard.(SketchArtboardDiff)
		data := _artboard.Data

		//li.appendFrame(data, li.ArtboardPath, li.frame, loc)

		if li.LayerID == "" {
			li.appendItems(data, loc, srclayer, dstlayer)
		}

		//set actual differnce to artboard
		actual = &_artboard
		actualID = li.ArtboardID
		actualName = li.ArtboardName
		className = li.ClassName
		frame = li.frame
	}

	//if it is artboard
	if artboard != nil && li.LayerID != ""{

		//Get recent layer
		if len(li.LayerIDs) > 0 {
			layer = artboard.(SketchArtboardDiff).LayerDiff[li.LayerIDs[0]]

			if layer == nil {
				layer = SketchLayerDiff{Name: li.LayerNames[0], LayerDiff:make(map[string]interface{}) , MainDiff: MainDiff{Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, Data:make(map[string]interface{})}
				artboard.(SketchArtboardDiff).LayerDiff[li.LayerIDs[0]] = layer
				data := layer.(SketchLayerDiff).Data
				li.appendFrame(data, li.LayerPaths[0], li.LayerFrames[0], loc)
			}



			if len(li.LayerIDs) > 1 {
				for i := 1; i < len(li.LayerIDs); i++ {
					parentLayer := layer

					layer = parentLayer.(SketchLayerDiff).LayerDiff[li.LayerIDs[i]]
					if layer == nil {
						layer = SketchLayerDiff{Name: li.LayerNames[i], LayerDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, Data:make(map[string]interface{})}
						parentLayer.(SketchLayerDiff).LayerDiff[li.LayerIDs[i]] = layer
						data := layer.(SketchLayerDiff).Data
						li.appendFrame(data, li.LayerPaths[i], li.LayerFrames[i], loc)
					}


				}
				data := layer.(SketchLayerDiff).Data

				li.appendItems(data, loc, srclayer, dstlayer)
			} else {
				data := layer.(SketchLayerDiff).Data
				li.appendItems(data, loc, srclayer, dstlayer)
			}

		} else {
			layer = artboard.(SketchArtboardDiff).LayerDiff[li.LayerID]

			if layer == nil {
				layer = SketchLayerDiff{Name: li.LayerName, MainDiff: MainDiff{Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, Data:make(map[string]interface{})}
				artboard.(SketchArtboardDiff).LayerDiff[li.LayerID] = layer
				data := layer.(SketchLayerDiff).Data
				li.appendFrame(data, li.ActualPath, li.frame, loc)
			}

			data := layer.(SketchLayerDiff).Data

			li.appendItems(data, loc, srclayer, dstlayer)



		}
		_layer := layer.(SketchLayerDiff)

		actual = &_layer
		actualID = li.LayerID
		actualName = li.LayerName
		className = li.ClassName
		frame = li.frame
	} else if page != nil && li.LayerID != "" {
		if len(li.LayerIDs) > 0 {
			layer = page.(SketchPageDiff).LayerDiff[li.LayerIDs[0]]

			if layer == nil {
				layer = SketchLayerDiff{Name: li.LayerNames[0], LayerDiff:make(map[string]interface{}) , MainDiff: MainDiff{Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, Data:make(map[string]interface{})}
				page.(SketchPageDiff).LayerDiff[li.LayerIDs[0]] = layer
				data := layer.(SketchLayerDiff).Data
				li.appendFrame(data, li.LayerPaths[0], li.LayerFrames[0], loc)

			}

			if len(li.LayerIDs) > 1 {
				for i := 1; i < len(li.LayerIDs); i++ {
					parentLayer := layer

					layer = parentLayer.(SketchLayerDiff).LayerDiff[li.LayerIDs[i]]
					if layer == nil {
						layer = SketchLayerDiff{Name: li.LayerNames[i], LayerDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, Data:make(map[string]interface{})}
						parentLayer.(SketchLayerDiff).LayerDiff[li.LayerIDs[i]] = layer
						data := layer.(SketchLayerDiff).Data
						li.appendFrame(data, li.LayerPaths[i], li.LayerFrames[i], loc)
					}

				}
				data := layer.(SketchLayerDiff).Data

				li.appendItems(data, loc, srclayer, dstlayer)
			} else {
				data := layer.(SketchLayerDiff).Data
				li.appendItems(data, loc, srclayer, dstlayer)
			}

		} else {
			layer = page.(SketchPageDiff).LayerDiff[li.LayerID]

			if layer == nil {
				layer = SketchPageDiff{Name: li.LayerName, LayerDiff:make(map[string]interface{}), MainDiff: MainDiff{Diff: make(map[string]interface{}), DiffInfo: make(map[string]interface{})}}
				page.(SketchPageDiff).LayerDiff[li.LayerID] = layer
			}
		}
		_layer := layer.(SketchLayerDiff)

		actual = &_layer
		actualID = li.LayerID
		actualName = li.LayerName
		className = li.ClassName
		frame = li.frame
	}
	_=frame

	_=actual.SetDiff(action, li.ActualPath, diffSrc, diffDst, actualName, className, loc)

	return actualID, actual

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
//Compares to file sets from two folders
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
		mergeAction.FileExt = filepath.Ext(key)
		mergeAction.FileKey = strings.TrimSuffix(key, mergeAction.FileExt)
		mergeAction.Action = ADD
		fs.MergeActions = append(fs.MergeActions, *mergeAction)
	}
}

//Performs dependencies analysis
func (fsMerge * FileStructureMerge) ProduceDiffWithDependencies() error {
	for i := range fsMerge.MergeActions {
		fileName := fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
		//Go thru all differences
		for key, _ := range fsMerge.MergeActions[i].FileDiff.Doc1Diffs {
			matchingDiffs := make(map[string]interface{})
			//find dependent jsonpaths
			FindMatchingDiffs(SOURCE, fileName, key, fsMerge.MergeActions[i].FileDiff.DepDoc1.DepPath, fsMerge.MergeActions[i].FileDiff.DepDoc2.DepPath, matchingDiffs)

			//Add them to actions
			for mKey, mItem := range matchingDiffs {
				fsMerge.MergeActions[i].FileDiff.Doc1Diffs[mKey] = mItem
			}

		}
		if fsMerge.MergeActions[i].FileDiff.Doc1Diffs == nil {
			continue
		}


		//We need specific workaround for adding or removing pages
		//fileAction := fsMerge.MergeActions[i].Action
		//if strings.HasPrefix(fileName, "pages" + string(os.PathSeparator)) {
		//	if fileAction == ADD || fileAction == DELETE {
		//		fileActionPath := BuildFileAction(fileAction, fileName)
		//		fsMerge.MergeActions[i].FileDiff.Doc1Diffs[fileActionPath] = fileActionPath
		//		fsMerge.MergeActions[i].FileDiff.Doc1Diffs["~document.json~$[\"pages\"]"] = "~document.json~$[\"pages\"]"
		//	}
		//}

	}
	//return json.MarshalIndent(fsMerge, "", "  ")
	return nil

}

func getLayerFrame(iframe interface{}) *LayerFrame {

	if iframe == nil {
		return nil
	}

	frame, isFrame := iframe.(map[string]interface{})

	if isFrame {
		x, hasX := frame["x"].(json.Number)
		y, hasY := frame["y"].(json.Number)
		width, hasWidth := frame["width"].(json.Number)
		height, hasHeight := frame["height"].(json.Number)

		if hasX && hasY && hasWidth && hasHeight {
			return &LayerFrame{x, y, width, height}
		}

	}
	return nil

}

func (li * SketchLayerInfo) propertyChangeInfo(doc1 map[string]interface{}, doc2 map[string]interface{}, key1 string, key2 string, loc string, dump1, dump2 * DumpInfo) (interface{}, interface{}) {
	layerSel, _, _ := Parse(li.ActualPath)
	srcSel, srcact, _ := Parse(key1)
	dstSel, _, _ := Parse(key2)

	if srcact == ValueChange {
		layervalue, _, _ := layerSel.Apply(doc1)
		var srclayer interface{}
		srcvalue, lastnode, _ := srcSel.ApplyWithEvent(doc1, func(v interface{}, prevNode Node, node Node) bool {
			if prevNode == nil {
				layer := v.(map[string]interface{})
				if layer != nil {
					srclayer = layer
				}
			} else if prevNode.GetKey() == "layers" {
				layer := v.(map[string]interface{})
				if layer != nil {
					srclayer = layer
				}
			}
			return true
		})
		var dstlayer interface{}
		dstvalue, _, _ := dstSel.ApplyWithEvent(doc2, func(v interface{}, prevNode Node, node Node) bool {
			if prevNode == nil {
				layer := v.(map[string]interface{})
				if layer != nil {
					dstlayer = layer
				}
			} else if prevNode.GetKey() == "layers" {
				layer := v.(map[string]interface{})
				if layer != nil {
					dstlayer = layer
				}
			}
			return true

		})

		if layer, ok := layervalue.(map[string]interface{}); ok {
			data := make(map[string]interface{})

			for key, _ := range layer {
				if key != "layers" {
				 	if strings.Contains(key1,  fmt.Sprintf("\"%v\"", key)) {

						info := srcvalue
						//make([]interface{}, 2)
						//info[0] = srcvalue
						//info[1] = dstvalue

						prop := make(map[string]interface{})
						selkey := lastnode.GetKey()
						if selkey == nil || selkey == key {
							selkey = ""
							data[key] = info
						} else {
							prop[fmt.Sprintf("%v", selkey)] = info

							data[key] = prop
						}
					}
				}
			}

			meta := make(map[string]interface{})

			meta["diff"] = data
			li.Data[key1] = meta
		}

		log.Printf("dump1: %v\n", len(dump1.ObjectsMap))
		if layer, ok := srclayer.(map[string]interface{}); ok {

			layerObjID, okLayer := layer["do_objectID"].(string)
			dumpObjID, okDump := dump1.ObjectsMap[layerObjID].(map[string]interface{})
			if dump1 != nil && okDump && okLayer {

				layer = dumpObjID
				layer["_class"] = srclayer.(map[string]interface{})["_class"]
			}

			data := make(map[string]interface{})

			for key, value := range layer {
				if key != "layers" {
					data[key] = value
				}
			}
			srcvalue = data
		}

		log.Printf("dump2: %v\n", len(dump2.ObjectsMap))
		if layer, ok := dstlayer.(map[string]interface{}); ok {

			layerObjID, okLayer := layer["do_objectID"].(string)
			dumpObjID, okDump := dump2.ObjectsMap[layerObjID].(map[string]interface{})
			if dump2 != nil && okDump && okLayer {
				layer = dumpObjID
				layer["_class"] = dstlayer.(map[string]interface{})["_class"]
			}

			data := make(map[string]interface{})

			for key, value := range layer {
				if key != "layers" {
					data[key] = value
				}
			}
			dstvalue = data
		}

		return srcvalue, dstvalue
		//li.Data["original"] = dstvalue
		//li.Data[loc] = srcvalue

	}

	return nil, nil

}

//Find most actual object representing given layer
func ReadKeyValue(doc1 map[string]interface{}, doc2 map[string]interface{}, key string) (SketchLayerInfo, ApplyAction) {
	var pageID = ""
	var pageName = ""

	var artboardID = ""
	var artboardName = ""


	layerIDs := make([]string, 0)
	layerNames := make([]string, 0)
	layerFrames := make([]*LayerFrame, 0)
	layerPaths := make([]string, 0)

	var layerID = ""
	var layerName string = ""
	var layerPath string = ""

	var actualPath string = ""
	var artboardPath string = ""

	var className = ""

	var frame * LayerFrame

	//Parse jsonpath in key
	srcSel, srcact, _ := Parse(key)
	doc := doc1

	//Delete action usually belongs to destination document which is doc2
	if /*item == "" &&*/ srcact == ValueDelete {
		doc = doc2
	}

	//Walk thru parsed path key
	_, _, err := srcSel.ApplyWithEvent(doc, func(v interface{}, prevNode Node, node Node) bool {

		if prevNode == nil {
			//This is root element which is page
			layer := v.(map[string]interface{})
			if layer != nil {
				lname := layer["name"]
				lid := layer["do_objectID"]
				lclass := layer["_class"]
				lframe := layer["frame"]
				if lname == nil || lid == nil {
					return true
				}

				pageName = lname.(string)
				pageID = lid.(string)
				layerPath = pageName
				actualPath = GetPath(node)
				if lclass != nil {
					className = lclass.(string)
				}
				if lframe != nil {
					frame = getLayerFrame(lframe)
				}

			}
		} else if prevNode.GetKey() == "layers" {
			//This is any layer inside page
			layer := v.(map[string]interface{})
			if layer != nil {
				lname := layer["name"]
				lid := layer["do_objectID"]
				lclass := layer["_class"]
				lframe := layer["frame"]
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
					actualPath =  GetPath(node)
					artboardPath = actualPath
					if lclass != nil {
						className = lclass.(string)
					}
					if lframe != nil {
						frame = getLayerFrame(lframe)
					}
				} else if layer["_class"] == "artboard" {
					artboardName = lname.(string)
					artboardID = lid.(string)
					layerPath += "/" + artboardName
					actualPath =  GetPath(node)
					artboardPath = actualPath
					if lclass != nil {
						className = lclass.(string)
					}
					if lframe != nil {
						frame = getLayerFrame(lframe)
					}
				} else  {
					layerName = lname.(string)
					layerID = lid.(string)
					layerPath += "/" + layerName
					actualPath =  GetPath(node)
					if lclass != nil {
						className = lclass.(string)
					}
					if lframe != nil {
						layerFrames = append(layerFrames, getLayerFrame(lframe))
					} else {
						layerFrames = append(layerFrames, nil)
					}
					layerIDs = append(layerIDs, layerID)
					layerNames = append(layerNames, layerName)
					layerPaths = append(layerPaths, actualPath)

				}





			}

		}
		return true;
	})

	if err!=nil {
		log.Printf("Error occurired while building nice diff: %v", err)
	}



	diff := SketchLayerInfo{ layerName, layerID,
				layerNames, layerIDs, layerFrames, layerPaths,
				artboardName, artboardID,
				pageName, pageID,
				actualPath, artboardPath, className, frame, make(map[string]interface{})}

	return diff, srcact
}

//There some dependencies in meta.json file which we have to omit
func isPageActionToOmit (fileName, key string) bool {
	if strings.Contains(key, "~meta.json~") {
		return false

	}
	return true
}

//Builds nice json for pages only
func (skDiff * SketchDiff) ProduceNiceDiff(loc string, fileAction FileActionType, fileName string, doc1 map[string]interface{}, doc2 map[string]interface{}, diff map[string]interface{}, depPaths1 map[string]interface{}, depPaths2 map[string]interface{}, depObj1,depObj2 *DependentObjects, docDep1, docDep2 * DependentObjects)  {

	defer TimeTrack(time.Now(), "ProduceNiceDiff " + fileName)
	if diff==nil {
		return
	}

	//Go thru all differences jsonpath's in actual file
	for key, item := range diff {

		diff, srcact := ReadKeyValue(doc1, doc2, key)

		var srclayer, dstlayer interface{}
		if(skDiff.ShowInfo) {
			srclayer, dstlayer = diff.propertyChangeInfo(doc1, doc2, key, item.(string), loc, skDiff.Dump1, skDiff.Dump2)
		}

		diff.SetDifference(srcact, *skDiff, key, item.(string), loc, srclayer, dstlayer)

		mathingDiffs := make(map[string]interface{})

		FindMatchingDiffs(SOURCE, fileName, key, depPaths1, depPaths2, mathingDiffs)


		for mKey, mItem := range mathingDiffs {
			if strings.HasPrefix(fileName, "pages" + string(os.PathSeparator)) {
				if fileAction != MERGE {
					if !isPageActionToOmit(fileName, mKey) {
						continue
					}
				}
			}
			diff.SetDifference(srcact, *skDiff, mKey, mItem.(string), loc, nil, nil)
		}

		if strings.HasPrefix(fileName, "pages" + string(os.PathSeparator)) {
			if fileAction != MERGE {
				fileKey := strings.TrimSuffix(strings.TrimPrefix(fileName, "pages" + string(os.PathSeparator)), filepath.Ext(fileName))

				//if fileAction == ADD {
					items1, hasItems1 := depObj1.DepObj[fileKey].([]interface{})
					if hasItems1 {
						for i := range items1 {
							diff.SetDifference(srcact, *skDiff, items1[i].(DependentObj).JsonPath, items1[i].(DependentObj).Ref, loc, nil, nil)
						}
					}
				//} else {
					items2, hasItems2 := depObj2.DepObj[fileKey].([]interface{})
					if hasItems2 {
						for i := range items2 {
							r1, r2 := ReversAction(items2[i].(DependentObj).JsonPath, items2[i].(DependentObj).Ref)
							if r1 != "" {
								diff.SetDifference(srcact, *skDiff, r1, r2, loc, nil, nil)
							}
						}
					}

				if docDep1 != nil {
					page, hasPage := docDep1.DepObj[fileKey].([]interface{})
					if hasPage {
						for i := range page {
							dep, isDep := page[i].(DependentObj)
							if isDep {
								if fileAction == ADD {
									diff.SetDifference(srcact, *skDiff, "~document.json~+"+dep.JsonPath, "[\"pages\"]", loc, nil, nil)
								} else {
									diff.SetDifference(srcact, *skDiff, "~document.json~-"+dep.JsonPath, "", loc, nil, nil)
								}
							}
						}
					}
				}
				if docDep2 != nil {
					page, hasPage := docDep2.DepObj[fileKey].([]interface{})
					if hasPage {
						for i := range page {
							dep, isDep := page[i].(DependentObj)
							if isDep {
								if fileAction == ADD {
									diff.SetDifference(srcact, *skDiff, "~document.json~+"+dep.JsonPath, "[\"pages\"]", loc, nil, nil)
								} else {
									diff.SetDifference(srcact, *skDiff, "~document.json~-"+dep.JsonPath, "", loc, nil, nil)


								}
							}
						}
					}
				}
				//}
				//fileActionPath := BuildFileAction(fileAction, fileName)
				//diff.SetDifference(srcact, *skDiff, fileActionPath, fileActionPath, loc)
				//if fileAction == ADD {
				//	diff.SetDifference(srcact, *skDiff, "~meta.json~+$[\"pagesAndArtboards\"][\""+fileKey+"\"]", "~meta.json~$[\"pagesAndArtboards\"]", loc)
				////	diff.SetDifference(srcact, *skDiff, "~user.json~+$[\""+fileKey+"\"]", "~user.json~$", loc)
				//} else if fileAction == DELETE {
				//	diff.SetDifference(srcact, *skDiff, "~meta.json~-$[\"pagesAndArtboards\"][\""+fileKey+"\"]", "", loc)
				////	diff.SetDifference(srcact, *skDiff, "~user.json~-$[\""+fileKey+"\"]", "~user.json~$", loc)
				//}
				//diff.SetDifference(srcact, *skDiff, "~document.json~$[\"pages\"]", "~document.json~$[\"pages\"]", loc)
			}
		}
	}

}

//This function is used in order to organize jsonpaths in "local" and "remote" groups
//because it allows to store SketchDiff outside
func (fm * FileMerge) NiceDifference(loc string, workingDirV1, workingDirV2 string, skDiff1 SketchDiff, skDiff2 SketchDiff, depObj1, depObj2 *DependentObjects, fileMergeDoc * FileMerge) error {
	doc1File := workingDirV1 + string(os.PathSeparator) + fm.FileKey + fm.FileExt
	doc2File := workingDirV2 + string(os.PathSeparator) + fm.FileKey + fm.FileExt

	var result1 map[string]interface{}
	var result2 map[string]interface{}

	if _, err := os.Stat(doc1File); os.IsNotExist(err) {
		result1 = make(map[string]interface{})
	} else {
		var err1 error
		result1, err1 = readJSON(doc1File)

		if err1 != nil {
			return err1
		}
	}


	if _, err := os.Stat(doc2File); os.IsNotExist(err) {
		result2 = make(map[string]interface{})
	} else {
		var err2 error
		result2, err2 = readJSON(doc2File)

		if err2 != nil {
			return err2
		}
	}

	jsCompare := fm.FileDiff

	fileName := fm.FileKey + fm.FileExt
	fileAction := fm.Action

	var docDep1 * DependentObjects
	var docDep2 * DependentObjects

	if fileMergeDoc != nil {
		docDep1 = fileMergeDoc.FileDiff.DepDoc1
		docDep2 = fileMergeDoc.FileDiff.DepDoc2
	}

	skDiff1.ProduceNiceDiff(loc, fileAction, fileName, result1, result2, jsCompare.Doc1Diffs, jsCompare.DepDoc1.DepPath, jsCompare.DepDoc2.DepPath, depObj1, depObj2, docDep1, docDep2)

	if fileAction == ADD {
		fileAction = DELETE
	} else if fileAction == DELETE {
		fileAction = ADD
	}

	skDiff2.ProduceNiceDiff(loc, fileAction, fileName, result2, result1, jsCompare.Doc2Diffs, jsCompare.DepDoc2.DepPath, jsCompare.DepDoc1.DepPath, depObj2, depObj1, docDep2, docDep1)

	return nil
}

//This function groups differences in local and remote groups
//local, remote changes/jsopaths allows to apply changes from different sources
func ProcessFileStructures3Way(workingDirV0, workingDirV1, workingDirV2 string, fsMerge1 * FileStructureMerge, fsMerge2 * FileStructureMerge, depObj11, depObj12, depObj21, depObj22 *DependentObjects, fileMergeDoc1, fileMergeDoc2 * FileMerge) (*FileStructureMerge,error) {
	fmMap := make(map[string]FileMerge)
	sk1Map := make(map[string]interface{})
	sk2Map := make(map[string]interface{})

	for i := range fsMerge1.MergeActions {
		fm := fsMerge1.MergeActions[i]
		fileName := fm.FileKey + fm.FileExt
		fmMap[fileName] = fm

		if filepath.Ext(strings.ToLower(fm.FileKey + fm.FileExt)) != ".json" {
			continue
		}

		skDiff1 := SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), DiffInfo: make(map[string]interface{})}}
		skDiff2 := SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), DiffInfo: make(map[string]interface{})}}
		if err :=fm.NiceDifference("local", workingDirV1, workingDirV0, skDiff1, skDiff2, depObj11, depObj12, fileMergeDoc1); err != nil {
			return nil, err
		}
		sk1Map[fileName] = skDiff1
		sk2Map[fileName] = skDiff2

	}

	for i := range fsMerge2.MergeActions {

		fm := fsMerge2.MergeActions[i]
		fileName := fsMerge2.MergeActions[i].FileKey + fsMerge2.MergeActions[i].FileExt
		skDiff1 := sk1Map[fileName]
		skDiff2 := sk2Map[fileName]

		if skDiff1 == nil || skDiff2 == nil {
			fmMap[fileName] = fsMerge2.MergeActions[i]
			skDiff1 = SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), DiffInfo: make(map[string]interface{})}}
			skDiff2 = SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), DiffInfo: make(map[string]interface{})}}

			if filepath.Ext(strings.ToLower(fm.FileKey + fm.FileExt)) != ".json" {
				continue
			}

			sk1Map[fileName] = skDiff1
			sk2Map[fileName] = skDiff2
		}

		if filepath.Ext(strings.ToLower(fm.FileKey + fm.FileExt)) != ".json" {
			continue
		}

		if err :=fm.NiceDifference("remote", workingDirV2, workingDirV0, skDiff1.(SketchDiff), skDiff2.(SketchDiff), depObj21, depObj22, fileMergeDoc2); err != nil {
			return nil, err
		}
	}

	fsMerge := FileStructureMerge{MergeActions: make([]FileMerge, len(fmMap) )}

	i := 0
	for fileName, item := range fmMap {

		skDiff1 := sk1Map[fileName]
		skDiff2 := sk2Map[fileName]

		if skDiff1 != nil {
			item.FileDiff.Doc1Diffs = map[string]interface{}{"nice_diff": skDiff1}
		}
		if skDiff2 != nil {
			item.FileDiff.Doc2Diffs = map[string]interface{}{"nice_diff": skDiff2}
		}

		fsMerge.MergeActions[i] = item
		i++

	}

	return &fsMerge, nil
}

func launchSketchToolExport(sketchApp, objType, sketchFile, items, exportPath string) error {
	cmdArgs := []string{"export", objType, sketchFile, "--use-id-for-name=YES", "--scales=1", "--formats=png", "--items="+items, "--output="+exportPath}
	out, err := exec.Command(sketchApp, cmdArgs...).Output()
	if err == nil {
		log.Printf("%v", string(out))
	}
	return err
}

func (diff * SketchDiff) extractExportList() ([]string, []string) {
	pageDiff := diff.PageDiff
	pages := make([]string, 0)
	artboards := make([]string, 0)
	if len(pageDiff) > 0 {
		for key, _ := range pageDiff {
			pages = append(pages, key)
			if artDiff, ok := pageDiff[key].(SketchPageDiff); ok {
				artboardDiff := artDiff.ArtboardDiff
				for key, _ := range artboardDiff {
					artboards = append(artboards, key)
				}
			}
		}
	}

	return pages, artboards
}

//This function is used for two way merge
func (fsMerge * FileStructureMerge) ProduceNiceDiffWithDependencies(loc string, workingDirV1, workingDirV2 string, depObj1, depObj2 *DependentObjects, fileMergeDoc * FileMerge) error {
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
					return err1
				}
			}


			if _, err := os.Stat(doc2File); os.IsNotExist(err) {
				result2 = make(map[string]interface{})
			} else {
				var err2 error
				result2, err2 = readJSON(doc2File)

				if err2 != nil {
					return err2
				}
			}

			var docDep1 * DependentObjects
			var docDep2 * DependentObjects

			if fileMergeDoc != nil {
				docDep1 = fileMergeDoc.FileDiff.DepDoc1
				docDep2 = fileMergeDoc.FileDiff.DepDoc2
			}

			jsCompare := fsMerge.MergeActions[i].FileDiff

			fileName := fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
			fileAction := fsMerge.MergeActions[i].Action
			skDiff1 := SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, ShowInfo: fsMerge.hasInfo, Dump1: fsMerge.dump1, Dump2: fsMerge.dump2}

			skDiff1.ProduceNiceDiff(loc, fileAction, fileName, result1, result2, jsCompare.Doc1Diffs, jsCompare.DepDoc1.DepPath, jsCompare.DepDoc2.DepPath, depObj1, depObj2, docDep1, docDep2)

			if fsMerge.sketchPath != nil && fsMerge.exportPath != nil {
				pages, artboards := skDiff1.extractExportList()
				if len(artboards) > 0 {
					launchSketchToolExport(*fsMerge.sketchPath, "artboards", *fsMerge.sketchFileV1, strings.Join(artboards, ","), *fsMerge.exportPath + string(os.PathSeparator) + "v1")
				} else if len(pages) > 0 {
					launchSketchToolExport(*fsMerge.sketchPath, "pages", *fsMerge.sketchFileV1, strings.Join(pages, ","), *fsMerge.exportPath + string(os.PathSeparator) + "v1")
				}
			}
			if len(jsCompare.Doc1Diffs) > 0 {
				fsMerge.MergeActions[i].FileDiff.Doc1Diffs = map[string]interface{}{ "nice_diff" : skDiff1 }
			}

			if fileAction == ADD {
				fileAction = DELETE
			} else if fileAction == DELETE {
				fileAction = ADD
			}

			skDiff2 := SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), DiffInfo: make(map[string]interface{})}, ShowInfo: fsMerge.hasInfo, Dump1: fsMerge.dump1, Dump2: fsMerge.dump2}

			skDiff2.ProduceNiceDiff(loc, fileAction, fileName, result2, result1, jsCompare.Doc2Diffs, jsCompare.DepDoc2.DepPath, jsCompare.DepDoc1.DepPath, depObj2, depObj1, docDep2, docDep1)

			if fsMerge.sketchPath != nil && fsMerge.exportPath != nil {
				pages, artboards := skDiff2.extractExportList()
				if len(artboards) > 0 {
					launchSketchToolExport(*fsMerge.sketchPath, "artboards", *fsMerge.sketchFileV2, strings.Join(artboards, ","), *fsMerge.exportPath + string(os.PathSeparator) + "v2")
				} else if len(pages) > 0 {
					launchSketchToolExport(*fsMerge.sketchPath, "pages", *fsMerge.sketchFileV2, strings.Join(pages, ","), *fsMerge.exportPath + string(os.PathSeparator) + "v2")
				}
			}

			if len(jsCompare.Doc2Diffs) > 0 {
				fsMerge.MergeActions[i].FileDiff.Doc2Diffs = map[string]interface{}{ "nice_diff" : skDiff2 }
			}
		} else {
			fsMerge.MergeActions[i].Action = ADD
		}
	}

	return nil
	//return json.MarshalIndent(fsMerge, "", "  ")
}

//Compares two Sketch documents
func (fsMerge * FileStructureMerge) CompareDocuments(workingDirV1, workingDirV2 string) error {
	for i := range fsMerge.MergeActions {
		fileName := fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt
		if filepath.Ext(strings.ToLower(fileName)) == ".json" {
			result, err := CompareJSON(workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt,  workingDirV2 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)
			if err != nil {
				return err
			}
			fsMerge.MergeActions[i].FileDiff = *result

			//if fsMerge.MergeActions[i].Action != MERGE {
			//	result.FileDependendObject(fsMerge.MergeActions[i].Action, SOURCE, fsMerge.MergeActions[i].FileKey, fileName)
			//	result.FileDependendObject(fsMerge.MergeActions[i].Action, DESTINATION, fsMerge.MergeActions[i].FileKey, fileName)
			//}
		}
	}

	return nil
}


func (jsc * JsonStructureCompare) traverseDependentObjects(objKey string, docTree * interface{}, dep * DependentObjects, jsonpath string) bool  {

	switch itemType := (*docTree).(type) {
	case map[string]interface{}:
		for key, item := range (*docTree).(map[string]interface{}) {
			if !jsc.traverseDependentObjects(key, &item, dep, jsonpath + `["`+key+`"]`) {
				dep.AddDependentObject(key, item, jsonpath )
			}
		}
	case []interface{}:
		for i:=range (*docTree).([]interface{}){
			if !jsc.traverseDependentObjects(objKey, &((*docTree).([]interface{})[i]), dep, jsonpath + `[` + strconv.Itoa(i) + `]` ) {
				dep.AddDependentObject(objKey, (*docTree).([]interface{})[i], jsonpath )
			}
		}
	default:
		_=itemType
		return false
	}

	return true
}

//Adds dependencies to given objKey
func (jsc * JsonStructureCompare) AddDependentObjects(objKey string, docTree * interface{}, dep * DependentObjects, jsonpath string) bool {
	if !!jsc.traverseDependentObjects(objKey, docTree, dep, jsonpath) {

		return dep.AddDependentObject(objKey, *docTree, jsonpath )
	}
	return false
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
				jsc.AddDependentObjects(key, &item, jsc.DepDoc1, pathDoc1  + `["` + key + `"]`)
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

	//find all dependent properties
	if hasDiff {
		for key, item := range doc2TreeMap {
			if key != jsc.ObjectKeyName {
				jsc.AddDependentObjects(key, &item, jsc.DepDoc2, pathDoc2  + `["` + key + `"]`)
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
	keysDoc1 := make(map[string]interface{}, len(doc1TreeArray))
	keysDoc2 := make(map[string]interface{}, len(doc2TreeArray))

	//put doc1 indeces to map by given key
	//there could be elements with the same object id
	for index, item := range doc1TreeArray {
		if itemTreeMap, isItemMap := item.(map[string]interface{}); isItemMap {
			if objectId, ok := itemTreeMap[objectKeyName]; ok {
				var arr []int
				if value, ok := keysDoc1[objectId.(string)]; !ok {
					arr = make([]int, 0)
				} else {
					arr = value.([]int)
				}
				keysDoc1[objectId.(string)] = append(arr, index)
			}
		}

	}

	//put doc2 indeces to map by given key
	//there could be elements with the same object id
	for index, item := range doc2TreeArray {
		if itemTreeMap, isItemMap := item.(map[string]interface{}); isItemMap {
			if objectId, ok := itemTreeMap[objectKeyName]; ok {
				var arr []int
				if value, ok := keysDoc2[objectId.(string)]; !ok {
					arr = make([]int, 0)
				} else {
					arr = value.([]int)
				}
				keysDoc2[objectId.(string)] = append(arr, index)
			}
		}

	}



	//build index change map for doc1
	for key, _idxDoc1 := range keysDoc1 {
		if _idxDoc2, ok := keysDoc2[key]; ok {
			//NOTE: indeces could be different
			idxDoc1 := _idxDoc1.([]int)
			idxDoc2 := _idxDoc2.([]int)
			j:=0
			for i := range idxDoc1 {
				if j < len(idxDoc2) {
					doc1Changes[idxDoc1[i]] = idxDoc2[j]
					j++
				} else {
					doc1Changes[idxDoc1[i]] = -1
				}
			}

		} else {
			idxDoc1 := _idxDoc1.([]int)
			for i := range idxDoc1 {
				doc1Changes[idxDoc1[i]] = -1
			}
		}
	}

	//build index change map for doc2
	for key, _idxDoc2 := range keysDoc2 {
		if _idxDoc1, ok := keysDoc1[key]; ok {
			//NOTE: indeces could be different
			//doc2Changes[idxDoc2] = idxDoc1
			idxDoc2 := _idxDoc2.([]int)
			idxDoc1 := _idxDoc1.([]int)
			j:=0
			for i := range idxDoc2 {
				if j < len(idxDoc1) {
					doc2Changes[idxDoc2[i]] = idxDoc1[j]
					j++
				} else {
					doc2Changes[idxDoc2[i]] = -1
				}
			}
		} else {
			idxDoc2 := _idxDoc2.([]int)
			for i := range idxDoc2 {
				doc2Changes[idxDoc2[i]] = -1
			}
		}

		//log.Println("doc2Changes:" + strconv.Itoa(len(keysDoc2)) +":" + strconv.Itoa(idxDoc2) +":"+ strconv.Itoa(doc2Changes[idxDoc2]) )
	}

	//log.Printf(" key1: %v , %v\nkey2: %v , %v\n", keysDoc1,doc1Changes, keysDoc2,  doc2Changes)

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

func artboardAddress(doc1TreeArray []interface{}, idxDoc1 int) string {
	alternateAddr1 := strconv.Itoa(idxDoc1)

	if idxDoc1 != -1 {
		if doc1Dict, isDict := doc1TreeArray[idxDoc1].(map[string]interface{}); isDict {
			doc1ObjID, isObjID := doc1Dict["do_objectID"].(string)
			doc1ObjClass := doc1Dict["_class"]

			if isObjID && (doc1ObjClass == "artboard" || doc1ObjClass == "symbolMaster") {
				alternateAddr1 = "@do_objectID='" + doc1ObjID + "'"
			}
		}
	}

	return alternateAddr1
}

//Compare each element in array node
func (jsc * JsonStructureCompare) CompareSlices(doc1TreeArray []interface{}, doc2TreeArray []interface{}, pathDoc1 string, pathDoc2 string) (string, string, bool) {
	//defer timeTrack(time.Now(), "CompareSlices " + path)

	doc1Changes, doc2Changes := CompareSequence(jsc.ObjectKeyName, doc1TreeArray, doc2TreeArray)

	doc1ChangesCopy := deepcopy.Copy(doc1Changes).(map[int]int)
	doc2ChangesCopy := deepcopy.Copy(doc2Changes).(map[int]int)
	//fmt.Printf("docChanges: %v , %v\n", doc1Changes, doc2Changes)


	//go thru array associations with the same objectKeyName for doc1
	for idxDoc1, idxDoc2 := range doc1Changes {

		alternateAddr1 := artboardAddress(doc1TreeArray, idxDoc1)
		alternateAddr2 := artboardAddress(doc2TreeArray, idxDoc2)

		jsonpathDoc1 := strings.Join([]string{pathDoc1, "[", alternateAddr1, "]"}, "")
		jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", alternateAddr2, "]"}, "")
		if idxDoc1 == idxDoc2 {
			//remove similar indeces
			delete(doc1ChangesCopy, idxDoc1)
		}

		if idxDoc2 == -1 {

			//if there is no such element in doc2 array
			jsc.addDoc2Diff("-" + jsonpathDoc1, "","CompareSlices")
			jsc.addDoc1Diff("+" + jsonpathDoc1, pathDoc2, "CompareSlices")
			jsc.DepDoc2.AddDependentPath( "-" + jsonpathDoc1, "^" + pathDoc1, "^" + pathDoc2)
			jsc.AddDependentObjects("", &(doc1TreeArray[idxDoc1]), jsc.DepDoc1, jsonpathDoc1)
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

		alternateAddr2 := artboardAddress(doc2TreeArray, idxDoc2)

		jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", alternateAddr2, "]"}, "")

		if idxDoc1 == -1 {

			//if there is no such element in doc1 array
			jsc.addDoc1Diff("-" + jsonpathDoc2, "", "CompareSlices")
			jsc.addDoc2Diff("+" + jsonpathDoc2, pathDoc1, "CompareSlices")
			jsc.DepDoc1.AddDependentPath("-" + jsonpathDoc2, "^" + pathDoc2, "^" + pathDoc1)
			jsc.AddDependentObjects("", &(doc2TreeArray[idxDoc2]), jsc.DepDoc2, jsonpathDoc2)
		}
	}


	//if it's not a layers array compare it property by property
	if len(doc1Changes) == 0 && len(doc2Changes) == 0 {
		if !reflect.DeepEqual(doc1TreeArray, doc2TreeArray) {
			jsc.addDoc1Diff(pathDoc1, pathDoc2, "CompareSlices")
			jsc.addDoc2Diff(pathDoc2, pathDoc1, "CompareSlices")

			for idxDoc1 := range doc1TreeArray {
				jsonpathDoc1 := strings.Join([]string{pathDoc1, "[", strconv.Itoa(idxDoc1), "]"}, "")
				jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", strconv.Itoa(idxDoc1), "]"}, "")

				if idxDoc1 >= len(doc2TreeArray) {
					//jsc.addDoc2Diff("-" + jsonpathDoc1, "","CompareSlices")
					//jsc.AddDoc1Diff("+" + jsonpathDoc1, pathDoc2, "CompareSlices")
					jsc.AddDependentObjects("", &(doc1TreeArray[idxDoc1]), jsc.DepDoc1, jsonpathDoc1)
					continue
				}

				jsc.AddDependentObjects("", &(doc1TreeArray[idxDoc1]), jsc.DepDoc1, jsonpathDoc1)
				jsc.AddDependentObjects("", &(doc2TreeArray[idxDoc1]), jsc.DepDoc2, jsonpathDoc2)

				//if __jsonpath1, __jsonpath2, ok := jsc.CompareDocuments(&(doc1TreeArray[idxDoc1]), &(doc2TreeArray[idxDoc1]), jsonpathDoc1, jsonpathDoc2); !ok {
				//	jsc.AddDoc1Diff(__jsonpath1, __jsonpath2, "CompareSlices")
				//	jsc.addDoc2Diff(__jsonpath2, __jsonpath1, "CompareSlices")
				//}
			}

			if len(doc2TreeArray) > len(doc1TreeArray) {
				idxStart := len(doc1TreeArray)
				idxEnd := len(doc2TreeArray)

				for idxDoc2 := idxStart; idxDoc2 < idxEnd; idxDoc2++ {
					jsonpathDoc2 := strings.Join([]string{pathDoc2, "[", strconv.Itoa(idxDoc2), "]"}, "")
					if idxDoc2 >= len(doc1TreeArray) {
						//jsc.AddDoc1Diff("-"+jsonpathDoc2, "", "CompareSlices")
						//jsc.addDoc2Diff("+"+jsonpathDoc2, pathDoc1, "CompareSlices")
						jsc.AddDependentObjects("", &(doc2TreeArray[idxDoc2]), jsc.DepDoc2, jsonpathDoc2)
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
	defer TimeTrack(time.Now(), "Compare" + path)
	jsc.CompareProperties(doc1TreeMap, doc2TreeMap, path, path)
}

func NewJsonStructureCompare() *JsonStructureCompare {
	return &JsonStructureCompare{make(map[string]interface{}),
				     make(map[string]interface{}),
						make(map[string]interface{}),
						make(map[string]interface{}),
						"do_objectID",
							&DependentObjects{ SOURCE,make(map[string]interface{}), make(map[string]interface{})},
							&DependentObjects{ DESTINATION, make(map[string]interface{}), make(map[string]interface{})}}
}

func CreateJsonStructureCompare() JsonStructureCompare {
	return JsonStructureCompare{make(map[string]interface{}),
				     make(map[string]interface{}),
				     make(map[string]interface{}),
				     make(map[string]interface{}),
				     "do_objectID",
				     &DependentObjects{ SOURCE,make(map[string]interface{}), make(map[string]interface{})},
				     &DependentObjects{ DESTINATION, make(map[string]interface{}), make(map[string]interface{})}}
}

func BuildFileAction(fileAction FileActionType, fileName string) string {
	switch fileAction {
	case ADD:
		return "A~" + fileName + "~$"
	case DELETE:
		return "D~" + fileName + "~$"
	}
	return ""
}

func (jsc * JsonStructureCompare) FileDependendObject(fileAction FileActionType, docType DocumentType, fileKey, fileName string) {

	if !strings.HasPrefix(fileName, "pages" + string(os.PathSeparator)) {
		return
	}

	dep := []interface{}{DependentObj{JsonPath:"$"}}

	if docType == SOURCE {
		jsc.DepDoc1.DepObj[strings.TrimPrefix(fileKey, "pages" + string(os.PathSeparator))] = dep
		//fileActionPath := BuildFileAction(fileAction, fileName)
		//jsc.AddDoc1Diff(fileActionPath, fileActionPath)
		//
		//if fileAction == ADD {
		//	jsc.addDoc1Diff("$", "$", "FileDependendObject")
		//} else if fileAction == DELETE {
		//	jsc.addDoc1Diff("-$", "", "FileDependendObject")
		//}
	} else if docType == DESTINATION {
		jsc.DepDoc2.DepObj[strings.TrimPrefix(fileKey, "pages" + string(os.PathSeparator))] = dep

		//if fileAction == ADD {
		//	jsc.addDoc2Diff("-$", "", "FileDependendObject")
		//} else if fileAction == DELETE {
		//	jsc.addDoc2Diff("$", "$", "FileDependendObject")
		//}
		//if fileAction == ADD {
		//	fileAction = DELETE
		//} else if fileAction == DELETE {
		//	fileAction = ADD
		//}
		//fileActionPath := BuildFileAction(fileAction, fileName)
		//jsc.addDoc2Diff(fileActionPath, fileActionPath)
	}
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


	Test(flag.Arg(0) + "/document.json", flag.Arg(1) + "/document.json")

}

