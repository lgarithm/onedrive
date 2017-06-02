package onedrive

import (
	"bytes"
	"flag"
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
	clientID     = flag.String("client_id", "", "")
	clientSecret = flag.String("client_secret", "", "")

	scopes = []string{scopeReadWriteAll, scopeOfflineAccess}
)

func Auth() {
	if *clientID == "" {
		glog.Exit("-client_id is required.")
	}
	if *clientSecret == "" {
		glog.Exit("-client_secret is required.")
	}
	q := url.Values{}
	q.Set("client_id", *clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Add("scope", strings.Join(scopes, " "))
	u, _ := url.Parse(authURL)
	u.RawQuery = q.Encode()
	fmt.Printf("%s\n", u.String())
	{
		u, _ := url.Parse(redirectURI)
		http.HandleFunc("/", authAction)
		http.ListenAndServe(u.Host, nil)
	}
}

func authAction(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	for k, vs := range req.Header {
		for _, v := range vs {
			fmt.Printf("%s: %s\n", k, v)
		}
	}
	code := req.FormValue("code")
	fmt.Println(code)
	if err := getAcceccToken(code); err != nil {
		return
	}
	fmt.Fprintf(w, "DONE")
	os.Exit(0)
}

func getAcceccToken(code string) error {
	q := url.Values{}
	q.Set("client_id", *clientID)
	q.Set("client_secret", *clientSecret)
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
	q.Set("client_id", *clientID)
	q.Set("client_secret", *clientSecret)
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
