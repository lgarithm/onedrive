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
	case "auth":
		onedrive.Auth()
	case "refresh":
		if err := onedrive.RefreshAcceccToken(); err != nil {
			glog.Exit(err)
		}
	case "upload":
		if len(args) < 2 {
			usage()
			return
		}
		cli, err := onedrive.New()
		if err != nil {
			glog.Exit(err)
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
		cli, err := onedrive.New()
		if err != nil {
			glog.Exit(err)
		}
		is, nextLink, err := cli.List(args[1:]...)
		if err != nil {
			glog.Exit(err)
		}
		const format = "%-20s %-16s %-48s %s\n"
		fmt.Printf(format, "id", "size", "url", "name")
		for _, i := range is {
			fmt.Printf(format, i.ID, showSize(i.Size), i.URL, i.Name)
		}
		if nextLink != "" {
			fmt.Printf("%s\n", nextLink)
		}
	case "download":
		cli, err := onedrive.New()
		if err != nil {
			glog.Exit(err)
		}
		if err := cli.Download(args[1:]...); err != nil {
			glog.Exit(err)
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
