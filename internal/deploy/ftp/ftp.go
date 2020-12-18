package ftp

import (
	"crypto/tls"
	"fmt"
	"github.com/golangee/log"
	"gopkg.in/dutchcoders/goftp.v1"
	"strconv"
)

func Upload(host, login, password, localDir, remoteDir string, port int, debug, insecureSkipVerify bool) error {
	ftp, err := goftp.Connect(host + ":" + strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("unable to connect: %w", err)
	}

	defer ftp.Close()

	if debug {
		log.Println("ftp connected to " + host)
	}

	config := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		ClientAuth:         tls.RequestClientCert,
	}

	if !insecureSkipVerify {
		config.ServerName = host
	}

	if err = ftp.AuthTLS(config); err != nil {
		return fmt.Errorf("unable to AuthTLS: %w", err)
	}

	if err = ftp.Login(login, password); err != nil {
		return fmt.Errorf("unable to login: %w", err)
	}

	if err = ftp.Cwd(remoteDir); err != nil {
		return fmt.Errorf("unable to change remote dir: %w", err)
	}

	if debug {
		files, err := ftp.List("")
		if err != nil {
			return fmt.Errorf("unable to list files")
		}

		for _, file := range files {
			log.Println(file)
		}
	}

	if err := ftp.Upload(localDir); err != nil {
		return fmt.Errorf("unable to upload local dir '%s': %w", err)
	}

	return nil
}

/*
func uploadDir(ftp *goftp.FTP, remoteDir, localDir string, debug bool) error {
	err := ftp.Cwd(remoteDir)
	if err != nil {
		return fmt.Errorf("cannot cwd to '%s': %w", remoteDir, err)
	}

	files, err := ioutil.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("cannot read local dir '%s': %w", localDir, err)
	}

	for _, file := range files {
		remotePath := remoteDir+"/"+file.Name()
		if file.IsDir() {
			_ = ftp.Mkd(file.Name())
			if err := uploadDir(ftp,remotePath, filepath.Join(localDir, file.Name()), debug); err != nil {
				return err
			}
		}else{
			ftp.Upload()
		}
	}
}*/
