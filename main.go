package main

import (
	"flag"
	"fmt"
	"log"

	"code.google.com/p/google-api-go-client/drive/v2"
)

// Flags
var (
	secretFile = flag.String("secret_file", "client_secret.json",
		"Name of a file containing OAuth client ID and secret downloaded from https://console.developers.google.com")
	debug            = flag.Bool("debug", true, "show HTTP traffic")
	localDir         = flag.String("local_dir", "", "The directory on your local machine to backup")
	remoteFolderName = flag.String("remote_folder", "", "The name of the folder on Google Drive to use for backups")
	schedule         = flag.String("schedule", "", "The schedule for backups (see documentation for valid values)")
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
	//	if remoteFolder == nil {
	//		if *createRemote {
	//			driveFile := drive.File{
	//				Title: file.base,
	//				Parents: []*drive.ParentReference{
	//					&drive.ParentReference{Id: folderId},
	//				},
	//			}
	//
	//				_, err := service.Files.Insert(req.driveFile).Media(req.localFile).Do()
	//
	//		} else {
	//			log.Fatalf("Could not find remote folder %s", *remoteFolderName)
	//		}
	//	}

	fmt.Printf("Found folder: %v", remoteFolder)
}
