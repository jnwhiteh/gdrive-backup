package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"code.google.com/p/goauth2/oauth"
)

type clientCredentials struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURI      string `json:"auth_uri"`
	TokenURI     string `json:"token_uri"`
}

func getClientCredentials(filename string) clientCredentials {
	creds := &struct {
		Installed clientCredentials `json:"installed"`
	}{}

	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading %q: %v", filename, err)
	}
	err = json.Unmarshal(contents, creds)
	if err != nil {
		log.Fatalf("Could not decode client credentials: %v", err)
	}
	return creds.Installed
}

func NewOAuthConfigFromFile(filename, scope string) *oauth.Config {
	credentials := getClientCredentials(filename)
	return NewOAuthConfig(credentials, scope)
}

// NewOAuthConfig creates a new default configuration for authentication
// against the Google OAuth2 API
func NewOAuthConfig(credentials clientCredentials, scope string) *oauth.Config {
	return &oauth.Config{
		ClientId:     credentials.ClientId,
		ClientSecret: credentials.ClientSecret,
		Scope:        scope,
		AuthURL:      credentials.AuthURI,
		TokenURL:     credentials.TokenURI,
	}
}

// New OAuthClient creates a new client against the Google OAuth2 API
func NewOAuthClient(appName string, debug bool, config *oauth.Config) *http.Client {
	cacheFile := tokenCacheFilename(appName, config)
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token = tokenFromWeb(debug, config)
		saveToken(cacheFile, token)
	} else {
		log.Printf("Using cached token %#v from %q", token, cacheFile)
	}

	t := &oauth.Transport{
		Token:     token,
		Config:    config,
		Transport: condDebugTransport(debug, http.DefaultTransport),
	}
	return t.Client()
}

// tokenCacheFilename returns the local cache filename for a given oauth
// configuration tuple (id, secret, scope).
func tokenCacheFilename(appName string, config *oauth.Config) string {
	hash := fnv.New32a()
	hash.Write([]byte(config.ClientId))
	hash.Write([]byte(config.ClientSecret))
	hash.Write([]byte(config.Scope))
	fn := fmt.Sprintf("%s-token%v", appName, hash.Sum32())
	return filepath.Join(osUserCacheDir(), url.QueryEscape(fn))
}

// osUserCacheDir returns the cache directory used for oAuth tokens depending
// on operating system.
func osUserCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	case "linux", "freebsd":
		return filepath.Join(os.Getenv("HOME"), ".cache")
	}
	log.Printf("TODO: osUserCacheDir on GOOS %q", runtime.GOOS)
	return "."
}

// tokenFromFile returns the oAuth token stored in a given filename.
func tokenFromFile(file string) (*oauth.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := new(oauth.Token)
	err = gob.NewDecoder(f).Decode(t)
	return t, err
}

// saveToken stores an oAuth token in the given filename.
func saveToken(file string, token *oauth.Token) {
	f, err := os.Create(file)
	if err != nil {
		log.Printf("Warning: failed to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(token)
}

// tokenFromWeb attempts to authorize the application by directing the user to
// Google. This spawns a web server on localhost which the user is eventually
// redirected to.
func tokenFromWeb(debug bool, config *oauth.Config) *oauth.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		log.Printf("no code")
		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authUrl := config.AuthCodeURL(randState)
	go openUrl(authUrl)
	log.Printf("Authorize this app at: %s", authUrl)
	code := <-ch
	log.Printf("Got code: %s", code)

	t := &oauth.Transport{
		Config:    config,
		Transport: condDebugTransport(debug, http.DefaultTransport),
	}
	_, err := t.Exchange(code)
	if err != nil {
		log.Fatalf("Token exchange error: %v", err)
	}
	return t.Token
}

func openUrl(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser.")
}

func condDebugTransport(debug bool, rt http.RoundTripper) http.RoundTripper {
	if debug {
		return &LoggingRoundTripper{os.Stdout, rt}
	}
	return rt
}
