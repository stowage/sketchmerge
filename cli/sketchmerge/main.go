package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"github.com/stowage/sketchmerge"
	_"path/filepath"
	_"encoding/json"
)

const (
	MergeOpType = iota
	DiffOpType
)


func main() {

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("    sketchmerge diff [optional] <merge_file> <dst_file_or_dir> <src_file_or_dir> ...\n")
		fmt.Printf("    sketchmerge merge -o <output dir> <merge_file> <dst_file_or_dir> <src_file_or_dir> ...\n")
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
		fmt.Printf("	  (NOT IMPLEMENTED)--dependencies (-d) analyze objects dependencies\n")
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
			mergeInfo, err := sketchmerge.ProcessFileDiff(files[0], files[1], isNice)
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			if outputToFile != "" {
				sketchmerge.WriteToFile(outputToFile, mergeInfo)
			} else {
				fmt.Println(string(mergeInfo))
			}
		} else {
			mergeInfoBaseToDst, err := sketchmerge.ProcessFileDiff(files[0], baselineVersion, isNice)
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			if outputToFile != "" {
				sketchmerge.WriteToFile(outputToFile + string(os.PathSeparator) + "dst.diff", mergeInfoBaseToDst)
			} else {
				fmt.Println(string(mergeInfoBaseToDst))
			}

			mergeInfoBaseToSrc, err := sketchmerge.ProcessFileDiff(files[1], baselineVersion, isNice)
			if err != nil {
				fmt.Printf("Error occured: %v\n", err)
				os.Exit(1)
			}

			if outputToFile != "" {
				sketchmerge.WriteToFile(outputToFile + string(os.PathSeparator) + "src.diff", mergeInfoBaseToSrc)
			} else {
				fmt.Println(string(mergeInfoBaseToSrc))
			}
		}


	}

	if opType == MergeOpType {
		files := make([]string,0)
		outputToDir := ""
		for argc := 1; argc < flag.NArg(); argc++ {
			switch flag.Arg(argc) {

			case "-o", "--output":
				argc++
				outputToDir = flag.Arg(argc)
				break
			default:
				if strings.HasPrefix(flag.Arg(argc), "--output=") {
					outputToDir = strings.TrimPrefix(flag.Arg(argc), "--output=")
				} else {
					files = append(files, flag.Arg(argc))
				}
			}

		}


		if len(files) != 3 {
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

		err := sketchmerge.ProcessFileMerge(files[0], files[1], files[2], outputToDir )

		if err!=nil {
			fmt.Printf("Error occured: %v\n", err)
			os.Exit(1)
		}

	}
}

