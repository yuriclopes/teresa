package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/jhoonb/archivex"
	"github.com/mozillazg/request"
	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"
)

// TODO: create a file like gitignore to upload a package without unecessary files, or get from app config?!?!?

var createDeployCmd = &cobra.Command{
	Use:   "create APP_FOLDER",
	Short: "deploy an app",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			log.Debug("App folder not provided")
			return newInputError("App Folder not provided")
		}
		return createDeploy(filepath.Clean(args[0]))
	},
}

func createDeploy(p string) error {
	if p == "" {
		log.Error("App folder not provided")
		return newSysError("App folder not provided")
	}
	tar, err := createTempArchiveToUpload(p)
	if err != nil {
		return err
	}
	h := &http.Client{}
	req := request.NewRequest(h)
	file, _ := os.Open(tar)
	req.Files = []request.FileField{
		request.FileField{FieldName: "apparchive", FileName: filepath.Base(tar), File: file},
	}
	cluster, err := getCurrentCluster()
	if err != nil {
		return err
	}
	req.Headers = map[string]string{
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", cluster.Token),
	}
	resp, err := req.Post(fmt.Sprintf("%s/deploy", cluster.Server))
	if err != nil {
		log.WithError(err).Error("Error when uploading an app archive to start a deploy")
		return newSysError("Error when trying to do this action")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		log.Debug("User not logged... informing to retry after login")
		return newSysError("You need to login before do this action")
	}
	if resp.StatusCode > 401 && resp.StatusCode <= 500 {
		fields := logrus.Fields{
			"statusCode": resp.StatusCode,
		}
		if body, err := resp.Text(); err == nil {
			fields["contentBody"] = body
		}
		log.WithFields(fields).Error("Http status diff from 200 when requesting a login")
		return newSysError("Error when trying to do this action")
	}
	return nil
}

// create a temporary archive file of the app to deploy and return the path of this file
func createTempArchiveToUpload(source string) (path string, err error) {
	id := uuid.NewV1()
	base := filepath.Base(source)
	path = filepath.Join(archiveTempFolder, fmt.Sprintf("%s_%s.tar.gz", base, id))
	if err = createArchive(source, path); err != nil {
		return "", err
	}
	return
}

// create an archive of the source folder
func createArchive(source string, target string) error {
	log.WithField("dir", source).Debug("Creating archive")
	base := filepath.Dir(source)
	dir, err := os.Stat(base)
	if err != nil {
		log.WithError(err).WithField("baseDir", base).Error("Dir not found to create an archive")
		return err
	} else if !dir.IsDir() {
		log.WithField("baseDir", base).Error("Path to create the app archive isn't a directory")
		return errors.New("Path to create the app archive isn't a directory")
	}
	tar := new(archivex.TarFile)
	tar.Create(target)
	tar.AddAll(source, true)
	tar.Close()
	return nil
}

func init() {
	// setClusterCmd.Flags().StringVarP(&serverFlag, "server", "s", "", "URI of the server")
	// setClusterCmd.Flags().BoolVarP(&currentFlag, "default", "d", false, "Is the default server")
	deployCmd.AddCommand(createDeployCmd)
}