// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"bytes"
	"github.com/globocom/tsuru/app"
	"github.com/globocom/tsuru/db"
	"labix.org/v2/mgo/bson"
	. "launchpad.net/gocheck"
)

func (s *S) TestLogWriter(c *C) {
	var b bytes.Buffer
	a := app.App{Name: "newApp"}
	err := db.Session.Apps().Insert(a)
	c.Assert(err, IsNil)
	defer db.Session.Apps().Remove(bson.M{"name": a.Name})
	writer := LogWriter{&a, &b}
	data := []byte("ble")
	_, err = writer.Write(data)
	c.Assert(err, IsNil)
	c.Assert(b.Bytes(), DeepEquals, data)
	instance := app.App{}
	err = db.Session.Apps().Find(bson.M{"name": a.Name}).One(&instance)
	logLen := len(instance.Logs)
	c.Assert(instance.Logs[logLen-1].Message, Equals, string(data))
}

func (s *S) TestLogWriterShouldReturnsTheDataSize(c *C) {
	var b bytes.Buffer
	a := app.App{Name: "newApp"}
	err := db.Session.Apps().Insert(a)
	c.Assert(err, IsNil)
	defer db.Session.Apps().Remove(bson.M{"name": a.Name})
	writer := LogWriter{&a, &b}
	data := []byte("ble")
	n, err := writer.Write(data)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(data))
}
