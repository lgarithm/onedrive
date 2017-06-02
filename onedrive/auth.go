package onedrive

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/golang/glog"
)

const (
	authURL     = `https://login.microsoftonline.com/common/oauth2/v2.0/authorize`
	tokenURL    = `https://login.microsoftonline.com/common/oauth2/v2.0/token`
	redirectURI = `http://localhost:4869/`

	scopeReadWriteAll  = `files.readwrite.all`
	scopeOfflineAccess = `offline_access`
)

var (
	scopes = []string{scopeReadWriteAll, scopeOfflineAccess}
)

func Auth() {
	c, err := loadConfig()
	if err != nil {
		glog.Exitf("Failed to load config: %v", err)
	}
	q := url.Values{}
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Add("scope", strings.Join(scopes, " "))
	u, _ := url.Parse(authURL)
	u.RawQuery = q.Encode()
	fmt.Printf("%s\n", u.String())
	{
		u, _ := url.Parse(redirectURI)
		http.HandleFunc("/", c.authAction)
		http.ListenAndServe(u.Host, nil)
	}
}

func (c *Config) authAction(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	for k, vs := range req.Header {
		for _, v := range vs {
			fmt.Printf("%s: %s\n", k, v)
		}
	}
	code := req.FormValue("code")
	fmt.Println(code)
	if err := getAcceccToken(c, code); err != nil {
		return
	}
	fmt.Fprintf(w, "DONE")
	os.Exit(0)
}

func getAcceccToken(c *Config, code string) error {
	q := url.Values{}
	q.Set("client_id", c.ClientID)
	q.Set("client_secret", c.ClientSecret)
	q.Set("redirect_uri", redirectURI)
	q.Set("grant_type", "authorization_code")
	q.Set("code", code)
	res, bs, err := postQuery(tokenURL, q)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s\n%s", res.Status, string(bs))
	}
	return SaveToken(bs)
}

func RefreshAcceccToken() error {
	token, err := loadToken()
	if err != nil {
		return err
	}
	q := url.Values{}
	c, err := loadConfig()
	if err != nil {
		return fmt.Errorf("Failed to load config: %v", err)
	}
	q.Set("client_id", c.ClientID)
	q.Set("client_secret", c.ClientSecret)
	q.Set("redirect_uri", redirectURI)
	q.Set("grant_type", "refresh_token")
	q.Set("refresh_token", token.RefreshToken)
	res, bs, err := postQuery(tokenURL, q)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s\n%s", res.Status, string(bs))
	}
	return SaveToken(bs)
}

func postQuery(api string, query url.Values) (*http.Response, []byte, error) {
	body := &bytes.Buffer{}
	body.Write([]byte(query.Encode()))
	res, err := http.DefaultClient.Post(api, `application/x-www-form-urlencoded`, body)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	return res, bs, nil
}
