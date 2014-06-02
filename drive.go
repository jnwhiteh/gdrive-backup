package main

import (
	"fmt"
	"log"

	"code.google.com/p/google-api-go-client/drive/v2"
)

func getFolderByName(service *drive.Service, name string) *drive.File {
	query := fmt.Sprintf("mimeType = 'application/vnd.google-apps.folder' and title = '%s' and trashed != true", name)
	fileList, err := service.Files.List().Q(query).Do()
	if err != nil {
		log.Printf("Error fetching file list: %v", err)
		return nil
	}
	if len(fileList.Items) != 1 {
		log.Printf("Folder name %v is ambiguous, found %d matches", name, len(fileList.Items))
		return nil
	}
	return fileList.Items[0]
}
