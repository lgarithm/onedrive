package onedrive

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

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
	return &token, nil
}

func saveToken(bs []byte) error {
	var token oauth2.Token
	if err := json.Unmarshal(bs, &token); err != nil {
		return err
	}
	return ioutil.WriteFile(tokenFile(), bs, os.ModePerm)
}

func tokenFile() string {
	return path.Join(os.Getenv("HOME"), ".onedrive", "access_token.json")
}