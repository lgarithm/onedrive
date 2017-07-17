package onedrive

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/golang/glog"
	"golang.org/x/oauth2"
)

// Config is a subset of oauth2.Config
type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (c Config) toOauthConfig() oauth2.Config {
	const (
		authURL            = `https://login.microsoftonline.com/common/oauth2/v2.0/authorize`
		tokenURL           = `https://login.microsoftonline.com/common/oauth2/v2.0/token`
		redirectURL        = `http://localhost:4869/`
		scopeReadWriteAll  = `files.readwrite.all`
		scopeOfflineAccess = `offline_access`
	)
	return oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		RedirectURL: redirectURL,
		Scopes:      []string{scopeReadWriteAll, scopeOfflineAccess},
	}
}

// CreateConfig creates Config file at default location
func CreateConfig(clientID, clientSecret string) {
	if clientID == "" {
		glog.Exit("-client_id is required.")
	}
	if clientSecret == "" {
		glog.Exit("-client_secret is required.")
	}
	c := Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	bs, err := json.MarshalIndent(c, "  ", "")
	if err != nil {
		glog.Exitf("Failed to json.MarshalIndent config: %v", err)
	}
	if err := SaveConfig(bs); err != nil {
		glog.Exitf("Failed to save config: %v", err)
	}
}

func loadConfig() (*oauth2.Config, error) {
	bs, err := ioutil.ReadFile(configFile())
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(bs, &c); err != nil {
		return nil, err
	}
	config := c.toOauthConfig()
	return &config, nil
}

// SaveConfig saves config file to default location
func SaveConfig(bs []byte) error {
	var c Config
	if err := json.Unmarshal(bs, &c); err != nil {
		return err
	}
	filename := configFile()
	os.MkdirAll(path.Dir(filename), os.ModePerm)
	return ioutil.WriteFile(filename, bs, os.ModePerm)
}

func configFile() string {
	return path.Join(os.Getenv("HOME"), ".onedrive", "config.json")
}
