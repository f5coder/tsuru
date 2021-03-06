// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provision

import (
	. "github.com/globocom/tsuru/api/service"
	. "launchpad.net/gocheck"
)

func (s *S) TestgetServiceOrError(c *C) {
	srv := Service{Name: "foo", OwnerTeams: []string{s.team.Name}}
	err := srv.Create()
	c.Assert(err, IsNil)
	defer srv.Delete()
	rSrv, err := getServiceOrError("foo", s.user)
	c.Assert(err, IsNil)
	c.Assert(rSrv.Name, Equals, srv.Name)
}

func (s *S) TestServicesAndInstancesByOwnerTeams(c *C) {
	srvc := Service{Name: "mysql", OwnerTeams: []string{s.team.Name}}
	err := srvc.Create()
	c.Assert(err, IsNil)
	defer srvc.Delete()
	srvc2 := Service{Name: "mongodb"}
	err = srvc2.Create()
	c.Assert(err, IsNil)
	defer srvc2.Delete()
	sInstance := ServiceInstance{Name: "foo", ServiceName: "mysql"}
	err = sInstance.Create()
	c.Assert(err, IsNil)
	defer sInstance.Delete()
	sInstance2 := ServiceInstance{Name: "bar", ServiceName: "mongodb"}
	err = sInstance2.Create()
	defer sInstance2.Delete()
	results := servicesAndInstancesByOwner(s.user)
	expected := []ServiceModel{
		{Service: "mysql", Instances: []string{"foo"}},
	}
	c.Assert(results, DeepEquals, expected)
}
