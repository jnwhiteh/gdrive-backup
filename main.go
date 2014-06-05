package main

import (
	"flag"
	"fmt"
	"log"

	"code.google.com/p/google-api-go-client/drive/v2"
)

// Options
var (
	secretFile = flag.String("secret_file", "client_secret.json",
		"Name of a file containing OAuth client ID and secret downloaded from https://console.developers.google.com")
	debug            = flag.Bool("debug", true, "show HTTP traffic")
	localDir         = flag.String("local_dir", ".", "The directory on your local machine that should be backed up")
	remoteFolderName = flag.String("remote_folder", "gdrive-backups", "The name of the folder on Google Drive to use for backups")
	createRemote     = flag.Bool("create_remote", true, "Create the remote directory if it does not already exist")
	workers          = flag.Int("workers", 1, "Number of concurrent uploads")
)

func main() {
	flag.Parse()

	config := NewOAuthConfigFromFile(*secretFile, drive.DriveScope)
	client := NewOAuthClient("gdrive-backup", *debug, config)
	service, err := drive.New(client)
	if err != nil {
		log.Fatalf("Failed to create Drive client: %v", err)
	}

	remoteFolder := findFolder(service, *remoteFolderName)
	if remoteFolder == nil {
		if *createRemote {
			log.Printf("Creating new remote folder %s", *remoteFolderName)
			remoteFolder, err = mkdir(service, *remoteFolderName, "")
		}
	} else {
		fmt.Printf("Found folder: %v", remoteFolder)
	}

	localFiles := getLocalFiles(*localDir)
	fmt.Printf("Local files:\n")
	for _, file := range localFiles {
		fmt.Printf("%s\t%s\n", file.md5, file.filename)
	}

	remoteFiles, err := getRemoteFiles(service, remoteFolder.Id)
	if err != nil {
		log.Fatalf("Could not fetch remote files: %s", err)
	}

	fmt.Printf("Remote files:\n")
	for _, file := range remoteFiles {
		fmt.Printf("%s\t%s\n", file.Md5Checksum, file.Title)
	}

	fileHashes := make(map[string]bool)
	for _, file := range remoteFiles {
		fileHashes[file.Md5Checksum] = true
	}

	var toUpload []localFile
	for _, file := range localFiles {
		if present, ok := fileHashes[file.md5]; !ok || !present {
			toUpload = append(toUpload, file)
		}
	}

	log.Printf("Skipping %d files, uploading %d", len(localFiles)-len(toUpload), len(toUpload))
	if len(toUpload) > 0 {
		work := make(chan localFile, 1)
		done := make(chan struct{}, len(toUpload))
		for i := 0; i < *workers; i++ {
			go uploadWorker(service, remoteFolder, work, done)
		}

		for _, file := range toUpload {
			select {
			case work <- file:
			}
		}

		for i := 0; i < len(toUpload); i++ {
			<-done
		}
	} else {
		log.Printf("No files to upload")
	}
}

func uploadWorker(service *drive.Service, remoteFolder *drive.File, work <-chan localFile, done chan struct{}) {
	for file := range work {
		log.Printf("Uploading %s", file.filename)
		_, err := uploadFile(service, file.Name(), remoteFolder.Id, file.filename)
		if err != nil {
			log.Fatalf("Failed when uploading file %s: %s", file.filename, err)
		}
		log.Printf("Finished uploading %s", file.filename)
		done <- struct{}{}
	}
}
