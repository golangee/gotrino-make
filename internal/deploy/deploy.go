package deploy

import (
	"fmt"
	"github.com/golangee/gotrino-make/internal/fs/local"
	"github.com/golangee/gotrino-make/internal/fs/sftp"
	"github.com/golangee/log"
	"github.com/worldiety/go-tip/1.16/io/fs"
	"io"
	"os"
)

var Debug = false

type MkdirAll interface {
	MkdirAll(name string) error
}

type OpenFile interface {
	OpenFile(name string, flag int, perm os.FileMode) (fs.File, error)
}

type RemoveAll interface {
	RemoveAll(name string) error
}

func SyncSFTP(remoteDir, localDir string, host, user, password string, port int) error {
	sftpFS, err := sftp.Connect(sftp.Options{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	})

	if err != nil {
		return fmt.Errorf("unable to connect sftp FS: %w", err)
	}

	dst, err := fs.Sub(sftpFS, remoteDir)
	if err != nil {
		return fmt.Errorf("unable to sub dst: %w", err)
	}

	src, err := fs.Sub(local.Get(), localDir)
	if err != nil {
		return fmt.Errorf("unable to sub src: %w", err)
	}

	return Sync(dst.(fs.ReadDirFS), src.(fs.ReadDirFS))
}

func Sync(dst, src fs.ReadDirFS) error {
	srcFiles, err := src.ReadDir(".")
	if err != nil {
		return err
	}

	for _, file := range srcFiles {
		if file.IsDir() {
			if Debug {
				log.Println(fmt.Sprintf("copy dir: %s", file.Name()))
			}

			if err := dst.(MkdirAll).MkdirAll(file.Name()); err != nil {
				return fmt.Errorf("unable to ensure directory in dst: %w", err)
			}

			subSrc, err := fs.Sub(src, file.Name())
			if err != nil {
				return fmt.Errorf("unable to subroot src: %w", err)
			}

			subDst, err := fs.Sub(dst, file.Name())
			if err != nil {
				return fmt.Errorf("unable to subroot dst: %w", err)
			}

			if err := Sync(subDst.(fs.ReadDirFS), subSrc.(fs.ReadDirFS)); err != nil {
				return err
			}
		} else {
			if Debug {
				log.Println(fmt.Sprintf("copy file: %s", file.Name()))
			}

			dstFile, err := dst.(OpenFile).OpenFile(file.Name(), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
			if err != nil {
				return fmt.Errorf("unable to write dst file: %w", err)
			}

			srcFile, err := src.Open(file.Name())
			if err != nil {
				_ = dstFile.Close()
				return fmt.Errorf("unable to open src file: %w", err)
			}

			if _, err := io.Copy(dstFile.(io.Writer), srcFile); err != nil {
				_ = srcFile.Close()
				_ = dstFile.Close()
				return fmt.Errorf("unable to copy src to dst: %w", err)
			}

			_ = srcFile.Close()
			_ = dstFile.Close()
		}

	}

	// check extra files in dst
	dstFiles, err := dst.ReadDir(".")
	if err != nil {
		return err
	}

	for _, file := range dstFiles {
		has := false
		for _, srcFile := range srcFiles {
			if srcFile.Name() == file.Name() {
				has = true
				break
			}
		}

		if !has {
			if Debug {
				log.Println(fmt.Sprintf("removing extra file: %s, isDir=%v", file.Name(), file.IsDir()))
			}

			if err := dst.(RemoveAll).RemoveAll(file.Name()); err != nil {
				return fmt.Errorf("unable to remove: %s: %w", file.Name(), err)
			}
		}

	}

	return nil
}
