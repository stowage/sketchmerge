package sketchmerge

import (
	"regexp"
	"strings"
	"log"
	"time"
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

	if objKey != nil {
		depKey = objKey.(string)
		depMap = dep.DepObj
	} else {
		depKey = jsonpath
		jsonpath = ""
		depMap = dep.DepPath
	}

	if IsSketchID(key) {
		depItem := depMap[depKey]
		if depItem == nil {
			depItem = make([]interface{}, 0)
		}
		depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath, Ref:key})
	}

	if IsSketchID(value) {
		depItem := depMap[depKey]
		if depItem == nil {
			depItem = make([]interface{}, 0)
		}
		depMap[depKey] = append(depItem.([]interface{}), DependentObj{JsonPath:jsonpath, Ref:value.(string)})
	}
}

/*func (dep* DependentObjects) AddMergeDependentObject(key string, ref string, jsonpathSrc string, jsonpathDst string)  {


}


func (dep* DependentObjects) ResolveDependencies(filepath string, jsonpath string, doc map[string]interface{}) error {
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

}*/