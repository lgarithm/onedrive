package onedrive

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path"

	"github.com/golang/glog"
)

var (
	clientID     = flag.String("client_id", "", "")
	clientSecret = flag.String("client_secret", "", "")
)

type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func CreateConfig() {
	if *clientID == "" {
		glog.Exit("-client_id is required.")
	}
	if *clientSecret == "" {
		glog.Exit("-client_secret is required.")
	}
	c := Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
	}
	bs, err := json.MarshalIndent(c, "  ", "")
	if err != nil {
		glog.Exitf("Failed to json.MarshalIndent config: %v", err)
	}
	if err := SaveConfig(bs); err != nil {
		glog.Exitf("Failed to save config: %v", err)
	}
}

func loadConfig() (*Config, error) {
	bs, err := ioutil.ReadFile(configFile())
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(bs, &c); err != nil {
		return nil, err
	}
	return &c, nil
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
