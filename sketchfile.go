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
	SetDiff(src string, dst string, niceDescShort string, niceDesc string)
}

type MainDiff struct {
	Description map[string]string `json:"description,omitempty"`
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


func (sd* MainDiff) SetDiff(src string, dst string, niceDescShort string, niceDesc string) {
	sd.Description["nice_description_short"] = niceDescShort
	sd.Description["nice_description"] = niceDesc
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

func getNiceTextForUnknown(srcact ApplyAction, key string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "Added property %v"
		niceDesc = "Added property %v"
	case ValueDelete:
		niceDescShort = "Property %v removed"
		niceDesc = "Property %v removed"
	case ValueChange, SequenceChange:
		niceDescShort = "Property %v has changed"
		niceDesc = "Property %v has changed"
	}

	return fmt.Sprintf(niceDescShort, key), fmt.Sprintf(niceDesc, key )
}


func getNiceTextForPage(srcact ApplyAction, pageName string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "Page %v was added"
		niceDesc = "Page %v was added"
	case ValueDelete:
		niceDescShort = "Page %v is deleted"
		niceDesc = "Page %v is deleted"
	case ValueChange:
		niceDescShort = "Page %v has changed"
		niceDesc = "Page %v has changed"
	case SequenceChange:
		niceDescShort = "Sequence inside page %v has changed"
		niceDesc = "Sequence inside page %v has changed"
	}

	return fmt.Sprintf(niceDescShort, pageName), fmt.Sprintf(niceDesc, pageName )
}

func getNiceTextForArtboard(srcact ApplyAction, artboardName string, pageName string) (string, string) {

	var niceDesc = ""
	var niceDescShort = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "Artboard %v was added"
		niceDesc = "Artboard %v was added to page %v"
	case ValueDelete:
		niceDescShort = "Artboard %v is deleted"
		niceDesc = "Artboard %v is deleted from page %v"
	case ValueChange:
		niceDescShort = "Artboard %v has changed"
		niceDesc = "Artboard %v has changed on page %v"
	case SequenceChange:
		niceDescShort = "Sequence of items inside %v has changed"
		niceDesc = "Sequence of items inside %v has changed on page %v"
	}

	return fmt.Sprintf(niceDescShort, artboardName), fmt.Sprintf(niceDesc, artboardName, pageName )
}

func getNiceTextForUnknownLayer(srcact ApplyAction, layerName string, layerPath string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "New layer %v"
		niceDesc = "New layer %v was added at location %v"
	case ValueDelete:
		niceDescShort = "Delete %v layer "
		niceDesc = "Deleted %v layer at location %v"
	case ValueChange:
		niceDescShort = "Layer %v has changed"
		niceDesc = "Layer %v has changed at location %v"
	case SequenceChange:
		niceDescShort = "Layers sequence inside %v has changed"
		niceDesc = "Layers sequence inside %v has changed at location %v"
	}

	return fmt.Sprintf(niceDescShort, layerName), fmt.Sprintf(niceDesc, layerName, layerPath )
}

func getNiceTextForLayer(srcact ApplyAction, layerName string, pageName string, artboardName string, layerPath string) (string, string) {

	var niceDesc string = ""
	var niceDescShort string = ""

	switch srcact {
	case ValueAdd:
		niceDescShort = "New layer %v"
		niceDesc = "New layer %v was added to page %v in %v artboard (%v)"
	case ValueDelete:
		niceDescShort = "Delete %v layer "
		niceDesc = "Deleted %v layer from page %v in %v artboard (%v)"
	case ValueChange:
		niceDescShort = "Layer %v has changed"
		niceDesc = "Layer %v has changed on page %v in %v artboard (%v)"
	case SequenceChange:
		niceDescShort = "Layers sequence inside %v has changed"
		niceDesc = "Layers sequence inside %v has changed on page %v in %v artboard (%v)"
	}

	return fmt.Sprintf(niceDescShort, layerName), fmt.Sprintf(niceDesc, layerName, pageName, artboardName, layerPath )
}


func (li * SketchLayerInfo) SetDifference(diff SketchDiff, diffSrc string, diffDst string) {

	var page interface{}
	var artboard interface{}
	var layer interface{}

	var actual Difference = &diff

	if li.PageID != "" {
		page = diff.PageDiff[li.PageID]
		if page == nil {
			page = SketchPageDiff{Name: li.PageName, ArtboardDiff: make(map[string]interface{}), MainDiff: MainDiff{ Diff: make(map[string]interface{}), Description: make(map[string]string)}}
			diff.PageDiff[li.PageID] = page
		}
		_page := page.(SketchPageDiff)
		actual = &_page
	}

	if page != nil {
		artboard = page.(SketchPageDiff).ArtboardDiff[li.ArtboardID]

		if artboard == nil {
			artboard = SketchArtboardDiff{Name: li.ArtboardName,  LayerDiff: make(map[string]interface{}), MainDiff:MainDiff{ Diff: make(map[string]interface{}), Description: make(map[string]string)} }
			page.(SketchPageDiff).ArtboardDiff[li.ArtboardID] = artboard
		}
		_artboard := artboard.(SketchArtboardDiff)
		actual = &_artboard
	}

	if artboard != nil {
		layer = artboard.(SketchArtboardDiff).LayerDiff[li.LayerID]

		if layer == nil {
			layer = SketchLayerDiff{Name: li.LayerName, MainDiff:MainDiff{ Diff: make(map[string]interface{}), Description: make(map[string]string)}}
			artboard.(SketchArtboardDiff).LayerDiff[li.LayerID] = layer
		}
		_layer := layer.(SketchLayerDiff)
		actual = &_layer
	}


	actual.SetDiff(diffSrc, diffDst, li.NiceDescriptionShort, li.NiceDescription)

}


func ProduceNiceDiff(doc1 map[string]interface{}, doc2 map[string]interface{}, diff map[string]interface{}, isSeqChange bool) map[string]interface{}  {

	if diff==nil {
		return nil
	}

	niceDiff := make(map[string]interface{})
	skDiff := SketchDiff{PageDiff: make(map[string]interface{}), MainDiff: MainDiff{Diff:make(map[string]interface{}), Description: make(map[string]string)}}

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

		srcSel, srcact, _ := Parse(key)
		doc := doc1

		if item == "" && srcact == ValueDelete {
			doc = doc2
		}

		_, lastNode, err := srcSel.ApplyWithEvent(doc, func(v interface{}, prevNode Node, node Node) bool {
			if prevNode == nil {
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
				layer := v.(map[string]interface{})
				if layer != nil {
					lname := layer["name"]
					lid := layer["do_objectID"]
					if lname == nil || lid == nil {
						return true
					}


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

		if isSeqChange {
			srcact = SequenceChange
		}
		if pageID != "" && artboardID != "" && layerID != "" {
			niceDescShort, niceDesc = getNiceTextForLayer(srcact, layerName, pageName, artboardName, layerPath)
		} else if layerID != "" {
			niceDescShort, niceDesc = getNiceTextForUnknownLayer(srcact, layerName, layerPath)
		} else if artboardID != "" {
			niceDescShort, niceDesc = getNiceTextForArtboard(srcact, artboardName, pageName)
		} else if pageID != "" {
			niceDescShort, niceDesc = getNiceTextForPage(srcact, pageName)
		} else {
			niceDescShort, niceDesc = getNiceTextForUnknown(srcact, fmt.Sprintf("%v", lastNode.GetKey()))
		}


		diff := SketchLayerInfo{layerName, layerID,
			artboardName, artboardID,
			pageName, pageID,
			niceDescShort, niceDesc}

		diff.SetDifference(skDiff, key, item.(string))

	}

	if len(diff) > 0 {
		niceDiff["nice_diff"] = skDiff
	}

	return niceDiff

}

func CompareJSONNice(doc1File string, doc2File string) (*JsonStructureCompare, error) {
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

	jsCompare.Doc1Diffs = ProduceNiceDiff(result1, result2, jsCompare.Doc1Diffs, false)
	jsCompare.Doc2Diffs = ProduceNiceDiff(result2, result1, jsCompare.Doc2Diffs, false)


	return jsCompare, nil
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
	fsMerge.FileSetChange(baseFileStruct, newFileStruct)

	if !isNice {
		for i := range fsMerge.MergeActions {
			//fmt.Printf("ext: %v", filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey)))
			if filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)) == ".json" {
				result, err := CompareJSON(workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt,  workingDirV2 + "/" + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)
				if err != nil {
					return nil, err
				}
				fsMerge.MergeActions[i].FileDiff = *result
			}
		}


		mergeInfo, _ := json.MarshalIndent(fsMerge, "", "  ")

		return mergeInfo, nil
	} else {
		for i := range fsMerge.MergeActions {
			//fmt.Printf("ext: %v", filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey)))
			if filepath.Ext(strings.ToLower(fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)) == ".json" {
				result, err := CompareJSONNice(workingDirV1 + string(os.PathSeparator) + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt,  workingDirV2 + "/" + fsMerge.MergeActions[i].FileKey + fsMerge.MergeActions[i].FileExt)
				if err != nil {
					return nil, err
				}
				fsMerge.MergeActions[i].FileDiff = *result
			}
		}


		mergeInfo, _ := json.MarshalIndent(fsMerge, "", "  ")

		return mergeInfo, nil
	}
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
	for key, item := range docDiffs {
		if item == "" {
			deleteActions[key] = ""
		} else if !strings.HasPrefix(key, "^") {
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

func mergeActions(workingDirV1 string, workingDirV2 string, mergeJSON FileStructureMerge) error {

	for i := range mergeJSON.MergeActions {

		if mergeJSON.MergeActions[i].FileDiff.Doc1Diffs == nil {
			continue
		}

		if err := merge(workingDirV1, workingDirV2,
			mergeJSON.MergeActions[i].FileKey + mergeJSON.MergeActions[i].FileExt,
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

