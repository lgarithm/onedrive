package onedrive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/golang/glog"
)

func Auth() {
	config, err := loadConfig()
	if err != nil {
		glog.Exitf("Failed to load config: %v", err)
	}
	fmt.Printf("%s\n", config.AuthCodeURL(""))
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		token, err := config.Exchange(context.TODO(), req.FormValue("code"))
		if err != nil {
			glog.Exit(err)
		}
		SaveToken(*token)
		os.Exit(0)
	})
	u, _ := url.Parse(config.RedirectURL)
	http.ListenAndServe(u.Host, nil)
}

func RefreshAcceccToken() error {
	token, err := loadToken()
	if err != nil {
		return err
	}
	q := url.Values{}
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("Failed to load config: %v", err)
	}
	q.Set("client_id", config.ClientID)
	q.Set("client_secret", config.ClientSecret)
	q.Set("redirect_uri", config.RedirectURL)
	q.Set("grant_type", "refresh_token")
	q.Set("refresh_token", token.RefreshToken)
	res, err := postQuery(config.Endpoint.TokenURL, q)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := readJSON(res.Body, &token); err != nil {
		return err
	}
	return SaveToken(*token)
}

func postQuery(api string, query url.Values) (*http.Response, error) {
	body := &bytes.Buffer{}
	body.Write([]byte(query.Encode()))
	res, err := http.DefaultClient.Post(api, `application/x-www-form-urlencoded`, body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return res, errors.New(res.Status)
	}
	return res, nil
}
