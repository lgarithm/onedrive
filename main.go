package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/lgarithm/onedrive/onedrive"
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
		for _, file := range args[1:] {
			glog.Infof("Uploading %q", file)
			res, err := cli.Upload(file)
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
		is, err := cli.List(args[1:]...)
		if err != nil {
			glog.Exit(err)
		}
		for _, i := range is {
			fmt.Printf("%s\n", i)
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