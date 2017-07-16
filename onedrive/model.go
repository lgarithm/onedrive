package onedrive

import (
	"encoding/json"
)

// Item is https://dev.onedrive.com/resources/item.htm
type Item struct {
	ID string `json:"id"`

	File *struct {
		Hashes   map[string]string `json:"hashes"`
		MimeType string            `json:"mimeType"`
	} `json:"file"`

	Folder *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder"`

	Name string `json:"name"`
	URL  string `json:"webUrl"`
	Size int64  `json:"size"`
}

func (i Item) String() string {
	bs, _ := json.MarshalIndent(i, "", "  ")
	return string(bs)
}

type ListItemResult struct {
	Value    []Item `json:"value"`
	NextLink string `json:"@odata.nextLink"`
}

type UploadSession struct {
	NextExpectedRanges []string `json:"nextExpectedRanges"`
	UploadURL          string   `json:"uploadUrl"`
}
