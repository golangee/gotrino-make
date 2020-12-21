package sftp

import (
	"fmt"
	"github.com/pkg/sftp"
	"github.com/worldiety/go-tip/1.16/io/fs"
	"os"
)

var _ fs.ReadDirFile = (*file)(nil)

type file struct {
	parent   *FS
	name     string
	openFile *sftp.File
	flag     int
	perm     os.FileMode
}

// ReadDir reads the directory named by dirname and returns a list of
// directory entries.
func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	files, err := f.parent.client.ReadDir(f.name)
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
	info, err := f.parent.client.Stat(f.name)
	if err != nil {
		return nil, err
	}

	return infoDelegate{info}, nil
}

// Read follows io.Reader semantics.
func (f *file) Read(bytes []byte) (int, error) {
	if f.openFile == nil {
		file, err := f.parent.client.Open(f.name)
		if err != nil {
			return 0, fmt.Errorf("unable to open file '%s': %w", f.name, err)
		}

		f.openFile = file
	}

	return f.openFile.Read(bytes)
}

// Write follows io.Writer semantics.
func (f *file) Write(bytes []byte) (int, error) {
	if f.openFile == nil {
		file, err := f.parent.client.OpenFile(f.name, f.flag)
		if err != nil {
			return 0, fmt.Errorf("unable to openFile file '%s': %w", f.name, err)
		}

		f.openFile = file
	}

	return f.openFile.Write(bytes)
}

// Close closes the File, rendering it unusable for I/O.
func (f *file) Close() error {
	if f.openFile != nil {
		return f.openFile.Close()
	}

	return nil
}
