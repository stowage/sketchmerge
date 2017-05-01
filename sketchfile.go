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
	_"time"
	"path"
)



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

	if _, _, err := ProceedDependencies(workingDirV1, workingDirV2, fsMerge.MergeActions); err!=nil {
		return nil, err
	}


	err := fsMerge.ProduceDiffWithDependencies()
	return fsMerge, err

}

func ProcessNiceFileDiff(sketchFileV1 string, sketchFileV2 string) (*FileStructureMerge, error) {

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

	if _, _, err := ProceedDependencies(workingDirV1, workingDirV2, fsMerge.MergeActions); err!=nil {
		return nil, err
	}

	err := fsMerge.ProduceNiceDiffWithDependencies("local", workingDirV1, workingDirV2)
	return fsMerge, err

}

func ProcessNiceFileDiff3Way(sketchFileV0, sketchFileV1, sketchFileV2 string) (*FileStructureMerge, error) {

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

	if _, _, err := ProceedDependencies(workingDirV1, workingDirV0, fsMerge1.MergeActions); err!=nil {
		return nil, err
	}

	if err := fsMerge2.CompareDocuments(workingDirV2, workingDirV0); err != nil {
		return nil, err
	}

	if _, _, err := ProceedDependencies(workingDirV2, workingDirV0, fsMerge2.MergeActions); err!=nil {
		return nil, err
	}

	return ProcessFileStructures3Way(workingDirV0, workingDirV1, workingDirV2, fsMerge1, fsMerge2)

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

func (mergeDoc * MergeDocuments) mergeDeletions(deleteActions map[string]string) error {
	//Perform all deletions
	//First iteration will only mark to delete
	for key, _ := range deleteActions {
		if err := mergeDoc.MergeByJSONPath("", key, MarkElementToDelete); err != nil {
			return err
		}
	}

	//second iteration will delete
	//TODO: optimize second call
	for key, _ := range deleteActions {
		if err := mergeDoc.MergeByJSONPath("", key, DeleteMarked); err != nil {
			continue
		}
	}

	return nil
}

func (mergeDoc * MergeDocuments) mergeSequentions(objectKeyName string, seqDiff map[string]string) error {
	//Perform sorting
	for key, item := range seqDiff {
		if err := mergeDoc.MergeSequenceByJSONPath(objectKeyName, key, item); err != nil {
			return err
		}
	}

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
	deleteMerges := make(map[string]interface{})
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
			deleteMerges[fileName] = deleteActions
			seqMerges1[fileName] = seqDiff
			return _err
		} ); err != nil {
			return err
		}

	}

	for i := range mergeActionsRemote {

		if err := mergeActionsRemote[i].PerformMergeChanges(workingDirV0, workingDirV2, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			deleteActions :=  deleteMerges[fileName]
			if deleteActions == nil {
				deleteActions = make(map[string]string)
			}

			seqDiff := make(map[string]string)

			_err := mergeDoc.mergeChanges(srcFilePath, dstFilePath,
				fileName,
				fm.FileDiff.Doc1Diffs, deleteActions.(map[string]string), seqDiff)
			deleteMerges[fileName] = deleteActions
			seqMerges2[fileName] = seqDiff
			return _err
		}); err != nil {
			return err
		}

	}

	//Perform delete actions
	for i := range mergeActionsLocal {

		if err := mergeActionsLocal[i].PerformMergeChanges(workingDirV0, workingDirV1, func(srcFilePath, dstFilePath, fileName string, fm * FileMerge, mergeDoc * MergeDocuments) error {
			deleteActions :=  deleteMerges[fileName]
			if deleteActions == nil {
				return nil
			}
			return mergeDoc.mergeDeletions(deleteActions.(map[string]string))
		} ); err != nil {
			return err
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
			return err
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
			return err
		}

	}

	return nil
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

func Process3WayFileMerge(mergeFileName1, mergeFileName2 string, sketchFileV0, sketchFileV1, sketchFileV2 string, outputDir string) error {


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



