// Copyright 2017 Sergey Fedoseev. All rights reserved.
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
	"bufio"
	"io"
	"fmt"
	"time"
	"path"
	"os/exec"
)

type PageFilter struct {
	FilterPageID string
	FilterArtboardID string
	FilterClassName string
}

type DumpInfo struct {
	ObjectsMap map[string]interface{}
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


func WriteToFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0755 )
}

func (di * DumpInfo) Traverse(docTree * interface{})  {

	switch itemType := (*docTree).(type) {
	case map[string]interface{}:
		prop := (*docTree).(map[string]interface{})
		if objectId, ok := prop["objectID"].(string); ok {
			log.Printf("store ref: %v", objectId)
			di.ObjectsMap[objectId] = prop
		}
		if _layers, ok := prop["layers"]; ok {
			layers := _layers
			di.Traverse(&layers)
		} else if _layers, ok := prop["pages"]; ok {
			layers := _layers
			di.Traverse(&layers)
		}

	case []interface{}:
		for i:=range (*docTree).([]interface{}){
			di.Traverse( &((*docTree).([]interface{})[i]))
		}
	default:
		_=itemType
	}
}

func (di * DumpInfo) BuildDumpFileHash(dumpFileName string) {
	if dumpDoc, err := readJSON(dumpFileName); err == nil {
		var docTree interface{} = dumpDoc
		di.Traverse(&docTree)
	}
}

func launchSketchToolDump(sketchApp, sketchFile, dumpFile string) error {
	cmdArgs := []string{"dump", sketchFile}
	cmd := exec.Command(sketchApp, cmdArgs...)
	dump, err := os.Create(dumpFile)

	if (err != nil) {
		return err
	}
	defer dump.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(dump)
	defer writer.Flush()
	err = cmd.Start()
	if (err != nil) {
		return err
	}
	io.Copy(writer, stdout)
	cmd.Wait()
	return err

}



//2-way diff
func ProcessFileDiff(sketchFileV1 string, sketchFileV2 string) (*FileStructureMerge, error) {

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

	if err := fsMerge.CompareDocuments(workingDirV1, workingDirV2); err != nil {
		return nil, err
	}

	if _, _, _,  err := ProceedDependencies(workingDirV1, workingDirV2, fsMerge.MergeActions); err!=nil {
		return nil, err
	}


	err := fsMerge.ProduceDiffWithDependencies()
	return fsMerge, err

}

//2-way file difference
func ProcessNiceFileDiff(sketchFileV1 string, sketchFileV2 string, hasInfo bool, dumpFile1 * string, dumpFile2 * string, sketchPath * string, exportPath * string) (*FileStructureMerge, error) {
	defer TimeTrack(time.Now(), "ProcessNiceFileDiff " + sketchFileV1)

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

	sketchFileSrc := sketchFileV1
	sketchFileDst := sketchFileV2

	if sketchPath != nil && dumpFile1 != nil && dumpFile2 != nil {

		if isSrcDir {
			sketchFileSrc := strings.TrimSuffix( sketchFileV1, strings.TrimPrefix(sketchFileV1, filepath.Dir(sketchFileV1)))
			Zipit(workingDirV1, sketchFileSrc)
		}
		if isDstDir {
			sketchFileDst := strings.TrimSuffix( sketchFileV2, strings.TrimPrefix(sketchFileV2, filepath.Dir(sketchFileV2)))
			Zipit(workingDirV2, sketchFileDst)
		}

		if err := launchSketchToolDump(*sketchPath, sketchFileSrc, *dumpFile1); err != nil {
			return nil, err
		}

		if err := launchSketchToolDump(*sketchPath, sketchFileDst, *dumpFile2); err != nil {
			return nil, err
		}
	}

	baseFileStruct, newFileStruct := ExtractSketchDirStruct(workingDirV1, workingDirV2)

	fsMerge := new(FileStructureMerge)
	fsMerge.FileSetChange(newFileStruct, baseFileStruct)
	fsMerge.hasInfo = hasInfo
	fsMerge.sketchFileV1 = &sketchFileSrc
	fsMerge.sketchFileV2 = &sketchFileDst

	fsMerge.exportPath = exportPath
	fsMerge.sketchPath = sketchPath

	if exportPath != nil {
		os.MkdirAll(*exportPath + string(os.PathSeparator) + "v1", 0777)
		os.MkdirAll(*exportPath + string(os.PathSeparator) + "v2", 0777)
	}

	if dumpFile1 != nil && dumpFile2 != nil {
		di1 := DumpInfo{make(map[string]interface{})}
		di1.BuildDumpFileHash(*dumpFile1)
		fsMerge.dump1 = &di1

		di2 := DumpInfo{make(map[string]interface{})}
		di2.BuildDumpFileHash(*dumpFile2)
		fsMerge.dump2 = &di2
	}

	if err := fsMerge.CompareDocuments(workingDirV1, workingDirV2); err != nil {
		return nil, err
	}

	if depObj1, depObj2, fileMergeDoc, err := ProceedDependencies(workingDirV1, workingDirV2, fsMerge.MergeActions); err == nil {
		return fsMerge, fsMerge.ProduceNiceDiffWithDependencies("local", workingDirV1, workingDirV2, depObj1, depObj2, fileMergeDoc)
	} else  {
		return nil, err
	}

}

//#-way file difference
func ProcessNiceFileDiff3Way(sketchFileV0, sketchFileV1, sketchFileV2 string) (*FileStructureMerge, error) {
	defer TimeTrack(time.Now(), "ProcessNiceFileDiff3Way " + sketchFileV0)

	isSrcDir := false
	isDstDir := false
	isBaseDir :=false

	sketchFileV0Info, errv0 := os.Stat(sketchFileV0)

	if errv0 != nil {
		return nil, errv0
	}

	isBaseDir = sketchFileV0Info.IsDir()

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

	workingDirV0, err0 := prepareWorkingDir(!isBaseDir)
	if err0!=nil {
		return nil, err0
	}
	defer removeWorkingDir(workingDirV0, isBaseDir)

	if isBaseDir {
		workingDirV0 = sketchFileV0
	}

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

	if !isBaseDir {
		if err := Unzip(sketchFileV0, workingDirV0); err != nil {
			return nil, err
		}
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

	baseFileStruct1, newFileStruct1 := ExtractSketchDirStruct(workingDirV1, workingDirV0)
	fsMerge1 := new(FileStructureMerge)
	fsMerge1.FileSetChange(newFileStruct1, baseFileStruct1)

	baseFileStruct2, newFileStruct2 := ExtractSketchDirStruct(workingDirV2, workingDirV0)
	fsMerge2 := new(FileStructureMerge)
	fsMerge2.FileSetChange(newFileStruct2, baseFileStruct2)

	if err := fsMerge1.CompareDocuments(workingDirV1, workingDirV0); err != nil {
		return nil, err
	}
	depObj11, depObj12, fileMergeDoc1, err := ProceedDependencies(workingDirV1, workingDirV0, fsMerge1.MergeActions)
	if err!=nil {
		return nil, err
	}

	if err := fsMerge2.CompareDocuments(workingDirV2, workingDirV0); err != nil {
		return nil, err
	}
	depObj21, depObj22, fileMergeDoc2, err := ProceedDependencies(workingDirV2, workingDirV0, fsMerge2.MergeActions)
	if err != nil {
		return nil, err
	}

	return ProcessFileStructures3Way(workingDirV0, workingDirV1, workingDirV2, fsMerge1, fsMerge2, depObj11, depObj12, depObj21, depObj22, fileMergeDoc1, fileMergeDoc2)

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
	delActionsArr := GetSortedDescDelActions(deleteActions)
	//for key, _ := range deleteActions {
	for i:= range delActionsArr {
		mergeDoc.MergeByJSONPath("", delActionsArr[i], MarkElementToDelete) //
	}

	seqDiffKeyArr, seqDiffItemArr := GetSortedDescActions(seqDiff)
	//Perform sorting
	//for key, item := range seqDiff {
	for i:= range seqDiffKeyArr {
		mergeDoc.MergeSequenceByJSONPath(objectKeyName, seqDiffKeyArr[i], seqDiffItemArr[i])
	}

	//second iteration will delete
	//TODO: optimize second call
	//for key, _ := range deleteActions {
	for i:= range delActionsArr {
		mergeDoc.MergeByJSONPath("", delActionsArr[i], DeleteMarked)
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

//Performs 3-way merge
func (mergeDoc * MergeDocuments) mergeChanges(srcFilePath string, dstFilePath string, fileName string, docDiffs map[string]interface{} , deleteActions, seqDiff map[string]string) error {

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
			if err := mergeDoc.MergeByJSONPath(key, item.(string), DeleteMarked); err != nil {
				return err
			}
		}
	}

	return nil
}

//That function will only set value of an element in a sequence to nil
func (mergeDoc * MergeDocuments) mergeDeletions(deleteActions map[string]string) error {
	//Perform all deletions
	//First iteration will only mark to delete
	delActionsArr := GetSortedDescDelActions(deleteActions)
	//for key, _ := range deleteActions {
	for i:= range delActionsArr {
		if err := mergeDoc.MergeByJSONPath("", delActionsArr[i] , MarkElementToDelete); err != nil {
			return err
		}
	}

	return nil
}

//This will perform deletion of elements which ara nil
func (mergeDoc * MergeDocuments) mergeConfirmDeletions(deleteActions map[string]string) error {
	delActionsArr := GetSortedDescDelActions(deleteActions)
	//second iteration will delete
	//TODO: optimize second call
	//for key, _ := range deleteActions {
	for i:= range delActionsArr {
		if err := mergeDoc.MergeByJSONPath("", delActionsArr[i], DeleteMarked); err != nil {
			continue
		}
	}

	return nil
}

//Reorder sequences in destination document so it will close to source sequence
func (mergeDoc * MergeDocuments) mergeSequentions(objectKeyName string, seqDiff map[string]string) error {
	seqDiffKeyArr, seqDiffItemArr := GetSortedDescActions(seqDiff)
	//Perform sorting
	//for key, item := range seqDiff {
	for i:= range seqDiffKeyArr {
		if err := mergeDoc.MergeSequenceByJSONPath(objectKeyName, seqDiffKeyArr[i], seqDiffItemArr[i]); err != nil {
			return err
		}
	}

	return nil
}

func (fl * PageFilter) FilteroutContent(workingDirV1, workingDirV2, fileKey string) error {

	if !strings.HasPrefix(fileKey, "pages" + string(os.PathSeparator)) {
		updateFile(workingDirV1, workingDirV2, fileKey)
		return nil
	}

	srcFilePath := workingDirV1 + string(os.PathSeparator) + fileKey
	dstFilePath := workingDirV2 + string(os.PathSeparator) + fileKey
	baseFileName := path.Base(dstFilePath)

	targetDir := strings.TrimSuffix(dstFilePath, baseFileName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0777)
	}


	fileDoc1, eDoc1 := ioutil.ReadFile(srcFilePath)
	if eDoc1 != nil {
		return eDoc1
	}

	var result1 map[string]interface{}
	if eDoc1 == nil {
		var decoder1= json.NewDecoder(bytes.NewReader(fileDoc1))
		decoder1.UseNumber()

		if err := decoder1.Decode(&result1); err != nil {
			return err
		}
	}

	layers, hasLayers := result1["layers"].([]interface{})
	newLayers := make([]interface{}, 0)

	if hasLayers {
		for i := range layers {
			layer, isLayer := layers[i].(map[string]interface{})
			if !isLayer {
				continue
			}
			artboardID, hasID := layer["do_objectID"].(string)
			className, hasClass := layer["_class"].(string)

			if !hasID && !hasClass {
				continue
			}

			if fl.FilterArtboardID == artboardID ||
			   fl.FilterClassName == className || (fl.FilterArtboardID == fl.FilterPageID && result1["do_objectID"] == fl.FilterPageID) {
				newLayers = append(newLayers, layer)
			}
		}
	}

	if len(newLayers) > 0 || result1["do_objectID"] != fl.FilterPageID {
		result1["layers"] = newLayers
	}

	//if result1["do_objectID"] != fl.FilterPageID {
	//	updateFile(workingDirV1, workingDirV2, fileKey)
	//	return nil
	//}

	data, err := json.Marshal(result1)

	if err != nil {
		return err
	}

	return WriteToFile(dstFilePath, data)
}

//This functions overwrites file
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

//Create FileMerge structure for given action
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

//Converts jsopaths with file references to plain mergeActions structures per file
//this is required to process dependencies
func buildFileActions(workingDirV1 string, workingDirV2 string, mergeJSON FileStructureMerge) ([]FileMerge) {

	//New merge actions array
	newMergeActions := make([]FileMerge, 0)

	//We will use maps in order to group jsonpaths per file
	mergeMap := make(map[string]interface{})

	//go thru all merge actions (usually pages)
	for i := range mergeJSON.MergeActions {

		//Go thru all go diffs
		for key, item := range mergeJSON.MergeActions[i].FileDiff.Doc1Diffs {

			//TODO: Implement reverse actions
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
			//Extract only file name
			fileKey := ReadFileKey(key)
			//Extract file name with action 'A' add or 'D' delete
			fileAction := ReadFileAction(key)

			//if we have file name in jsonpath use it as key
			//that means that given jsonpath belons to other file
			//so we should put it into other merge action structure
			if fileKey != "" {
				fileName = fileKey
			}

			//Find marge action for given fileName
			//and create new FileMerge if there is no such value in the map
			fileMerge, ok := mergeMap[fileName].(FileMerge)
			if !ok {


				fileMerge = createMergeAction(fileName, fileAction)
				mergeMap[fileName] = fileMerge

			}

			//Determine first letter of action
			if strings.HasPrefix(key,"A") {
				fileMerge.Action = ADD
				mergeMap[fileName] = fileMerge
				continue
			} else if strings.HasPrefix(key,"D") {
				fileMerge.Action = DELETE
				mergeMap[fileName] = fileMerge
				continue
			}

			//remove all file references and set as normal json
			fileMerge.FileDiff.Doc1Diffs[FlatJsonPath(key, false)] = FlatJsonPath(item.(string), false)
		}


	}

	//add all values into array
	for _, value := range mergeMap {
		newMergeActions = append(newMergeActions, value.(FileMerge))
	}

	return newMergeActions
}

func (fm * FileMerge) PerformMergeChanges(workingDirV0, workingDirV1 string, mergeMethod func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error ) error {

	fileName := fm.FileKey + fm.FileExt

	if !fm.IsDirectory && fm.Action == ADD {
		updateFile(workingDirV1, workingDirV0, fileName)
		return nil
	}

	if fm.FileDiff.Doc1Diffs == nil {
		return nil
	}

	srcFilePath := workingDirV1 + string(os.PathSeparator) + fileName
	dstFilePath := workingDirV0 + string(os.PathSeparator) + fileName

	//get files jsons
	jsonDoc1, jsonDoc2, err := decodeMergeFiles(srcFilePath, dstFilePath)

	if err != nil {
		return err
	}

	//Create merge documets structure
	mergeDoc := MergeDocuments{jsonDoc1, jsonDoc2}

	if err := mergeMethod(srcFilePath, dstFilePath,
		fileName,
		fm, &mergeDoc); err!=nil {
		return err
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

func mergeActions3Way(workingDirV0, workingDirV1, workingDirV2 string, mergeJSON1, mergeJSON2 FileStructureMerge) error {


	mergeActionsLocal := buildFileActions(workingDirV1, workingDirV0, mergeJSON1)
	mergeActionsRemote := buildFileActions(workingDirV2, workingDirV0, mergeJSON2)

	info1, _ := json.MarshalIndent(mergeActionsLocal, "", "  ")
	fmt.Printf("local: %v\n", string(info1))

	info2, _ := json.MarshalIndent(mergeActionsRemote, "", "  ")
	fmt.Printf("remote: %v\n", string(info2))

	//We will perform delete operations after isertions to avoid
	//actions on the same index
	deleteMerges1 := make(map[string]interface{})
	deleteMerges2 := make(map[string]interface{})
	seqMerges1 := make(map[string]interface{})
	seqMerges2 := make(map[string]interface{})


	//Perform property change actions
	for i := range mergeActionsLocal {

		if err := mergeActionsLocal[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			seqDiff := make(map[string]string)
			deleteActions := make(map[string]string)
			_err := mergeDoc.mergeChanges(srcFilePath, dstFilePath,
				fileName,
				fm.FileDiff.Doc1Diffs, deleteActions, seqDiff)
			deleteMerges1[fileName] = deleteActions
			seqMerges1[fileName] = seqDiff
			return _err
		} ); err != nil {
			continue
			//return err
		}

	}

	for i := range mergeActionsRemote {

		if err := mergeActionsRemote[i].PerformMergeChanges(workingDirV0, workingDirV2, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			deleteActions :=  deleteMerges1[fileName]
			if deleteActions == nil {
				deleteActions = make(map[string]string)
			}

			seqDiff := make(map[string]string)

			_err := mergeDoc.mergeChanges(srcFilePath, dstFilePath,
				fileName,
				fm.FileDiff.Doc1Diffs, deleteActions.(map[string]string), seqDiff)
			deleteMerges2[fileName] = deleteActions
			seqMerges2[fileName] = seqDiff
			return _err
		}); err != nil {
			//return err
			continue
		}

	}

	//Perform delete actions
	for i := range mergeActionsLocal {

		if err := mergeActionsLocal[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			deleteActions :=  deleteMerges1[fileName]
			if deleteActions == nil {
				return nil
			}
			return mergeDoc.mergeDeletions(deleteActions.(map[string]string))
		} ); err != nil {
			continue
			//return err
		}

	}

	//Perform delete actions
	for i := range mergeActionsRemote {

		if err := mergeActionsRemote[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			deleteActions :=  deleteMerges2[fileName]
			if deleteActions == nil {
				return nil
			}
			return mergeDoc.mergeDeletions(deleteActions.(map[string]string))
		} ); err != nil {
			continue
			//return err
		}

	}

	//Perform sequence change actions
	for i := range mergeActionsLocal {

		if err := mergeActionsLocal[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			seqDiff := seqMerges1[fileName]
			if seqDiff == nil {
				return nil
			}
			return mergeDoc.mergeSequentions(fm.FileDiff.ObjectKeyName, seqDiff.(map[string]string))
		} ); err != nil {
			continue
			//return err
		}

	}

	for i := range mergeActionsRemote {

		if err := mergeActionsRemote[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			seqDiff := seqMerges2[fileName]
			if seqDiff == nil {
				return nil
			}
			return mergeDoc.mergeSequentions(fm.FileDiff.ObjectKeyName, seqDiff.(map[string]string))
		} ); err != nil {
			continue
			//return err
		}

	}

	//Perform confirm delete actions
	for i := range mergeActionsLocal {

		if err := mergeActionsLocal[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			deleteActions :=  deleteMerges1[fileName]
			if deleteActions == nil {
				return nil
			}
			return mergeDoc.mergeConfirmDeletions(deleteActions.(map[string]string))
		} ); err != nil {
			//we will ignore this kind of errors
			//if we are using adreesing $["layers"][@do_objectID='59E11126-A64E-4325-9832-1F4D625C272B']
			//there wil be nil objects/layers because they are marked to delete
			continue
			//return err
		}

	}

	for i := range mergeActionsRemote {

		if err := mergeActionsRemote[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			deleteActions :=  deleteMerges2[fileName]
			if deleteActions == nil {
				return nil
			}
			return mergeDoc.mergeConfirmDeletions(deleteActions.(map[string]string))
		} ); err != nil {
			//we will ignore this kind of errors
			//if we are using adreesing $["layers"][@do_objectID='59E11126-A64E-4325-9832-1F4D625C272B']
			//there wil be nil objects/layers because they are marked to delete
			continue
			//return err
		}

	}

	return nil
}

func mergeActions(workingDirV1 string, workingDirV2 string, mergeJSON FileStructureMerge, fl * PageFilter) error {
	defer TimeTrack(time.Now(), "mergeActions " + workingDirV1)

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
			if fl == nil {
				updateFile(workingDirV1, workingDirV2, fileName)
			} else {
				fl.FilteroutContent(workingDirV1, workingDirV2, fileName)
			}
			continue
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

func ProcessFileMerge(mergeFileName string, sketchFileV1 string, sketchFileV2 string, outputDir string, filter * PageFilter) error {
	defer TimeTrack(time.Now(), "ProcessFileMerge " + sketchFileV1)

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

	if err := mergeActions(workingDirV1, workingDirV2, mergeJSON, filter); err != nil  {
		return err
	}

	if !isDstDir {
		sketchFile := outputDir + string(os.PathSeparator) + strings.TrimPrefix(sketchFileV2, filepath.Dir(sketchFileV2))
		//similar to zip -y -r -q -8 testVCS2.sketch ./pages/ ./previews/ document.json meta.json user.json
		Zipit(workingDirV2, sketchFile)
	}

	return nil

}

func Process3WayFileMerge(mergeFileName1, mergeFileName2 string, sketchFileV0, sketchFileV1, sketchFileV2 string, outputDir string) error {
	defer TimeTrack(time.Now(), "Process3WayFileMerge " + sketchFileV0)

	isSrcDir := false
	isDstDir := false
	isBaseDir :=false

	sketchFileV0Info, errv0 := os.Stat(sketchFileV0)

	if errv0 != nil {
		return errv0
	}

	isBaseDir = sketchFileV0Info.IsDir()

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

	workingDirV0, err0 := prepareWorkingDir(!isBaseDir)
	if err0!=nil {
		return err0
	}
	defer removeWorkingDir(workingDirV0, isBaseDir)

	if isBaseDir {
		workingDirV0 = sketchFileV0
	}

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

	if !isBaseDir {
		if err := Unzip(sketchFileV0, workingDirV0); err != nil {
			return err
		}
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

	mergeFile1, err := ioutil.ReadFile(mergeFileName1)
	if err != nil {
		return err
	}

	var mergeJSON1 FileStructureMerge
	var decoder1 = json.NewDecoder(bytes.NewReader(mergeFile1))
	decoder1.UseNumber()

	if err := decoder1.Decode(&mergeJSON1); err != nil {
		return  err
	}

	mergeFile2, err := ioutil.ReadFile(mergeFileName2)
	if err != nil {
		return err
	}

	var mergeJSON2 FileStructureMerge
	var decoder2 = json.NewDecoder(bytes.NewReader(mergeFile2))
	decoder2.UseNumber()

	if err := decoder2.Decode(&mergeJSON2); err != nil {
		return  err
	}

	if err := mergeActions3Way(workingDirV0, workingDirV1, workingDirV2, mergeJSON1, mergeJSON2); err != nil  {
		return err
	}

	if !isBaseDir {
		sketchFile := outputDir + string(os.PathSeparator) + strings.TrimPrefix(sketchFileV0, filepath.Dir(sketchFileV0))
		//similar to zip -y -r -q -8 testVCS2.sketch ./pages/ ./previews/ document.json meta.json user.json
		Zipit(workingDirV0, sketchFile)
	}

	return nil

}



