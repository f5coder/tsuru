// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"fmt"
	"github.com/globocom/config"
	gandalf "github.com/globocom/go-gandalfclient"
	"github.com/globocom/tsuru/api/bind"
	"github.com/globocom/tsuru/db"
	"github.com/globocom/tsuru/repository"
	"labix.org/v2/mgo/bson"
	"strconv"
)

// insertApp is an implementation for the action interface.
type insertApp struct{}

// insertApp forward stores the app with "pending" as your state.
func (a *insertApp) forward(app *App) error {
	app.State = "pending"
	return db.Session.Apps().Insert(app)
}

// insertApp backward removes the app from the database.
func (a *insertApp) backward(app *App) {
	db.Session.Apps().Remove(bson.M{"name": app.Name})
}

func (a *insertApp) rollbackItself() bool {
	return false
}

// createBucketIam is an implementation for the action interface.
type createBucketIam struct{}

// createBucketIam forward creates a bucket and exports
// the related info as environs in the app machine.
func (a *createBucketIam) forward(app *App) error {
	env, err := createBucket(app)
	if err != nil {
		return err
	}
	host, _ := config.GetString("host")
	envVars := []bind.EnvVar{
		{Name: "APPNAME", Value: app.Name},
		{Name: "TSURU_HOST", Value: host},
	}
	variables := map[string]string{
		"ENDPOINT":           env.endpoint,
		"LOCATIONCONSTRAINT": strconv.FormatBool(env.locationConstraint),
		"ACCESS_KEY_ID":      env.AccessKey,
		"SECRET_KEY":         env.SecretKey,
		"BUCKET":             env.bucket,
	}
	for name, value := range variables {
		envVars = append(envVars, bind.EnvVar{
			Name:         fmt.Sprintf("TSURU_S3_%s", name),
			Value:        value,
			InstanceName: s3InstanceName,
		})
	}
	app.SetEnvsToApp(envVars, false, true)
	return nil
}

// createBucketIam backward destroys the app bucket.
func (a *createBucketIam) backward(app *App) {
	destroyBucket(app)
}

func (a *createBucketIam) rollbackItself() bool {
	return true
}

// provisionApp is an implementation for the action interface.
type provisionApp struct{}

// provision forward provisions the app.
func (a *provisionApp) forward(app *App) error {
	return Provisioner.Provision(app)
}

// provision backward does nothing.
func (a *provisionApp) backward(app *App) {}

func (a *provisionApp) rollbackItself() bool {
	return false
}

// createRepository is an implementation for the action interface.
type createRepository struct{}

// createRepository forward creates a git repository using the
// gandalf client.
func (a *createRepository) forward(app *App) error {
	gUrl := repository.GitServerUri()
	var users []string
	for _, t := range app.teams() {
		users = append(users, t.Users...)
	}
	c := gandalf.Client{Endpoint: gUrl}
	_, err := c.NewRepository(app.Name, users, false)
	return err
}

// createRepository backward remove the git repository
// using the gandalf client.
func (a *createRepository) backward(app *App) {
	gUrl := repository.GitServerUri()
	c := gandalf.Client{Endpoint: gUrl}
	c.RemoveRepository(app.Name)
}

func (a *createRepository) rollbackItself() bool {
	return false
}
