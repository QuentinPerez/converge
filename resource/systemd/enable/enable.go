// Copyright © 2016 Asteris, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package enable

import (
	"time"

	"github.com/asteris-llc/converge/helpers"
	"github.com/asteris-llc/converge/resource/systemd/common"
	"github.com/coreos/go-systemd/dbus"
)

// Content renders a content to disk
type Enable struct {
	//TODO when arrays are implemented, change this to array
	Unit    string
	Runtime bool //Unit enabled for runtime only (true, run), or perstently (false, /etc)
	Force   bool //whether symlinks pointing to other units shall be replaced if necessary.
	Timeout time.Duration
}

// Check if the content needs to be rendered
func (e *Enable) Check() (status string, willChange bool, err error) {
	conn, err := dbus.New()
	if err != nil {
		return err.Error(), false, err
	}
	defer conn.Close()
	common.WaitToLoad(conn, e.Unit, e.Timeout)
	status, willChange, err = common.CheckUnitIsActive(conn, e.Unit)
	//Check runtime
	if e.Runtime {
		s, w, er := common.CheckUnitHasValidUFSRuntimes(conn, e.Unit)
		status, willChange, err = helpers.SquashCheck(status, willChange, err, s, w, er)
	} else {
		s, w, er := common.CheckUnitHasValidUFS(conn, e.Unit)
		status, willChange, err = helpers.SquashCheck(status, willChange, err, s, w, er)
	}
	return status, willChange, err
}

// Apply writes the content to disk
func (e *Enable) Apply() (err error) {
	conn, err := dbus.New()
	if err != nil {
		return err
	}
	defer conn.Close()
	_, _, err = conn.EnableUnitFiles([]string{e.Unit}, e.Runtime, e.Force)
	return err
}
