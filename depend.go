package sketchmerge

import (
	"regexp"
	"strings"
	"log"
	"time"
	"path/filepath"
	"os"
	_"fmt"
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

func (dep* DependentObjects) AddDependentObject(objKey interface{}, key string, value interface{}, jsonpath string)  {

	var depKey string
	var depMap map[string]interface{}

	isKeyObjectID := IsSketchID(key)
	isValueObjectID := IsSketchID(value)

	isObjID := false
	if objKey != nil {
		depKey = objKey.(string)
		depMap = dep.DepObj
		isObjID = true
	} else if isKeyObjectID {
		depKey = key
		depMap = dep.DepObj

	} else if isValueObjectID {
		depKey = value.(string)
		depMap = dep.DepObj

	} else {
		depKey = jsonpath
		jsonpath = ""
		depMap = dep.DepPath

	}

	if isObjID {
		depItem := depMap[depKey]
		if depItem == nil {
			depItem = make([]interface{}, 0)
		}
		depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath})
	}

	if isKeyObjectID {
		depItem := depMap[depKey]
		if depItem == nil {
			depItem = make([]interface{}, 0)
		}
		depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath})
	}

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

func (dep* DependentObjects) ResolveDependencies(fileKey string, filepath string, jsonpath1 string, jsonpath2 string, doc map[string]interface{}) error {
	srcSel, _, err1 := Parse(jsonpath1)

	if err1 != nil {
		return err1

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


func (dep * DependentObjects) buildDependencePaths(workingPathV1 string, workingPathV2 string, mergeActions []FileMerge) (map[string]interface{},error) {

	fileMap1 := make(map[string]interface{})

	for i := range mergeActions {
		fileMap1[mergeActions[i].FileKey + mergeActions[i].FileExt] = mergeActions[i]
		if filepath.Ext(strings.ToLower(mergeActions[i].FileKey + mergeActions[i].FileExt)) == ".json" {
			fullFilePath :=  mergeActions[i].FileKey + mergeActions[i].FileExt
			if mergeActions[i].Action == ADD {
				if strings.HasPrefix(mergeActions[i].FileKey, "pages/") {
					objectID := strings.TrimPrefix(mergeActions[i].FileKey, "pages/")
					jsonFilePath := "~" + fullFilePath + "~$"
					dep.AddDependent(objectID, "A" + jsonFilePath, "A" + jsonFilePath, mergeActions[i].FileKey)
				} else {
					jsonFilePath := "A~" + fullFilePath + "~$"
					dep.AddDependent( mergeActions[i].FileKey, jsonFilePath, jsonFilePath, mergeActions[i].FileKey)
				}
			} else if mergeActions[i].Action == DELETE {
				if strings.HasPrefix(mergeActions[i].FileKey, "pages/") {
					objectID := strings.TrimPrefix(mergeActions[i].FileKey, "pages/")
					jsonFilePath := "D~" + fullFilePath + "~$"
					dep.AddDependent(objectID, jsonFilePath, jsonFilePath,mergeActions[i].FileKey)
				} else {
					jsonFilePath := "D~" + fullFilePath + "~$"
					dep.AddDependent(mergeActions[i].FileKey, jsonFilePath, jsonFilePath, mergeActions[i].FileKey)
				}
			} else {

				if mergeActions[i].FileDiff.Doc1Diffs == nil {
					continue
				}

				if _, err := os.Stat(workingPathV1 + string(os.PathSeparator) + fullFilePath); os.IsNotExist(err) {
					continue
				}

				result, err := readJSON(workingPathV1 + string(os.PathSeparator) + fullFilePath)

				if err != nil {
					return nil, err
				}

				for key, item := range mergeActions[i].FileDiff.Doc1Diffs {
					if err := dep.ResolveDependencies(mergeActions[i].FileKey, fullFilePath, key, item.(string), result); err!=nil {
						return nil, err
					}
				}
			}
		}
	}

	return fileMap1, nil
}

