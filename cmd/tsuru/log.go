// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/globocom/tsuru/cmd"
	"io/ioutil"
	"net/http"
	"time"
)

type AppLog struct {
	GuessingCommand
}

func (c *AppLog) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "log",
		Usage: "log [--app appname]",
		Desc: `show logs for an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
}

type log struct {
	Date    time.Time
	Message string
}

func (c *AppLog) Run(context *cmd.Context, client cmd.Doer) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url := cmd.GetUrl(fmt.Sprintf("/apps/%s/log", appName))
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	logs := []log{}
	err = json.Unmarshal(result, &logs)
	if err != nil {
		return err
	}
	for _, l := range logs {
		context.Stdout.Write([]byte(l.Date.String() + " - " + l.Message + "\n"))
	}
	return err
}