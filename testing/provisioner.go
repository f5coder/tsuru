// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testing

import (
	"errors"
	"fmt"
	"github.com/globocom/tsuru/provision"
	"io"
	"strconv"
	"sync"
	"time"
)

func init() {
	provision.Register("fake", &FakeProvisioner{})
}

// Fake implementation for provision.Unit.
type FakeUnit struct {
	name    string
	machine int
	status  provision.Status
	actions []string
}

func (u *FakeUnit) GetName() string {
	u.actions = append(u.actions, "getname")
	return u.name
}

func (u *FakeUnit) GetMachine() int {
	u.actions = append(u.actions, "getmachine")
	return u.machine
}

func (u *FakeUnit) GetStatus() provision.Status {
	return u.status
}

// Fake implementation for provision.App.
type FakeApp struct {
	name      string
	framework string
	units     []provision.AppUnit
	logs      []string
	actions   []string
}

func NewFakeApp(name, framework string, units int) *FakeApp {
	app := FakeApp{
		name:      name,
		framework: framework,
		units:     make([]provision.AppUnit, units),
	}
	for i := 0; i < units; i++ {
		app.units[i] = &FakeUnit{name: name, machine: i + 1}
	}
	return &app
}

func (a *FakeApp) Log(message, source string) error {
	a.logs = append(a.logs, source+message)
	a.actions = append(a.actions, "log "+source+" - "+message)
	return nil
}

func (a *FakeApp) GetName() string {
	a.actions = append(a.actions, "getname")
	return a.name
}

func (a *FakeApp) GetFramework() string {
	a.actions = append(a.actions, "getframework")
	return a.framework
}

func (a *FakeApp) ProvisionUnits() []provision.AppUnit {
	a.actions = append(a.actions, "getunits")
	return a.units
}

type Cmd struct {
	Cmd  string
	Args []string
	App  provision.App
}

type failure struct {
	method string
	err    error
}

// Fake implementation for provision.Provisioner.
type FakeProvisioner struct {
	apps     []provision.App
	units    map[string][]provision.Unit
	cmds     []Cmd
	outputs  chan []byte
	failures chan failure
	cmdMut   sync.Mutex
	unitMut  sync.Mutex
}

func NewFakeProvisioner() *FakeProvisioner {
	p := FakeProvisioner{}
	p.outputs = make(chan []byte, 8)
	p.failures = make(chan failure, 8)
	p.units = make(map[string][]provision.Unit)
	return &p
}

func (p *FakeProvisioner) getError(method string) error {
	select {
	case fail := <-p.failures:
		if fail.method == method {
			return fail.err
		}
		p.failures <- fail
	case <-time.After(1e6):
	}
	return nil
}

// GetCmds returns a list of commands executed in an app. If you don't specify
// the command (""), it will return all commands executed in the given app.
func (p *FakeProvisioner) GetCmds(cmd string, app provision.App) []Cmd {
	var cmds []Cmd
	p.cmdMut.Lock()
	for _, c := range p.cmds {
		if (cmd == "" || c.Cmd == cmd) && app.GetName() == c.App.GetName() {
			cmds = append(cmds, c)
		}
	}
	p.cmdMut.Unlock()
	return cmds
}

func (p *FakeProvisioner) FindApp(app provision.App) int {
	for i, a := range p.apps {
		if a.GetName() == app.GetName() {
			return i
		}
	}
	return -1
}

func (p *FakeProvisioner) GetUnits(app provision.App) []provision.Unit {
	p.unitMut.Lock()
	defer p.unitMut.Unlock()
	return p.units[app.GetName()]
}

func (p *FakeProvisioner) PrepareOutput(b []byte) {
	p.outputs <- b
}

func (p *FakeProvisioner) PrepareFailure(method string, err error) {
	p.failures <- failure{method, err}
}

func (p *FakeProvisioner) Reset() {
	p.unitMut.Lock()
	p.units = make(map[string][]provision.Unit)
	p.unitMut.Unlock()

	p.cmdMut.Lock()
	p.cmds = nil
	p.cmdMut.Unlock()

	for {
		select {
		case <-p.outputs:
		case <-p.failures:
		default:
			return
		}
	}
}

func (p *FakeProvisioner) Provision(app provision.App) error {
	if err := p.getError("Provision"); err != nil {
		return err
	}
	index := p.FindApp(app)
	if index > -1 {
		return &provision.Error{Reason: "App already provisioned."}
	}
	p.apps = append(p.apps, app)
	p.unitMut.Lock()
	p.units[app.GetName()] = nil
	p.unitMut.Unlock()
	return nil
}

func (p *FakeProvisioner) Destroy(app provision.App) error {
	if err := p.getError("Destroy"); err != nil {
		return err
	}
	index := p.FindApp(app)
	if index == -1 {
		return &provision.Error{Reason: "App is not provisioned."}
	}
	copy(p.apps[index:], p.apps[index+1:])
	p.apps = p.apps[:len(p.apps)-1]
	p.unitMut.Lock()
	delete(p.units, app.GetName())
	p.unitMut.Unlock()
	return nil
}

func (p *FakeProvisioner) AddUnits(app provision.App, n uint) ([]provision.Unit, error) {
	if err := p.getError("AddUnits"); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("Cannot add 0 units.")
	}
	index := p.FindApp(app)
	if index < 0 {
		return nil, errors.New("App is not provisioned.")
	}
	name := app.GetName()
	framework := app.GetFramework()
	p.unitMut.Lock()
	defer p.unitMut.Unlock()
	length := uint(len(p.units[name]))
	for i := uint(0); i < n; i++ {
		unit := provision.Unit{
			Name:    fmt.Sprintf("%s/%d", name, length+i),
			AppName: name,
			Type:    framework,
			Status:  provision.StatusStarted,
			Ip:      fmt.Sprintf("10.10.10.%d", length+i),
			Machine: int(length + i),
		}
		p.units[name] = append(p.units[name], unit)
	}
	return p.units[name][length:], nil
}

func (p *FakeProvisioner) RemoveUnit(app provision.App, name string) error {
	if err := p.getError("RemoveUnit"); err != nil {
		return err
	}
	index := -1
	appName := app.GetName()
	if index := p.FindApp(app); index < 0 {
		return errors.New("App is not provisioned.")
	}
	p.unitMut.Lock()
	defer p.unitMut.Unlock()
	for i, unit := range p.units[appName] {
		if unit.Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Unit not found.")
	}
	copy(p.units[appName][index:], p.units[appName][index+1:])
	p.units[appName] = p.units[appName][:len(p.units[appName])-1]
	return nil
}

func (p *FakeProvisioner) RemoveUnits(app provision.App, n uint) error {
	if err := p.getError("RemoveUnits"); err != nil {
		return err
	}
	if index := p.FindApp(app); index < 0 {
		return errors.New("App is not provisioned.")
	}
	name := app.GetName()
	p.unitMut.Lock()
	defer p.unitMut.Unlock()
	length := uint(len(p.units[name]))
	if n >= length {
		return errors.New("Too many units.")
	}
	p.units[name] = p.units[name][n:]
	return nil
}

func (p *FakeProvisioner) ExecuteCommand(stdout, stderr io.Writer, app provision.App, cmd string, args ...string) error {
	var (
		output []byte
		err    error
	)
	command := Cmd{
		Cmd:  cmd,
		Args: args,
		App:  app,
	}
	p.cmdMut.Lock()
	p.cmds = append(p.cmds, command)
	p.cmdMut.Unlock()
	select {
	case output = <-p.outputs:
		select {
		case fail := <-p.failures:
			if fail.method == "ExecuteCommand" {
				stderr.Write(output)
				return fail.err
			} else {
				p.failures <- fail
			}
		case <-time.After(1e6):
			stdout.Write(output)
		}
	case fail := <-p.failures:
		if fail.method == "ExecuteCommand" {
			err = fail.err
			select {
			case output = <-p.outputs:
				stderr.Write(output)
			case <-time.After(1e6):
			}
		} else {
			p.failures <- fail
		}
	case <-time.After(2e9):
		return errors.New("FakeProvisioner timed out waiting for output.")
	}
	return err
}

func (p *FakeProvisioner) CollectStatus() ([]provision.Unit, error) {
	if err := p.getError("CollectStatus"); err != nil {
		return nil, err
	}
	units := make([]provision.Unit, len(p.apps))
	for i, app := range p.apps {
		unit := provision.Unit{
			Name:    app.GetName() + "/0",
			AppName: app.GetName(),
			Type:    app.GetFramework(),
			Status:  "started",
			Ip:      "10.10.10." + strconv.Itoa(i+1),
			Machine: i + 1,
		}
		units[i] = unit
	}
	return units, nil
}
