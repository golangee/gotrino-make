package sftp

import (
	"github.com/worldiety/go-tip/1.16/io/fs"
	"os"
)

var _ fs.FileInfo = infoDelegate{}

var _ fs.DirEntry = infoDelegate{}

type infoDelegate struct {
	os.FileInfo
}

func (i infoDelegate) Type() fs.FileMode {
	return i.Mode()
}

func (i infoDelegate) Info() (fs.FileInfo, error) {
	return i, nil
}

func (i infoDelegate) Mode() fs.FileMode {
	return fs.FileMode(i.FileInfo.Mode())
}
