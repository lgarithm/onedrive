package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/lgarithm/onedrive/onedrive"
)

var (
	clientID     = flag.String("client_id", "", "")
	clientSecret = flag.String("client_secret", "", "")

	all        = flag.Bool("all", false, "list all")
	remotePath = flag.String("path", "upload", "remote folder")
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
		return
	}
	switch args[0] {
	case "config":
		onedrive.CreateConfig(*clientID, *clientSecret)
		return
	case "auth":
		onedrive.Auth()
		return
	case "refresh":
		if err := onedrive.RefreshAcceccToken(); err != nil {
			glog.Exit(err)
		}
		return
	}
	cli, err := onedrive.New()
	if err != nil {
		glog.Exit(err)
	}
	switch args[0] {
	case "upload":
		if len(args) < 2 {
			usage()
			return
		}
		dirs := strings.Split(*remotePath, "/")
		for _, file := range args[1:] {
			glog.Infof("Uploading %q to %s", file, *remotePath)
			res, err := cli.Upload(file, dirs...)
			if err != nil {
				glog.Exit(err)
			}
			fmt.Println(res)
		}
	case "ls":
		const format = "%-20s %-16s %-48s %s\n"
		fmt.Printf(format, "ID", "Size", "URL", "Name")
		showItems := func(is []onedrive.Item) {
			for _, i := range is {
				fmt.Printf(format, i.ID, showSize(i.Size), i.URL, i.Name)
			}
		}
		is, nextLink, err := cli.List(args[1:]...)
		if err != nil {
			glog.Exit(err)
		}
		showItems(is)
		if nextLink != "" {
			if !*all {
				fmt.Printf("next link: %s\n", nextLink)
			} else {
				for {
					var result onedrive.ListItemResult
					if err := cli.GetJSON(nextLink, &result); err != nil {
						glog.Exit(err)
					}
					showItems(result.Value)
					if result.NextLink == "" {
						break
					}
					nextLink = result.NextLink
				}
			}
		}
	case "download":
		if err := cli.Download(args[1:]...); err != nil {
			glog.Exit(err)
		}
	case "del":
		for _, id := range args[1:] {
			if err := cli.DeleteByID(id); err != nil {
				glog.Warning(err)
			}
		}
	default:
		usage()
	}
}

func usage() {
	flag.Usage()
}

func showSize(size int64) string {
	return strconv.FormatInt(size, 10)
}
