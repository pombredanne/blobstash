package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"

	"github.com/tsileo/blobstash/client"
	"github.com/tsileo/blobstash/scheduler"
	"github.com/tsileo/blobstash/fs"
)

func main() {
	ignoredFiles := []string{"*~", "*.py[cod]", "nohup.out", "*.log", "tmp_*"}
	app := cli.NewApp()
	app.Name = "blobstash"
	app.Usage = "BlobStash command-line tool"
	app.Version = "0.1.0"
	//  app.Action = func(c *cli.Context) {
	//    println("Hello friend!")
	//  }
	app.Commands = []cli.Command{
		{
			Name:      "put",
			Usage:     "put a file/directory",
			Action: func(c *cli.Context) {
				client, _ := client.NewClient(ignoredFiles)
				b, m, wr, err := client.Put(c.Args().First())
				fmt.Printf("b:%+v,m:%+v,wr:%+v,err:%v\n", b, m, wr, err)
			},
		},
		//{
		//	Name:      "ls",
		//	Usage:     "List backups",
		//	Action: func(c *cli.Context) {
		//		client, _ := client.NewClient(ignoredFiles)
		//		metas, _ := client.List()
		//		for _, m := range metas {
		//			fmt.Printf("%+v\n", m)
		//		}
		//	},
		//},
		{
			Name:      "restore",
			Usage:     "Restore a snapshot",
			Action: func(c *cli.Context) {
				client, _ := client.NewClient(ignoredFiles)
				args := c.Args()
				snap, meta, rr, err := client.Get(args[0], args[1])
				fmt.Printf("snap:%+v,meta:%+v,rr:%+v/err:%v", snap, meta, rr, err)
			},
		},
		{
			Name:  "mount",
			Usage: "Mount the read-only filesystem to the given path",
			Action: func(c *cli.Context) {
				stop := make(chan bool, 1)
				stopped := make(chan bool, 1)
				fs.Mount(c.Args().First(), stop, stopped)
			},
		},
		{
			Name:      "daemon",
			Usage:     "Snapshot daemon",
			Action: func(c *cli.Context) {
				client, _ := client.NewClient(ignoredFiles)
				d := scheduler.New(client)
				d.Run()
			},
		},
	}
	app.Run(os.Args)
}
