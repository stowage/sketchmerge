### Main files

compare.go - performs comparison of sketch file directory structure and compares jsons
depend.go - finds dependent objects changes
merge.go - performs merge operations
sketchfile.go - performs diff and merge operations on sketch files

### COMPARE.GO

compare.go can produce two types of differences between sketch files:
nice difference
regular differences

# Nice differences type

Nice difference - returns difference of two json files in structured type page->artboarad->layer->…->layer
To implement "nice difference" following structures are used:
```
type SketchDiff struct <MainDiff>
	→type SketchPageDiff struct <MainDiff>
		→type SketchArtboardDiff struct <MainDiff>
			→type SketchLayerDiff struct <MainDiff>
	→type SketchLayerDiff struct <MainDiff>
```


# regular difference

returns only map of changes 

# compare.go MAIN types

`type FileStructureMerge struct` - contains difference in filesets of two sketch files, each file difference is represented using FileMerge type
`type FileMerge struct` - difference of single file. Each FileMerge element in FileStructureMerge has an action of type FileActionType. Which tells the merger how to deal with given file when applying changes, there are three types of file merge operations MERGE,	ADD, DELETE. 
`type JsonStructureCompare struct` - difference between json files. Has important property Doc1Diffs which contains a structure of changes, each change is described with a pair of jsonpaths, describing a rule to copy a data from-to.

# compare.go MAIN FUNCTIONS

`ExtractSketchDirStruct(baseDir string, newDir string) (SketchFileStruct, SketchFileStruct)` - extracts unzipped sketch file file-structure
 
`FileSetChange(baseSet SketchFileStruct, newSet SketchFileStruct)` - performs comparison of two unzipped sketch directories. Compares files by name. If both directories containing same file specifies MERGE file operation, otherwise ADD or DELETE.

after file changes are completed compare.go provides following function to compare json documents:
`CompareDocuments(workingDirV1, workingDirV2 string) error`

this function is applied to all files in FileStructureMerge

`CompareDocuments` creates `JsonStructureCompare` in each `FileMerge` element. It creates a rule described in jsonpath notations to copy values from source json to destination.

it also prepares hashes for dependent elements.

here is how changes map usually looks like:
```
"$[\"layers\"][0][\"layers\"][0][\"frame\"][\"x\"]": "$[\"layers\"][0][\"layers\"][0][\"frame\"][\"x\"]",
"$[\"layers\"][0][\"layers\"][0][\"frame\"][\"y\"]": "$[\"layers\"][0][\"layers\"][0][\"frame\"][\"y\"]",
"+$[\"layers\"][0][\"layers\"][1]": "$[\"layers\"][0][\"layers\"]",
"-$[\"layers\"][0][\"layers\"][1]": "",
"-$[\"layers\"][1][\"layers\"][2]": ""
```

DEPEND.GO

While compare.go is performing compare operations it calls `AddDependentObjects(objKey string, docTree * interface{}, dep * DependentObjects, jsonpath string) bool` if property of a sketch layer has any element which was recognized as a SketchID.

There are three know types of references in sketch:
${GUID} - usually references to a symbol meta information
images/${id} - references to sketch subdirectory "images" (points to appropriate file in a directory)
pages/${id} - refers to sketch subdirectory pages (points to exact file in a directory)

`AddDependentObjects` accumulates all references in `DependentObjects` of `JsonStructureCompare` property recursively

Function `ProceedDependencies(workingDirV1 string, workingDirV2 string, fileMerge []FileMerge ) (*DependentObjects, *DependentObjects, *FileMerge, error)`
create dependencies map, which can be used by to find a dependencies. This map has a jsonpath as a key, which allows to lookup dependent object by picking up any key in changes after CompareDocument operation has been performed.
So for example if there is a new layer containing some dependent object in "images" subdirectory, dependence map will help to pick up dependent object for extended jsonpath "add layer" operation "+$["layers"][3]". So it will pick up  "A~images/67346872634863847683453.png" operation which can help to extend existing map of changes.

the main function creating this associations is:
`FindMatchingDiffs(docType DocumentType,fileName string, matchingKey string, depPaths1 map[string]interface{}, depPaths2 map[string]interface{}, diffs map[string]interface{})`

it’s important to understand that dependent object may have also a dependent object that is why `FindMatchingDiffs` locates all dependencies with dependencies. Let say you have a layer which requires a dependency of a dependency. In that case you have to add all dependent objects to the changes map.

Here is how changes with dependencies usually looks like:
```
"+$[\"layers\"][@do_objectID='1DB3913B-453B-4792-B192-88E244428A27']": "$[\"layers\"]",
"A~images/1624441dba44708004548bd0e3c782d459c50933.png~$": "A~images/1624441dba44708004548bd0e3c782d459c50933.png~$",
"A~images/5791ea161a345d6c3b966ba67b761b5bb6c52100.png~$": "A~images/5791ea161a345d6c3b966ba67b761b5bb6c52100.png~$",
"~meta.json~+$[\"pagesAndArtboards\"][\"D38356F1-8FD4-4EB3-BBAC-EEF6EB45F6B8\"][\"artboards\"][\"1DB3913B-453B-4792-B192-88E244428A27\"]": "~meta.json~$[\"pagesAndArtboards\"][\"D38356F1-8FD4-4EB3-BBAC-EEF6EB45F6B8\"][\"artboards\"]"
```

so dependencies map allows to pick up dependent changes by looking up dependent jsonpaths using key "+$[\"layers\"][@do_objectID='1DB3913B-453B-4792-B192-88E244428A27']". That means that jsonpaths bellow are dependent changes required by given object above:
"A~images/1624441dba44708004548bd0e3c782d459c50933.png~$": "A~images/1624441dba44708004548bd0e3c782d459c50933.png~$",
"A~images/5791ea161a345d6c3b966ba67b761b5bb6c52100.png~$": "A~images/5791ea161a345d6c3b966ba67b761b5bb6c52100.png~$",
"~meta.json~+$[\"pagesAndArtboards\"][\"D38356F1-8FD4-4EB3-BBAC-EEF6EB45F6B8\"][\"artboards\"][\"1DB3913B-453B-4792-B192-88E244428A27\"]": "~meta.json~$[\"pagesAndArtboards\"][\"D38356F1-8FD4-4EB3-BBAC-EEF6EB45F6B8\"][\"artboards\"]"


### MERGE.GO

contains jsonpath parser and merge functions.

# parser

parses JsonPath like "$[\"layers\"][0][\"layers\"][0][\"frame\"][\"x\"]" into bidirectional linked list `type RootNode struct` implementing interfaces `type Selection interface` and `type Node interface`

Core functions are:
```
Parse(s string) (Node, ApplyAction, error)
translates jsonpath  to linked list Node’s

Apply(v interface{}) (interface{}, Node, error)
ApplyWithEvent(v interface{}, e NodeEvent) (interface{}, Node, error)
both methods return position in a tree for a parsed jsonpath
```

# MERGER

Merging is performed on structure `type MergeDocuments struct` containing both json documents.
following functions doing different kind of merges:
`MergeByJSONPath(srcPath string, dstPath string, mode DeleteMode) error` - merges properties
`MergeSequenceByJSONPath(objectKeyName string, srcPath string, dstPath string) error` - merges sequence of arrays

sketchfile.go contains function which performs merge operations:
`merge(workingDirV1 string, workingDirV2 string, fileName string, objectKeyName string, docDiffs map[string]interface{} ) error`

function `merge` works as follows: 
```
on the first step it applies only changes in properties
on second: marks element need to be removed (we can’t remove elements here because sequence change algorithm contains indeces of elements in jsonpath)
changes sequence of array properties (we apply longest jsonpaths first, because there could be changes of sequences in subsequences)
removes elements marked to delete
```


### SKETCHFILE.GO

Manages differences and merge operations for a given sketch file.

Following functions are important:

Merge functions:
```
- func merge(workingDirV1 string, workingDirV2 string, fileName string, objectKeyName string, docDiffs map[string]interface{} ) error 
- func ProcessFileMerge(mergeFileName string, sketchFileV1 string, sketchFileV2 string, outputDir string, filter * PageFilter) error 
- func (mergeDoc * MergeDocuments) mergeChanges(srcFilePath string, dstFilePath string, fileName string, docDiffs map[string]interface{} , deleteActions, seqDiff map[string]string) error 
func Process3WayFileMerge(mergeFileName1, mergeFileName2 string, sketchFileV0, sketchFileV1, sketchFileV2 string, outputDir string) error 
```

Diff functions:
```
- func ProcessNiceFileDiff3Way(sketchFileV0, sketchFileV1, sketchFileV2 string) (*FileStructureMerge, error) 
- func ProcessNiceFileDiff(sketchFileV1 string, sketchFileV2 string, hasInfo bool, dumpFile1 * string, dumpFile2 * string, sketchPath * string, exportPath * string) (*FileStructureMerge, error) 
func ProcessFileDiff(sketchFileV1 string, sketchFileV2 string) (*FileStructureMerge, error) 
```

there are two different types of nice diff function, 2-way and 3-way. The difference is they return two different sets of changes. 2-way functions return "original" and "local" changes. 3-way - "original", "local" and "remote". "original" - corresponds to original version of sketch document without user changes. "local" - corresponds to version changed by user. "remote" - version changed by someone against original version, so the "local" and "remote" changes may fall into conflict.