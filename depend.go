package sketchmerge

import (
	"regexp"
	"strings"
	"log"
	"time"
	"path/filepath"
	"os"
	"fmt"
)
const GuidFormat = "^[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$"

const RefImagesFormat = "^images/[a-z0-9]{40}$"

const RefPagesFormat = "^pages/[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$"

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

func (dep* DependentObjects) AddDependentObject(key string, value interface{}, jsonpath string)  {

	var depKey string
	var depMap map[string]interface{}

	if key == "_ref" {
		fmt.Printf("")
	}
	//isKeyObjectID := IsSketchID(key)
	isValueObjectID := IsSketchID(value)

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

	if strings.HasPrefix(depKey, "pages/") {
		depKey = strings.TrimPrefix(depKey, "pages/")
	}

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
	}

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
			if layer["_class"] == "symbolMaster" {
				sid := layer["symbolID"]
				if sid != nil {
					dep.AddDependent(sid.(string), fileJsonPath2, fileJsonPath1, fileKey)
				}

			}
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
				if strings.HasPrefix(mergeActions[i].FileKey, "pages/") {
					objectID := strings.TrimPrefix(mergeActions[i].FileKey, "pages/")
					jsonFilePath := "~" + fullFilePath + "~$"
					dep.AddDependent(objectID, fileActionPrefix+jsonFilePath, fileActionPrefix+jsonFilePath, mergeActions[i].FileKey)

				} else {
					jsonFilePath := fileActionPrefix + "~" + fullFilePath + "~$"
					dep.AddDependent(mergeActions[i].FileKey, jsonFilePath, jsonFilePath, mergeActions[i].FileKey)
				}
			// if we need to delete this file
			} else {

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
			if strings.HasPrefix(mergeActions[i].FileKey, "pages/") {
				objectID := strings.TrimPrefix(mergeActions[i].FileKey, "pages/")
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

