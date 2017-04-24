package sketchmerge

import (
	"os"
	"io"
)

func CopyFile(src, dst string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()

	destFile, err := os.Create(dst) // creates if file doesn't exist
	if err != nil {
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile) // check first var for number of bytes copied
	if err != nil {
		return
	}

	err = destFile.Sync()
	return
}
