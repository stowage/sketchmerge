package sketchmerge

import (
	_"bytes"
	_"fmt"
	"io"
	_"log"
	"os"
	"path/filepath"
	"strings"
	"archive/zip"
	"time"
	_"github.com/klauspost/compress/flate"
	_"github.com/klauspost/compress/zip"
)

func Zipit(source, target string) error {
	defer TimeTrack(time.Now(), "Zipit " + target)
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)

	/*var fw *flate.Writer

	// Register the deflator.
	archive.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		var err error
		if fw == nil {
			// Creating a flate compressor for every file is
			// expensive, create one and reuse it.
			fw, err = flate.NewWriter(out, flate.BestCompression)
		} else {
			fw.Reset(out)
		}
		return fw, err
	})*/


	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}


	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}


		header, err := zip.FileInfoHeader(info)

		if err != nil {
			return err
		}

		//header.CreatorVersion = 0

		if baseDir != "" {
			header.Name = strings.TrimPrefix(path, source + string(os.PathSeparator)) //filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if header.Name == source  {
			return  nil
		}
		if info.IsDir() {
			header.Name += "/"

		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}