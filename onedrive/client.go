package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/oauth2"
)

const (
	endpoint = `https://graph.microsoft.com/v1.0/me/drive`
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
	if !token.Valid() {
		glog.Info("Refreshing AcceccToken")
		RefreshAcceccToken()
		token, err = loadToken()
	}
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
func (c Client) Upload(localfile string, dirs ...string) (*Item, error) {
	const limit = 4 * 1024 * 1024
	info, err := os.Stat(localfile)
	if err != nil {
		return nil, err
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("TODO: support session upload")
	}
	return c.simpleUpload(localfile, dirs...)
}

// GetFile gets a file as bytes
func (c Client) GetFile(itemPath ...string) ([]byte, error) {
	item, err := c.getFileItem(itemPath...)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	if err := c.downloadByID(item.ID, b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Download downloads a file
func (c Client) Download(itemPath ...string) error {
	item, err := c.getFileItem(itemPath...)
	if err != nil {
		return err
	}
	w, err := os.Create(item.Name)
	if err != nil {
		return err
	}
	defer w.Close()
	return c.downloadByID(item.ID, w)
}

func (c Client) getItem(itemPath ...string) (*Item, error) {
	// GET /drive/root:/{item-path}
	u := c.endpoint
	{
		u.Path += "/root:"
		for _, p := range itemPath {
			u.Path += "/" + p
		}
	}
	var item Item
	if err := c.GetJSON(u.String(), &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c Client) getFileItem(itemPath ...string) (*Item, error) {
	item, err := c.getItem(itemPath...)
	if err != nil {
		return nil, err
	}
	if item.File == nil {
		return nil, fmt.Errorf("NOT a file")
	}
	return item, nil
}

// https://dev.onedrive.com/items/upload_put.htm
func (c Client) simpleUpload(localfile string, dirs ...string) (*Item, error) {
	lf, err := os.Open(localfile)
	if err != nil {
		return nil, err
	}
	name := path.Base(localfile)
	u := c.endpoint
	// PUT /drive/root:/{parent-path}/{filename}:/content
	u.Path += path.Join("/root:", strings.Join(dirs, "/"), name+":", "content")
	body := &bytes.Buffer{}
	io.Copy(body, lf)
	req, err := http.NewRequest("PUT", u.String(), body)
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, errors.New(res.Status)
	}
	var item Item
	if err := readJSON(res.Body, &item); err != nil {
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
func (c Client) downloadByID(id string, w io.Writer) error {
	// GET /drive/items/{item-id}/content
	u := c.endpoint
	u.Path += fmt.Sprintf("/items/%s/content", id)
	res, err := c.client.Get(u.String())
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}
	_, err = io.Copy(w, res.Body)
	return err
}

// DeleteByID implements https://dev.onedrive.com/items/delete.htm
func (c Client) DeleteByID(id string) error {
	u := c.endpoint
	u.Path += path.Join("/items", id)
	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		return errors.New(res.Status)
	}
	return nil
}

// List implements https://dev.onedrive.com/items/list.htm
func (c Client) List(dirs ...string) ([]Item, string, error) {
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
	var result ListItemResult
	if err := c.GetJSON(u.String(), &result); err != nil {
		return nil, "", err
	}
	return result.Value, result.NextLink, nil
}

// GetJSON gets a JSON object from rawurl
func (c Client) GetJSON(rawurl string, i interface{}) error {
	res, err := c.client.Get(rawurl)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}
	return readJSON(res.Body, i)
}

func readJSON(r io.Reader, i interface{}) error {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, i)
}
