package local

import "github.com/worldiety/go-tip/1.16/io/fs"

// assert interface
var _ fs.FS = (*FS)(nil)

type FS struct {
}

func (f FS) Open(name string) (fs.File, error) {
	return &file{
		name: name,
	}, nil
}

func Get() FS {
	return FS{}
}
