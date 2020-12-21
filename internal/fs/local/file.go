package local

import (
	"fmt"
	"github.com/worldiety/go-tip/1.16/io/fs"
	"io/ioutil"
	"os"
)

var _ fs.ReadDirFile = (*file)(nil)

type file struct {
	name     string
	openFile *os.File
}

func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	files, err := ioutil.ReadDir("/"+f.name)
	if err != nil {
		return nil, err
	}

	res := make([]fs.DirEntry, 0, len(files))
	for _, info := range files {
		res = append(res, infoDelegate{info})
	}

	return res, nil
}

func (f *file) Stat() (fs.FileInfo, error) {
	info, err := os.Stat("/"+f.name)
	if err != nil {
		return nil, err
	}

	return infoDelegate{info}, nil
}

func (f *file) Read(bytes []byte) (int, error) {
	if f.openFile == nil {
		file, err := os.Open("/"+f.name)
		if err != nil {
			return 0, fmt.Errorf("unable to open file '%s': %w", f.name, err)
		}

		f.openFile = file
	}

	return f.openFile.Read(bytes)
}

func (f *file) Close() error {
	if f.openFile != nil {
		return f.openFile.Close()
	}

	return nil
}
