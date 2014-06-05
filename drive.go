package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"code.google.com/p/google-api-go-client/drive/v2"
)

func findFolder(service *drive.Service, name string) *drive.File {
	query := fmt.Sprintf("mimeType = 'application/vnd.google-apps.folder' and title = '%s' and trashed != true", name)
	fileList, err := service.Files.List().Q(query).Do()
	if err != nil {
		log.Printf("Error fetching file list: %v", err)
		return nil
	}
	if len(fileList.Items) > 1 {
		log.Printf("Folder name %v is ambiguous, found %d matches", name, len(fileList.Items))
		return nil
	} else if len(fileList.Items) == 0 {
		return nil
	}

	return fileList.Items[0]
}

func newFileWithParent(name string, parentId string) *drive.File {
	driveFile := drive.File{
		Title: name,
	}
	if parentId != "" {
		driveFile.Parents = []*drive.ParentReference{
			&drive.ParentReference{Id: parentId},
		}
	}
	return &driveFile
}

func mkdir(service *drive.Service, name string, parentId string) (*drive.File, error) {
	driveFile := newFileWithParent(name, parentId)
	driveFile.MimeType = "application/vnd.google-apps.folder"
	return service.Files.Insert(driveFile).Do()
}

func uploadFile(service *drive.Service, name string, parentId, localFilename string) (*drive.File, error) {
	localFile, err := os.Open(localFilename)
	if err != nil {
		return nil, err
	}
	defer localFile.Close()

	driveFile := newFileWithParent(name, parentId)
	return service.Files.Insert(driveFile).Media(localFile).Do()
}

func getRemoteFiles(service *drive.Service, parentId string) ([]*drive.File, error) {
	if parentId == "" {
		parentId = "root"
	}

	var files []*drive.File
	childCall := service.Children.List(parentId).Q("mimeType != 'application/vnd.google-apps.folder' and trashed != true")
	childList, err := childCall.Do()
	for childList != nil && childList.Items != nil {
		if err != nil {
			return nil, err
		}
		for _, child := range childList.Items {
			file, err := service.Files.Get(child.Id).Do()
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
		if childList.NextPageToken == "" {
			break
		}
		childList, err = childCall.PageToken(childList.NextPageToken).Do()
	}

	return files, nil
}

func toJson(val interface{}) string {
	buf, err := json.Marshal(val)
	if err != nil {
		return ""
	}
	return string(buf)
}
