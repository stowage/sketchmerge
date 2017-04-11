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
	JsonPath string
	Location string
}

type DependentObjects struct {
	DepObj map[string][]interface{}
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

func (dep* DependentObjects) AddDependentObject(key interface{}, value interface{}, location string, jsonpath string)  {
	keyString := key.(string)
	valueString := key.(string)
	if IsSketchID(keyString) {
		depItem := dep.DepObj[keyString]
		if depItem != nil {
			depItem = make([]interface{}, 1)
		}
		dep.DepObj[keyString] = append(depItem, DependentObj{jsonpath, location})
	}

	if IsSketchID(valueString) {
		depItem := dep.DepObj[valueString]
		if depItem != nil {
			depItem = make([]interface{}, 1)
		}
		dep.DepObj[valueString] = append(depItem, DependentObj{jsonpath, location})
	}
}

func (dep* DependentObjects) removeOrphanObjects() {
	for key, item := range dep.DepObj {
		if len(item) == 1 {
			delete(dep.DepObj, key)
		}
	}
}
