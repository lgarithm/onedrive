package onedrive

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"time"

	"golang.org/x/oauth2"
)

func loadToken() (*oauth2.Token, error) {
	bs, err := ioutil.ReadFile(tokenFile())
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(bs, &token); err != nil {
		return nil, err
	}
	var tokenExpiry struct {
		Seconds int `json:"expires_in"`
	}
	if err := json.Unmarshal(bs, &tokenExpiry); err == nil {
		if info, err := os.Stat(tokenFile()); err == nil {
			const margin = 60
			token.Expiry = info.ModTime().Add(time.Duration(tokenExpiry.Seconds-margin) * time.Second)
		}
	}
	return &token, nil
}

// SaveToken saves token file to default location
func SaveToken(bs []byte) error {
	var token oauth2.Token
	if err := json.Unmarshal(bs, &token); err != nil {
		return err
	}
	filename := tokenFile()
	os.MkdirAll(path.Dir(filename), os.ModePerm)
	return ioutil.WriteFile(filename, bs, os.ModePerm)
}

func tokenFile() string {
	return path.Join(os.Getenv("HOME"), ".onedrive", "access_token.json")
}
