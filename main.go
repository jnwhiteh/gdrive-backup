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
)

func main() {
	flag.Parse()

	config := NewOAuthConfigFromFile(*secretFile, drive.DriveScope)
	client := NewOAuthClient("gdrive-backup", *debug, config)
	service, err := drive.New(client)
	if err != nil {
		log.Fatalf("Failed to create Drive client: %v", err)
	}

	remoteFolder := getFolderByName(service, *remoteFolderName)
	if remoteFolder == nil {
		if *createRemote {
			log.Printf("Creating new remote folder %s", *remoteFolderName)
			driveFile := drive.File{
				Title:    *remoteFolderName,
				MimeType: "application/vnd.google-apps.folder",
			}

			remoteFolder, err = service.Files.Insert(&driveFile).Do()
			if err != nil {
				log.Fatalf("Failed to create remote folder %s", *remoteFolderName)
			}
		}
	} else {
		fmt.Printf("Found folder: %v", remoteFolder)
	}
}
