// Copyright 2017 Sergey Fedoseev. All rights reserved.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"github.com/stowage/sketchmerge"
	"encoding/json"
	"path/filepath"
	"bytes"
	"bufio"
)

const (
	MergeOpType = iota
	DiffOpType
)


func main() {

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("    sketchmerge diff [optional] <dst_file_or_dir> <src_file_or_dir> ...\n")
		fmt.Printf("    sketchmerge merge -o <output dir> <merge_file> [<merge_file2>] <dst_file_or_dir> <src_file_or_dir> ...\n")
		fmt.Printf("\n")
		fmt.Printf("	Operations:\n")
		fmt.Printf("	  diff - show difference of src and dst file\n")
		fmt.Printf("	  merge - merge items using merge_file from src and dst file\n")
		fmt.Printf("\n")
		fmt.Printf("	Optional parameters for 'diff' operation:\n")
		fmt.Printf("	  --file-output=<path to file> (-f <path to file>) - output difference to file or directory (-b)\n")
		fmt.Printf("	  --baseline=<path to file or dir> (-b <path to file or dir>) - baseline file for 3-way merge case in nice mode (-n)\n")
		fmt.Printf("	  		<src_file_or_dir> <dst_file_or_dir> are compared against baseline file")
		fmt.Printf("	  --nice-description (-n) - analyze difference and provide natural language description\n")
		fmt.Printf("	  --info (-i) - show differences details\n")
		fmt.Printf("	  --dump-file=<damp file path 1> --dump-file=<damp file path 2> (-df <damp file path 1> <damp file path 2>) - use dump files for differences details\n")
		fmt.Printf("	  --sketchtool=<path to sketch tool> (-sk <path to sketch tool>) sketchtool path (if dumpfiles will be regenerate using specified sketchtool)\n")
		fmt.Printf("	  --export=<path to dir> (-e <path to dir>) export artboards and paths to dir\n")
		fmt.Printf("	  (NOT IMPLEMENTED)--dependencies (-d) ARE ENABLED\n")
		fmt.Printf("\n")
		fmt.Printf("	Optional parameters for 'merge' operation:\n")
		fmt.Printf("	  --baseline=<path to file or dir> (-b <path to file or dir>) - baseline file for 3-way merge\n")
		fmt.Printf("	  		<src_file_or_dir> <dst_file_or_dir> are merged into baseline file\n")
		fmt.Printf("	  		[<merge_file2>] should be also specified\n")
		fmt.Printf("\n")
		fmt.Printf("	Optional merge restrictions for whole page adding operations (uses OR conjuction for class and artboardID)\n")
		fmt.Printf("	  --filter-page=<page id> (-fp) export primarily only page with given page id\n")
		fmt.Printf("	  --filter-artboard=<artboard id> (-fa) allows only given artboard id\n")
		fmt.Printf("	  --filter-class=<className> (-fc) allows only objects of given class with sublayers\n")
		fmt.Printf("\n")
		fmt.Printf("	Required parameters for 'merge' operations:\n")
		fmt.Printf("	  --output=<path to dir> (-o <path to dir>) - output resulting sketch file to dir\n")
		fmt.Printf("\n")
		fmt.Printf("	Merge file format <merge_file>:\n")
		fmt.Printf(`		{
			  "merge_actions": [
			    {
			      "file_key": "pages` + string(os.PathSeparator) + `892342FF-2A18-4BFC-9124-28AF6F0D3CEE",
			      "file_ext": ".json",
			      "file_copy_action": 1,
			      "file_diff": {
				"src_to_dst_diff": {
				  "$[\"layers\"][0][\"layers\"][0][\"frame\"][\"x\"]": "$[\"layers\"][0][\"layers\"][0][\"frame\"][\"x\"]",
				  "$[\"layers\"][0][\"layers\"][0][\"frame\"][\"y\"]": "$[\"layers\"][0][\"layers\"][0][\"frame\"][\"y\"]",
				  "+$[\"layers\"][0][\"layers\"][1]": "$[\"layers\"][0][\"layers\"]",
				  "-$[\"layers\"][0][\"layers\"][1]": "",
				  "-$[\"layers\"][1][\"layers\"][2]": ""
				},
				"src_to_dst_seq_diff": {
				  "$[\"layers\"][0][\"layers\"]": "$[\"layers\"][0][\"layers\"]"
				},
				"seq_key": "do_objectID"
			      }
			    }
			  ]
		}
		`)
		fmt.Printf("\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 3 {
		flag.Usage()
		os.Exit(1)
	}

	opType := DiffOpType


	switch flag.Arg(0) {

	case "diff":
		opType = DiffOpType
	case "merge":
		opType = MergeOpType

		if flag.NArg() < 4 {
			flag.Usage()
			os.Exit(1)
		}
		break
	default:
		flag.Usage()
		os.Exit(1)
	}

	if opType == DiffOpType {
		files := make([]string,0)
		outputToFile := ""
		baselineVersion := ""
		isNice := false
		showInfo := false
		var sketchPath * string
		var exportPath * string
		var dumpFile1 * string
		var dumpFile2 * string
		is3WayMerge := false
		for argc := 1; argc < flag.NArg(); argc++ {
			switch flag.Arg(argc) {
			case "-n", "--nice-description":
				isNice = true
			case "-i", "--info":
				showInfo = true
			case "-d", "--dependencies":
				break
			case "-b", "--baseline":
				argc++
				baselineVersion = flag.Arg(argc)
				is3WayMerge = true
				isNice = true
				break
			case "-f", "--file-output":
				argc++
				outputToFile = flag.Arg(argc)
			case "-sk", "--sketchtool":
				argc++
				_sketchPath := flag.Arg(argc)
				sketchPath = &_sketchPath
			case "-e", "--export":
				argc++
				_exportPath := flag.Arg(argc)
				exportPath = &_exportPath
			case "-df", "--dump-file":
				argc++
				file1 := flag.Arg(argc)
				dumpFile1 = &file1
				argc++
				file2 := flag.Arg(argc)
				dumpFile2 = &file2
			default:
				if strings.HasPrefix(flag.Arg(argc), "--baseline=") {
					baselineVersion = strings.TrimPrefix(flag.Arg(argc), "--baseline=")
					is3WayMerge = true
					isNice = true
				} else if strings.HasPrefix(flag.Arg(argc), "--file-output=") {
					outputToFile = strings.TrimPrefix(flag.Arg(argc), "--file-output=")
				} else if strings.HasPrefix(flag.Arg(argc), "--sketchtool=") {
					_sketchPath := strings.TrimPrefix(flag.Arg(argc), "--sketchtool=")
					sketchPath = &_sketchPath
				} else if strings.HasPrefix(flag.Arg(argc), "--export=") {
					_exportPath := strings.TrimPrefix(flag.Arg(argc), "--export=")
					exportPath = &_exportPath
				} else if strings.HasPrefix(flag.Arg(argc), "--dump-file=") {
					file := strings.TrimPrefix(flag.Arg(argc), "--dump-file=")
					if dumpFile1 != nil {
						dumpFile2 = &file
					} else {
						dumpFile1 = &file
					}
				} else {
					files = append(files, flag.Arg(argc))
				}
			}

		}
		if len(files) != 2 {
			flag.Usage()
			os.Exit(1)
		}

		if !is3WayMerge {
			var fsMerge * sketchmerge.FileStructureMerge
			var err error
			if !isNice {
				fsMerge, err = sketchmerge.ProcessFileDiff(files[0], files[1])
			} else {
				fsMerge, err = sketchmerge.ProcessNiceFileDiff(files[0], files[1], showInfo, dumpFile1, dumpFile2, sketchPath, exportPath)
			}
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			var mergeInfo bytes.Buffer
			writer := bufio.NewWriter(&mergeInfo)
			encoder1 := json.NewEncoder(writer)
			encoder1.SetEscapeHTML(false)
			encoder1.SetIndent("", "  ")
			//mergeInfo, err := json.MarshalIndent(fsMerge, "", "  ")
			err = encoder1.Encode(fsMerge)
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			if outputToFile == "" && exportPath != nil{
				outputToFile = *exportPath + string(os.PathSeparator) + "diff.json"
			}

			if outputToFile != "" {
				if err := sketchmerge.WriteToFile(outputToFile, mergeInfo.Bytes()); err!=nil {
					fmt.Printf("Error occured: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Println(string(mergeInfo.Bytes()))
			}
		} else {
			fsMergeBaseToSrc, err := sketchmerge.ProcessNiceFileDiff3Way(baselineVersion, files[0], files[1])
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			mergeInfoBaseToSrc, err := json.MarshalIndent(fsMergeBaseToSrc, "", "  ")
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			if outputToFile != "" {
				if fileInfo, err := os.Stat(outputToFile); fileInfo != nil && fileInfo.IsDir() {
					if err != nil {
						fmt.Printf("Error occured: %v\n", err)
						os.Exit(1)
					}
					if err := sketchmerge.WriteToFile(outputToFile+string(os.PathSeparator)+strings.TrimSuffix(filepath.Base(files[1]), filepath.Ext(files[0]))+".diff", mergeInfoBaseToSrc); err!=nil {
						fmt.Printf("Error occured: %v\n", err)
						os.Exit(1)
					}
				} else {
					if err := sketchmerge.WriteToFile(outputToFile, mergeInfoBaseToSrc); err!=nil {
						fmt.Printf("Error occured: %v\n", err)
						os.Exit(1)
					}
				}
			} else {
				fmt.Println(string(mergeInfoBaseToSrc))
			}
		}


	}

	if opType == MergeOpType {
		files := make([]string,0)
		outputToDir := ""
		baselineVersion := ""
		is3WayMerge := false
		filterPageId := ""
		filterArtboardId := ""
		filterClassName := ""
		for argc := 1; argc < flag.NArg(); argc++ {
			switch flag.Arg(argc) {

			case "-o", "--output":
				argc++
				outputToDir = flag.Arg(argc)
				break
			case "-b", "--baseline":
				argc++
				baselineVersion = flag.Arg(argc)
				is3WayMerge = true
				break
			case "-fp", "--filter-page":
				argc++
				filterPageId = flag.Arg(argc)
				break
			case "-fa", "--filter-artboard":
				argc++
				filterArtboardId = flag.Arg(argc)
				break
			case "-fc", "--filter-class":
				argc++
				filterClassName = flag.Arg(argc)
				break

			default:
				if strings.HasPrefix(flag.Arg(argc), "--baseline=") {
					baselineVersion = strings.TrimPrefix(flag.Arg(argc), "--baseline=")
					is3WayMerge = true
				} else if strings.HasPrefix(flag.Arg(argc), "--filter-page=") {
					filterPageId = strings.TrimPrefix(flag.Arg(argc), "--filter-page=")
				} else if strings.HasPrefix(flag.Arg(argc), "--filter-artboard=") {
					filterArtboardId = strings.TrimPrefix(flag.Arg(argc), "--filter-artboard=")
				} else if strings.HasPrefix(flag.Arg(argc), "--filter-class=") {
					filterClassName = strings.TrimPrefix(flag.Arg(argc), "--filter-class=")
				} else if strings.HasPrefix(flag.Arg(argc), "--output=") {
					outputToDir = strings.TrimPrefix(flag.Arg(argc), "--output=")
				} else {
					files = append(files, flag.Arg(argc))
				}
			}

		}

		requiredFileCount := 3

		if is3WayMerge {
			requiredFileCount = 4
		}

		if len(files) != requiredFileCount {
			flag.Usage()
			os.Exit(1)
		}

		sketchFileV2Info, errv2 := os.Stat(files[2])

		if errv2 != nil {
			fmt.Printf("Error occured: %v\n", errv2)
			os.Exit(1)
		}

		isDstDir := sketchFileV2Info.IsDir()

		if !isDstDir && outputToDir == "" {
			flag.Usage()
			os.Exit(1)
		}



		if !is3WayMerge {

			var filter * sketchmerge.PageFilter
			if filterPageId != "" || filterArtboardId != "" || filterClassName != "" {
				filter = &sketchmerge.PageFilter{filterPageId, filterArtboardId, filterClassName}
			}
			err := sketchmerge.ProcessFileMerge(files[0], files[1], files[2], outputToDir, filter)

			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}
		} else {
			err := sketchmerge.Process3WayFileMerge(files[0], files[1], baselineVersion, files[2], files[3], outputToDir)
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}
		}

	}
}

