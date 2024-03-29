package sketchmerge

import (
	"fmt"
	"encoding/json"
	"testing"
)

func TestMergeDocuments_MergeByJSONPath1(t *testing.T) {
	var jsonDoc1 = make(map[string]interface{})
	err1 := json.Unmarshal([]byte(`{
		"layers":[
			{"do_objectID": "BE4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
			"name": "test1",
			"exportOptions": {
    				"_class": "exportOptions",
    				"exportFormats": [],
    				"includedLayerIds": [],
    				"layerOptions": 0,
    				"shouldTrim": false
  			},
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 300,
    				"x": 0,
    				"y": 0
  				}
  			},
  			{"do_objectID": "FE4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
  			"name": "test2",
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 300,
    				"x": 0,
    				"y": 0
  				}
  			},
  			{"do_objectID": "1E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
  			"name": "test3",
  			"overrides": {
				"0": {
				  "900F2395-B6A0-4A36-AFBF-AE318EB53D6D": "over 1",
				  "28798A7A-49DC-4140-81DC-0C15987AC10A": "over2",
				  "AE36349A-9F42-49CE-B7BC-BB47A37560AC": "30:00 over3",
				  "6D5897A5-FBCF-4E92-9971-F163C3121DC3": {
				    "_class": "MSJSONFileReference",
				    "_ref_class": "MSImageData",
				    "_ref": "images/2e7c958c5f76184aa7eea2ffb80ab76d1ff7a115"
				  },
				  "441BA102-5E44-429E-BF1C-0C1B95CBA48C": "over5",
				  "B2614EAC-0547-4469-B3D2-72997954030D": "over6:"
				}
			},
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 305,
    				"x": 0,
    				"y": 0
  				}
  			},
  			{"do_objectID": "2E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
  			"name": "test4",
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 300,
    				"x": 0,
    				"y": 0
  				}
  			},
  			{"do_objectID": "8E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
  			"name": "test5",
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 300,
    				"x": 0,
    				"y": 0
  				}
  			}

		],
		"fonts": [
			    "HelveticaNeue-Bold",
			    "HadassaLineProV2-Semibold",
			    "SimplerPro_V2-Bold",
			    "OpenSansHebrew-Regular",
			    "SimplerPro_V2-Regular",
			    "OpenSansHebrew-Bold",
			    "HadassaLineProV2-bold",
			    ".ArialHebrewDeskInterface",
			    "SimplerPro_V2-Black",
			    "FontAwesome",
			    "HadassaLineProV2-Regular",
			    "HelveticaNeue",
			    "LucidaGrande"
			  ]
		}`),
		&jsonDoc1)

	var jsonDoc2 = make(map[string]interface{})
	err2 := json.Unmarshal(
		[]byte(`{
		"layers":[
			{"do_objectID": "2E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
			"name": "test1",
			"exportOptions": {
    				"_class": "exportOptions",
    				"exportFormats": [],
    				"includedLayerIds": [],
    				"layerOptions": 0,
    				"shouldTrim": false
  			},
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 300,
    				"x": 0,
    				"y": 0
  				}
  			},
			{"do_objectID": "BE4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
			"name": "test2",
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 305,
    				"width": 300,
    				"x": 0,
    				"y": 5
  				}
  			},
  			{"do_objectID": "FE4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
  			"name": "test2",
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 300,
    				"x": 0,
    				"y": 0
  				}
  			},
  			{"do_objectID": "1E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
  			"name": "test3",
  			"overrides": {
				"0": {
				  "900F2395-B6A0-4A36-AFBF-AE318EB53D6D": "over 1",
				  "78798A7A-49DC-4140-81DC-0C15987AC10A": "over2",
				  "AE36349A-9F42-49CE-B7BC-BB47A37560AC": "30:00 over3" ,
				  "6D5897A5-FBCF-4E92-9971-F163C3121DC3": {
				    "_class": "MSJSONFileReference",
				    "_ref_class": "MSImageData",
				    "_ref": "images/2e7c958c5f76184aa7eea2ffb80ab76d1ff7a115"
				  },
				  "441BA102-5E44-429E-BF1C-0C1B95CBA48C": "over5",
				  "B2614EAC-0547-4469-B3D2-72997954030D": "over6:"
				}
			},
			"frame": {
    				"_class": "rect",
    				"constrainProportions": true,
    				"height": 300,
    				"width": 301,
    				"x": 1,
    				"y": 3
  				}
  			},
  			{"do_objectID": "9E4C0CBB-05E4-4D6D-9B75-A8A3ACB36CBA",
  			"name": "test1",
			"frame": {
    				"_class": "rect",
    				"constrainProportions": false,
    				"height": 300,
    				"width": 300,
    				"x": 0,
    				"y": 0
  				}
  			}


		],
		"fonts": [
			    "HelveticaNeue-Bold",
			    "HadassaLineProV2-Semibold",
			    "SimplerPro_V2-Bold",
			    "OpenSansHebrew-Regular",
			    "SimplerPro_V2-Regular",
			    "OpenSansHebrew-Regular",
			    "HadassaLineProV2-bold",
			    ".ArialHebrewDeskInterface",
			    "FontAwesome",
			    "HadassaLineProV2-Regular",
			    "HelveticaNeue",
			    "LucidaGrande"
			  ]
		}`),
		&jsonDoc2)

	if err1 != nil || err2 !=nil {
		fmt.Printf("Error occured %v %v", err1, err2)
	}

	mergeDoc := MergeDocuments{jsonDoc1, jsonDoc2}

	mergeDoc.MergeByJSONPath(`$["layers"][2]["frame"]["constrainProportions"]`, `$["layers"][3]["frame"]["constrainProportions"]`)

	mergeDoc.MergeByJSONPath(`+$["layers"][4]`, `$["layers"]`)
	mergeDoc.MergeByJSONPath(`+$["layers"][0]["exportOptions"]`, `$["layers"][1]`)
	mergeDoc.MergeByJSONPath("", `-$["layers"][0]["exportOptions"]`)
	mergeDoc.MergeByJSONPath(`$["layers"][2]`, `$["layers"][3]`)

	mergeDoc.MergeByJSONPath(`$["fonts"][5]`, `$["fonts"][5]`)
	mergeDoc.MergeByJSONPath(`$["fonts"][8]`, `$["fonts"][8]`)
	mergeDoc.MergeByJSONPath(`+$["fonts"][12]`, `$["fonts"]`)
	mergeDoc.MergeByJSONPath("", `-$["layers"][4]`)
	mergeDoc.MergeSequenceByJSONPath("do_objectID",`$["layers"]`, `$["layers"]`)

	mergeInfo2, _ := json.MarshalIndent(mergeDoc.DstDocument, "", "  ")

	fmt.Println(string(mergeInfo2))

}
