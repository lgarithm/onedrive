package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/golang/glog"
	"golang.org/x/oauth2"
)

const (
	endpoint = `https://graph.microsoft.com/v1.0/me/drive`
)

var (
	remotePath = flag.String("path", "upload", "remote folder")
)

// Client is a client for https://onedrive.live.com
type Client struct {
	client   *http.Client
	endpoint url.URL
}

// New creates a Client
func New() (*Client, error) {
	e, err := url.Parse(endpoint)
	if err != nil {
		glog.Exit(err)
	}
	token, err := loadToken()
	if err != nil {
		return nil, err
	}
	c := Client{
		client:   oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(token)),
		endpoint: *e,
	}
	return &c, nil
}

// Upload uploads a file
func (c Client) Upload(localfile string) (*Item, error) {
	const limit = 4 * 1024 * 1024
	info, err := os.Stat(localfile)
	if err != nil {
		return nil, err
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("TODO: support session upload")
	}
	return c.simpleUpload(localfile)
}

// Download downloads a file
func (c Client) Download(itemPath ...string) error {
	// GET /drive/root:/{item-path}
	u := c.endpoint
	{
		u.Path += "/root:"
		for _, p := range itemPath {
			u.Path += "/" + p
		}
	}
	res, err := c.client.Get(u.String())
	if err != nil {
		return fmt.Errorf("Failed to get Item meta data: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", res.Status)
	}
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var item Item
	if err := json.Unmarshal(bs, &item); err != nil {
		return err

	}
	if item.File == nil {
		return fmt.Errorf("NOT a file")
	}
	return c.downloadByID(item.ID, item.Name)
}

// https://dev.onedrive.com/items/upload_put.htm
func (c Client) simpleUpload(localfile string) (*Item, error) {
	lf, err := os.Open(localfile)
	if err != nil {
		return nil, err
	}
	name := path.Base(localfile)
	u := c.endpoint
	// PUT /drive/root:/{parent-path}/{filename}:/content
	u.Path += path.Join("/root:", *remotePath, name+":", "content")
	body := &bytes.Buffer{}
	io.Copy(body, lf)
	req, err := http.NewRequest("PUT", u.String(), body)
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%s\n%s", res.Status, string(bs))
	}
	var item Item
	if err := json.Unmarshal(bs, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// https://dev.onedrive.com/items/upload_large_files.htm
func (c Client) uploadFile(localfile string) error {
	name := path.Base(localfile)
	u := c.endpoint
	// POST /drive/root:/{path_to_item}:/createUploadSession
	var session UploadSession
	{
		u.Path += path.Join("/root:", name+":", "createUploadSession")
		body := &bytes.Buffer{}
		body.Write([]byte("{}"))
		req, err := http.NewRequest("POST", u.String(), body)
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			return err
		}
		res, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		bs, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(bs, &session); err != nil {
			return err
		}
	}
	// glog.Infof("Upload session created, URL: %q", session.UploadURL)
	{
		info, err := os.Stat(localfile)
		if err != nil {
			return err
		}
		lf, err := os.Open(localfile)
		if err != nil {
			return err
		}
		body := &bytes.Buffer{}
		io.Copy(body, lf)
		req, err := http.NewRequest("PUT", session.UploadURL, body)
		n := int(info.Size())
		req.Header.Set("Content-Length", strconv.Itoa(n))
		req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", 0, n-1, n))
		res, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		_, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

// https://dev.onedrive.com/items/download.htm#http-request
func (c Client) downloadByID(id string, localfile string) error {
	f, err := os.Create(localfile)
	if err != nil {
		return err
	}
	// GET /drive/items/{item-id}/content
	u := c.endpoint
	u.Path += fmt.Sprintf("/items/%s/content", id)
	res, err := c.client.Get(u.String())
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Invalid Status: %s", res.Status)
	}
	_, err = io.Copy(f, res.Body)
	return err
}

// List lists items
func (c Client) List(dirs ...string) ([]Item, error) {
	u := c.endpoint
	if len(dirs) == 0 {
		u.Path += "/root/children"
	} else {
		ps := []string{"/root:"}
		for _, d := range dirs {
			ps = append(ps, d+":")
		}
		ps = append(ps, "children")
		u.Path += path.Join(ps...)
	}
	glog.Infof("list path: %q", u.Path)
	res, err := c.client.Get(u.String())
	if err != nil {
		glog.Errorf("Failed to ListDrivers: %v", err)
		return nil, err
	}
	defer res.Body.Close()
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		glog.Errorf("Failed to ListDrivers: %v", err)
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s\n%s", res.Status, string(bs))
	}
	var result struct {
		Value []Item `json:"value"`
	}
	if err := json.Unmarshal(bs, &result); err != nil {
		return nil, err
	}
	return result.Value, err
}
