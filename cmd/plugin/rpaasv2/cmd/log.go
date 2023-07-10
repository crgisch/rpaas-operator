// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/urfave/cli/v2"

	rpaasclient "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
)

func NewCmdLogs() *cli.Command {
	return &cli.Command{
		Name:    "logs",
		Usage:   "Shows the log entries from instance pods",
		Aliases: []string{"log"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "service",
				Aliases: []string{"tsuru-service", "s"},
				Usage:   "the Tsuru service name",
			},
			&cli.StringFlag{
				Name:    "instance",
				Aliases: []string{"tsuru-service-instance", "i"},
				Usage:   "the reverse proxy instance name",
			},
			&cli.StringFlag{
				Name:    "pod",
				Aliases: []string{"p"},
				Usage:   "specific pod to log from (default: all pods from instance)",
			},
			&cli.PathFlag{
				Name:    "container",
				Aliases: []string{"c"},
				Usage:   "specific container to log from (default: all containers from pods)",
			},
			&cli.IntFlag{
				Name:    "lines",
				Aliases: []string{"l"},
				Usage:   "number of earlier log lines to show",
			},
			&cli.DurationFlag{
				Name:  "since",
				Usage: "only return logs newer than a relative duration like 5s, 2m, or 3h",
			},
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "specify if the logs should be streamed",
			},
			&cli.BoolFlag{
				Name:    "without-color",
				Aliases: []string{"no-color"},
				Usage:   "defines whether or not to display colorful output.",
			},
		},
		Before: setupClient,
		Action: runLogRpaas,
	}
}

func runLogRpaas(c *cli.Context) error {
	client, err := getClient(c)
	if err != nil {
		return err
	}

	return client.Log(c.Context, rpaasclient.LogArgs{
		Out:       c.App.Writer,
		Instance:  c.String("instance"),
		Lines:     c.Int("lines"),
		Since:     c.Duration("since"),
		Follow:    c.Bool("follow"),
		Pod:       c.String("pod"),
		Container: c.String("container"),
		Color:     !c.Bool("without-color"),
	})
}
