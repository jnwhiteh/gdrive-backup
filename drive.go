package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"

	"code.google.com/p/google-api-go-client/drive/v2"
	"code.google.com/p/google-api-go-client/googleapi"
)

var (
	ErrFolderNotFound = errors.New("folder not found")
)

func (c *DriveClient) retryAPICall(in []reflect.Value) []reflect.Value {
	// find the 'Do' method in the call object
	doMethod := in[0].Elem().MethodByName("Do")

	// call Do() and retry if we get rate limited
	for attempt := uint(0); attempt < 5; attempt++ {
		result := doMethod.Call([]reflect.Value{})
		gerror, ok := result[len(result)-1].Interface().(*googleapi.Error)
		if ok && gerror.Code == 403 {
			if gerror.Message == "Rate Limit Exceeded" || gerror.Message == "User Rate Limit Exceeded" {
				// delay and retry
				delay := getExponentialBackoffDelay(attempt, c.rand, c.randMutex)
				time.Sleep(delay)
				continue
			}
		}
		return result
	}
	// this is not reached
	return nil
}

type DriveClient struct {
	service   *drive.Service
	rand      *rand.Rand
	randMutex *sync.Mutex
}

func NewDriveClient(secretFile string, appName string, debug bool) *DriveClient {
	config := NewOAuthConfigFromFile(secretFile, drive.DriveScope)
	client := NewOAuthClient(appName, debug, config)
	service, err := drive.New(client)
	if err != nil {
		log.Fatalf("Failed to create drive client: %v", err)
	}
	return &DriveClient{
		service,
		rand.New(rand.NewSource(time.Now().UnixNano())),
		new(sync.Mutex),
	}
}

func (c *DriveClient) GetFolderByTitle(name string) (*drive.File, error) {
	query := fmt.Sprintf("mimeType = 'application/vnd.google-apps.folder' and title = '%s' and trashed != true", name)
	var fileList *drive.FileList
	var err error
	for attempt := uint(0); attempt < 5; attempt++ {
		fileList, err = c.service.Files.List().Q(query).Do()

		if err != nil && isRateLimitingError(err) {
			delay := getExponentialBackoffDelay(attempt, c.rand, c.randMutex)
			time.Sleep(delay)
			continue
		} else if err != nil {
			log.Printf("Error fetching file list: %v", err)
			return nil, err
		}
	}

	if fileList != nil && len(fileList.Items) > 1 {
		return nil, fmt.Errorf("Ambiguous folder name %v, found %d matches", name, len(fileList.Items))
	} else if len(fileList.Items) == 1 {
		return fileList.Items[0], nil
	}
	return nil, ErrFolderNotFound
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

// A RoundTripper that handles rateLimitExceeded errors
type rateLimitRetryRoundTripper struct {
	rt          http.RoundTripper
	rand        *rand.Rand
	randMutex   *sync.Mutex
	numAttempts int
}

func NewRateLimitRetryRoundTripper() http.RoundTripper {
	return &rateLimitRetryRoundTripper{
		http.DefaultTransport,
		rand.New(rand.NewSource(time.Now().UnixNano())),
		new(sync.Mutex),
		5,
	}
}

func (t *rateLimitRetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for i := 0; i < t.numAttempts; i++ {
		res, err := t.rt.RoundTrip(req)

		// Only handle 403 errors
		if res.StatusCode == 403 {
			body, err := ioutil.ReadAll(res.Body)
			// TODO: Make an error-throwing-reader
			if err != nil {
				log.Fatalf("Failed when reading response body: %v", err)
			}
			res.Body = ioutil.NopCloser(bytes.NewBuffer(body))

			if isRateLimitingResponse(body) {
				// Get a random exponential backoff delay

			}
		}
		return res, err
	}
	return nil, nil
}

func isRateLimitExceededError(err error) bool {
	gerror, ok := err.(*googleapi.Error)
	if ok && gerror.Code == 403 {
		return gerror.Message == "Rate Limit Exceeded" || gerror.Message == "User Rate Limit Exceeded"
	}
	return false
}

func (c *DriveClient) retryAboutCall(call interface {
	Do() (*drive.About, error)
}) (*drive.About, error) {
	for attempt := uint(0); attempt < 5; attempt++ {
		result, err := call.Do()
		if isRateLimitExceededError(err) {
			delay := getExponentialBackoffDelay(attempt, c.rand, c.randMutex)
			time.Sleep(delay)
			continue
		}
		return result, err
	}
	// This is unreachable
	return nil, nil
}

func (c *DriveClient) retryAppCall(call interface {
	Do() (*drive.App, error)
}) (*drive.App, error) {
	for attempt := uint(0); attempt < 5; attempt++ {
		result, err := call.Do()
		if isRateLimitExceededError(err) {
			delay := getExponentialBackoffDelay(attempt, c.rand, c.randMutex)
			time.Sleep(delay)
			continue
		}
		return result, err
	}
	// This is unreachable
	return nil, nil
}
