package sftp

import (
	"fmt"
	"github.com/pkg/sftp"
	"github.com/worldiety/go-tip/1.16/io/fs"
	"golang.org/x/crypto/ssh"
	"os"
	"time"
)

// Options to connect to an FTP over SSH service, respective the SSH file Transfer Protocol.
type Options struct {
	Host     string
	Port     int // Port default is 22.
	User     string
	Password string
	Callback ssh.HostKeyCallback // Callback default is ssh.InsecureIgnoreHostKey which must be considered insecure.
}

// assert interface
var _ fs.ReadDirFS = (*FS)(nil)
var _ fs.SubFS = (*FS)(nil)

type FS struct {
	prefix string
	client *sftp.Client
}

func (f *FS) Sub(dir string) (fs.FS, error) {
	return &FS{
		prefix: f.prefix + "/" + dir,
		client: f.client,
	}, nil
}

func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = f.prefix + "/" + name
	tmp := &file{
		parent: f,
		name:   name,
	}

	return tmp.ReadDir(0)
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error.
// If path is already a directory, MkdirAll does nothing and returns nil.
// If path contains a regular file, an error is returned
func (f *FS) MkdirAll(name string) error {
	name = f.prefix + "/" + name
	return f.client.MkdirAll(name)
}

// Mkdir creates the specified directory. An error will be returned if a file or
// directory with the specified path already exists, or if the directory's
// parent folder does not exist (the method cannot create complete paths).
func (f *FS) Mkdir(name string) error {
	name = f.prefix + "/" + name
	return f.client.Mkdir(name)
}

func (f *FS) RemoveAll(name string) error {
	name = f.prefix + "/" + name
	stat, err := f.client.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("unable to stat: %w", err)
	}

	if stat.IsDir() {
		files, err := f.client.ReadDir(name)
		if err != nil {
			return fmt.Errorf("unable to readdir: %w", err)
		}

		for _, info := range files {
			if err := f.RemoveAll(name + "/" + info.Name()); err != nil {
				return err
			}
		}
		if err := f.client.RemoveDirectory(name); err != nil {
			return fmt.Errorf("unable to remove cleared directory: %w", err)
		}

	} else {
		return f.client.Remove(name)
	}

	return nil
}

func (f *FS) Open(name string) (fs.File, error) {
	name = f.prefix + "/" + name
	return &file{
		parent: f,
		name:   name,
	}, nil
}

func (f *FS) OpenFile(name string, flag int, perm os.FileMode) (fs.File, error) {
	name = f.prefix + "/" + name
	return &file{
		parent: f,
		name:   name,
		flag:   flag,
		perm:   perm,
	}, nil
}

func Connect(opts Options) (*FS, error) {
	if opts.Port == 0 {
		opts.Port = 22
	}

	if opts.Callback == nil {
		opts.Callback = ssh.InsecureIgnoreHostKey()
	}

	config := &ssh.ClientConfig{
		User:            opts.User,
		Auth:            []ssh.AuthMethod{ssh.Password(opts.Password)},
		Timeout:         30 * time.Second,
		HostKeyCallback: opts.Callback,
	}

	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to SSH service: %w", err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to create sftp client: %w", err)
	}

	return &FS{client: client}, nil
}
