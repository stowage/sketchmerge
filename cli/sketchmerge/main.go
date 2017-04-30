package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"github.com/stowage/sketchmerge"
	"encoding/json"
	"path/filepath"
)

const (
	MergeOpType = iota
	DiffOpType
)


func main() {

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("    sketchmerge diff [optional] <merge_file> <dst_file_or_dir> <src_file_or_dir> ...\n")
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
		fmt.Printf("	  (NOT IMPLEMENTED)--dependencies (-d) ARE ENABLED\n")
		fmt.Printf("\n")
		fmt.Printf("	Optional parameters for 'merge' operation:\n")
		fmt.Printf("	  --baseline=<path to file or dir> (-b <path to file or dir>) - baseline file for 3-way merge\n")
		fmt.Printf("	  		<src_file_or_dir> <dst_file_or_dir> are merged into baseline file")
		fmt.Printf("	  		[<merge_file2>] should be also specified")
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
		is3WayMerge := false
		for argc := 1; argc < flag.NArg(); argc++ {
			switch flag.Arg(argc) {
			case "-n", "--nice-description":
				isNice = true
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
			default:
				if strings.HasPrefix(flag.Arg(argc), "--baseline=") {
					baselineVersion = strings.TrimPrefix(flag.Arg(argc), "--baseline=")
					is3WayMerge = true
					isNice = true
				} else if strings.HasPrefix(flag.Arg(argc), "--file-output=") {
					outputToFile = strings.TrimPrefix(flag.Arg(argc), "--file-output=")
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
				fsMerge, err = sketchmerge.ProcessNiceFileDiff(files[0], files[1])
			}
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			mergeInfo, err := json.MarshalIndent(fsMerge, "", "  ")
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			if outputToFile != "" {
				if err := sketchmerge.WriteToFile(outputToFile, mergeInfo); err!=nil {
					fmt.Printf("Error occured: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Println(string(mergeInfo))
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
			default:
				if strings.HasPrefix(flag.Arg(argc), "--baseline=") {
					baselineVersion = strings.TrimPrefix(flag.Arg(argc), "--baseline=")
					is3WayMerge = true
				} else if strings.HasPrefix(flag.Arg(argc), "--output=") {
					outputToDir = strings.TrimPrefix(flag.Arg(argc), "--output=")
				} else {
					files = append(files, flag.Arg(argc))
				}
			}

		}

		requiredFileCount := 3

		if is3WayMerge {
			requiredFileCount := 4
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
			err := sketchmerge.ProcessFileMerge(files[0], files[1], files[2], outputToDir)

			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}
		} else {
			//TODO: Implement 3-way merge
		}

	}
}

