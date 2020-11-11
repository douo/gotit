package main

import (
	"log"
	"net/url"
	"os"
	"path"

	"github.com/douo/gotit"
	"github.com/urfave/cli/v2"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	taskFlag := []cli.Flag{
		&cli.Int64Flag{
			Name:  "min-split-size",
			Value: 1 * 1024 * 1024,
			Usage: "Mininmum size of reqeust content",
		},
		&cli.IntFlag{
			Name:  "max-conn",
			Value: 10,
			Usage: "Maximum connection for single task",
		},
		&cli.IntFlag{
			Name:  "buf-size",
			Value: 1 * 1024 * 1024,
			Usage: "Buffer for per connection",
		},
	}
	app := &cli.App{
		Name:  "gotit",
		Usage: "multiple connection download tool make by golang",
		Commands: []*cli.Command{
			{
				Name:    "download",
				Aliases: []string{"d"},
				Flags: append([]cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "`File` to save content",
					},
				}, taskFlag...),
				Usage:     "Download a resource from given `Url`",
				ArgsUsage: "Url",
				Action: func(c *cli.Context) error {
					u, err := url.ParseRequestURI(c.Args().First())
					if err != nil {
						return err
					}
					config := gotit.Config{
						MinSplitSize: c.Int64("min-split-size"),
						MaxConn:      c.Int("max-conn"),
						BufSize:      c.Int("buf-size"),
					}
					o := c.String("output")
					if o == "" {
						o = path.Base(u.Path)
					}
					task, err := gotit.NewTask(u.String(), o, config)
					if err != nil {
						return err
					}
					return task.Start()

				},
			},
			{
				Name:      "resume",
				Aliases:   []string{"r"},
				Usage:     "Resume a imcomplete download from imcomplete output `File`",
				Flags:     taskFlag,
				ArgsUsage: "File",
				Action: func(c *cli.Context) error {
					log.Printf("resume:%q\n", c.Args().First())
					return nil
				},
			},
			{
				Name:      "status",
				Aliases:   []string{"s"},
				Usage:     "print a download status from imcomplete file",
				ArgsUsage: "File",
				Action: func(c *cli.Context) error {
					log.Printf("download:%q\n", c.Args().First())
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
