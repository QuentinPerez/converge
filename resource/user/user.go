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

package user

import (
	"fmt"
	"os/user"

	"github.com/asteris-llc/converge/resource"
	"github.com/pkg/errors"
)

// State type for User
type State string

const (
	// StatePresent indicates the user should be present
	StatePresent State = "present"

	// StateAbsent indicates the user should be absent
	StateAbsent State = "absent"
)

// User manages user users
type User struct {
	Username    string
	NewUsername string
	UID         string
	GroupName   string
	GID         string
	Name        string
	HomeDir     string
	MoveDir     bool
	State       State
	system      SystemUtils
}

// AddUserOptions are the options specified in the configuration to be used
// when adding a user
type AddUserOptions struct {
	UID       string
	Group     string
	Comment   string
	Directory string
}

// ModUserOptions are the options specified in the configuration to be used
// when modifying a user
type ModUserOptions struct {
	UID       string
	Group     string
	Comment   string
	Directory string
	MoveDir   bool
}

// SystemUtils provides system utilities for user
type SystemUtils interface {
	AddUser(userName string, options *AddUserOptions) error
	DelUser(userName string) error
	ModUser(userName string, options *ModUserOptions) error
	Lookup(userName string) (*user.User, error)
	LookupID(userID string) (*user.User, error)
	LookupGroup(groupName string) (*user.Group, error)
	LookupGroupID(groupID string) (*user.Group, error)
}

// ErrUnsupported is used when a system is not supported
var ErrUnsupported = fmt.Errorf("user: not supported on this system")

// NewUser constructs and returns a new User
func NewUser(system SystemUtils) *User {
	return &User{
		system: system,
	}
}

// Check if a user user exists
func (u *User) Check(resource.Renderer) (resource.TaskStatus, error) {
	var (
		userByID      *user.User
		uidErr        error
		userByNewName *user.User
		newNameErr    error
	)

	// lookup the user by name and lookup the user by uid
	// the lookups return ErrUnsupported if the system is not supported
	// Lookup returns user.UnknownUserError if the user is not found
	// LookupID returns user.UnknownUserIdError if the uid is not found
	userByName, nameErr := u.system.Lookup(u.Username)
	if u.UID != "" {
		userByID, uidErr = u.system.LookupID(u.UID)
	}
	if u.NewUsername != "" {
		userByNewName, newNameErr = u.system.Lookup(u.NewUsername)
	}

	status := &resource.Status{}

	if nameErr == ErrUnsupported {
		status.Level = resource.StatusFatal
		return status, ErrUnsupported
	}

	switch u.State {
	case StatePresent:
		_, nameNotFound := nameErr.(user.UnknownUserError)

		switch {
		case u.UID != "" && userByName != nil && userByID != nil && (userByName.Name != userByID.Name || userByName.Uid != userByID.Uid):
			status.Level = resource.StatusCantChange
			status.Output = append(status.Output, fmt.Sprintf("user %s and uid %s belong to different users", u.Username, u.UID))
			return status, fmt.Errorf("cannot modify user %s with uid %s", u.Username, u.UID)
		case u.UID != "" && userByName != nil && userByID != nil && *userByName == *userByID:
			status.Output = append(status.Output, fmt.Sprintf("user %s with uid %s already exists", u.Username, u.UID))
		case nameNotFound:
			// Add User
			switch {
			case u.UID == "":
				switch {
				case u.GroupName != "":
					_, err := u.system.LookupGroup(u.GroupName)
					if err != nil {
						status.Level = resource.StatusCantChange
						status.Output = append(status.Output, fmt.Sprintf("group %s does not exist", u.GroupName))
						return status, fmt.Errorf("cannot add user %s", u.Username)
					}
				case u.GID != "":
					_, err := u.system.LookupGroupID(u.GID)
					if err != nil {
						status.Level = resource.StatusCantChange
						status.Output = append(status.Output, fmt.Sprintf("group gid %s does not exist", u.GID))
						return status, fmt.Errorf("cannot add user %s", u.Username)
					}
				}
				status.Level = resource.StatusWillChange
				status.Output = append(status.Output, "add user")
				status.AddDifference("user", string(StateAbsent), fmt.Sprintf("user %s", u.Username), "")
			case u.UID != "":
				_, uidNotFound := uidErr.(user.UnknownUserIdError)

				switch {
				case uidNotFound:
					switch {
					case u.GroupName != "":
						_, err := u.system.LookupGroup(u.GroupName)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group %s does not exist", u.GroupName))
							return status, fmt.Errorf("cannot add user %s with uid %s", u.Username, u.UID)
						}
					case u.GID != "":
						_, err := u.system.LookupGroupID(u.GID)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group gid %s does not exist", u.GID))
							return status, fmt.Errorf("cannot add user %s with uid %s", u.Username, u.UID)
						}
					}
					status.Level = resource.StatusWillChange
					status.Output = append(status.Output, "add user with uid")
					status.AddDifference("user", string(StateAbsent), fmt.Sprintf("user %s with uid %s", u.Username, u.UID), "")
				case userByID != nil:
					status.Level = resource.StatusCantChange
					status.Output = append(status.Output, fmt.Sprintf("user uid %s already exists", u.UID))
					return status, fmt.Errorf("cannot add user %s with uid %s", u.Username, u.UID)
				}
			}
		case userByName != nil:
			// Modify User
			switch {
			case u.NewUsername == "" && u.UID == "":
				if noOptionsSet(u) {
					status.Output = append(status.Output, fmt.Sprintf("no modifications indicated for user %s", u.Username))
				} else {
					switch {
					case u.GroupName != "":
						_, err := u.system.LookupGroup(u.GroupName)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group %s does not exist", u.GroupName))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					case u.GID != "":
						_, err := u.system.LookupGroupID(u.GID)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group gid %s does not exist", u.GID))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					}
					status.Level = resource.StatusWillChange
					status.Output = append(status.Output, "modify user")
				}
			case u.NewUsername != "" && u.UID == "":
				_, newNameNotFound := newNameErr.(user.UnknownUserError)

				switch {
				case userByNewName != nil:
					status.Level = resource.StatusCantChange
					status.Output = append(status.Output, fmt.Sprintf("user modify: user %s already exists", u.NewUsername))
					return status, errors.New("cannot modify user")
				case newNameNotFound:
					switch {
					case u.GroupName != "":
						_, err := u.system.LookupGroup(u.GroupName)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group %s does not exist", u.GroupName))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					case u.GID != "":
						_, err := u.system.LookupGroupID(u.GID)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group gid %s does not exist", u.GID))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					}
					status.Level = resource.StatusWillChange
					status.Output = append(status.Output, "modify user name")
					status.AddDifference("user", fmt.Sprintf("user %s", u.Username), fmt.Sprintf("user %s", u.NewUsername), "")
				}
			case u.NewUsername == "" && u.UID != "":
				_, uidNotFound := uidErr.(user.UnknownUserIdError)

				switch {
				case userByID != nil:
					status.Level = resource.StatusCantChange
					status.Output = append(status.Output, fmt.Sprintf("user modify: user uid %s already exists", u.UID))
					return status, errors.New("cannot modify user")
				case uidNotFound:
					switch {
					case u.GroupName != "":
						_, err := u.system.LookupGroup(u.GroupName)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group %s does not exist", u.GroupName))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					case u.GID != "":
						_, err := u.system.LookupGroupID(u.GID)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group gid %s does not exist", u.GID))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					}
					status.Level = resource.StatusWillChange
					status.Output = append(status.Output, "modify user uid")
					status.AddDifference("user", fmt.Sprintf("user %s with uid %s", u.Username, userByName.Uid), fmt.Sprintf("user %s with uid %s", u.Username, u.UID), "")
				}
			case u.NewUsername != "" && u.UID != "":
				_, newNameNotFound := newNameErr.(user.UnknownUserError)
				_, uidNotFound := uidErr.(user.UnknownUserIdError)

				switch {
				case userByNewName != nil:
					status.Level = resource.StatusCantChange
					status.Output = append(status.Output, fmt.Sprintf("user modify: user %s already exists", u.NewUsername))
					return status, errors.New("cannot modify user")
				case userByID != nil:
					status.Level = resource.StatusCantChange
					status.Output = append(status.Output, fmt.Sprintf("user modify: user uid %s already exists", u.UID))
					return status, errors.New("cannot modify user")
				case newNameNotFound && uidNotFound:
					switch {
					case u.GroupName != "":
						_, err := u.system.LookupGroup(u.GroupName)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group %s does not exist", u.GroupName))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					case u.GID != "":
						_, err := u.system.LookupGroupID(u.GID)
						if err != nil {
							status.Level = resource.StatusCantChange
							status.Output = append(status.Output, fmt.Sprintf("group gid %s does not exist", u.GID))
							return status, fmt.Errorf("cannot modify user %s", u.Username)
						}
					}
					status.Level = resource.StatusWillChange
					status.Output = append(status.Output, "modify user name and uid")
					status.AddDifference("user", fmt.Sprintf("user %s with uid %s", u.Username, userByName.Uid), fmt.Sprintf("user %s with uid %s", u.NewUsername, u.UID), "")
				}
			}
		}
	case StateAbsent:
		switch {
		case u.UID == "":
			_, nameNotFound := nameErr.(user.UnknownUserError)

			switch {
			case nameNotFound:
				status.Output = append(status.Output, fmt.Sprintf("user %s does not exist", u.Username))
			case userByName != nil:
				status.Level = resource.StatusWillChange
				status.AddDifference("user", fmt.Sprintf("user %s", u.Username), string(StateAbsent), "")
			}
		case u.UID != "":
			_, nameNotFound := nameErr.(user.UnknownUserError)
			_, uidNotFound := uidErr.(user.UnknownUserIdError)

			switch {
			case nameNotFound && uidNotFound:
				status.Output = append(status.Output, "user name and uid do not exist")
			case nameNotFound:
				status.Level = resource.StatusCantChange
				status.Output = append(status.Output, fmt.Sprintf("user %s does not exist", u.Username))
				return status, fmt.Errorf("cannot delete user %s with uid %s", u.Username, u.UID)
			case uidNotFound:
				status.Level = resource.StatusCantChange
				status.Output = append(status.Output, fmt.Sprintf("user uid %s does not exist", u.UID))
				return status, fmt.Errorf("cannot delete user %s with uid %s", u.Username, u.UID)
			case userByName != nil && userByID != nil && userByName.Name != userByID.Name || userByName.Uid != userByID.Uid:
				status.Level = resource.StatusCantChange
				status.Output = append(status.Output, fmt.Sprintf("user %s and uid %s belong to different users", u.Username, u.UID))
				return status, fmt.Errorf("cannot delete user %s with uid %s", u.Username, u.UID)
			case userByName != nil && userByID != nil && *userByName == *userByID:
				status.Level = resource.StatusWillChange
				status.AddDifference("user", fmt.Sprintf("user %s with uid %s", u.Username, u.UID), string(StateAbsent), "")
			}
		}
	default:
		status.Level = resource.StatusFatal
		return status, fmt.Errorf("user: unrecognized state %v", u.State)
	}

	return status, nil
}

// Apply changes for user
func (u *User) Apply() (resource.TaskStatus, error) {
	var (
		userByID *user.User
		uidErr   error
	)

	// lookup the user by name and lookup the user by uid
	// the lookups return ErrUnsupported if the system is not supported
	// Lookup returns user.UnknownUserError if the user is not found
	// LookupID returns user.UnknownUserIdError if the uid is not found
	userByName, nameErr := u.system.Lookup(u.Username)
	if u.UID != "" {
		userByID, uidErr = u.system.LookupID(u.UID)
	}

	status := &resource.Status{}

	if nameErr == ErrUnsupported {
		status.Level = resource.StatusFatal
		return status, ErrUnsupported
	}

	switch u.State {
	case StatePresent:
		switch {
		case u.UID == "":
			_, nameNotFound := nameErr.(user.UnknownUserError)

			switch {
			case nameNotFound:
				options := SetAddUserOptions(u)
				err := u.system.AddUser(u.Username, options)
				if err != nil {
					status.Level = resource.StatusFatal
					status.Output = append(status.Output, fmt.Sprintf("error adding user %s", u.Username))
					return status, errors.Wrap(err, "group add")
				}
				status.Output = append(status.Output, fmt.Sprintf("added user %s", u.Username))
			default:
				status.Level = resource.StatusCantChange
				return status, fmt.Errorf("will not attempt add: user %s", u.Username)
			}
		case u.UID != "":
			_, nameNotFound := nameErr.(user.UnknownUserError)
			_, uidNotFound := uidErr.(user.UnknownUserIdError)

			switch {
			case nameNotFound && uidNotFound:
				options := SetAddUserOptions(u)
				err := u.system.AddUser(u.Username, options)
				if err != nil {
					status.Level = resource.StatusFatal
					status.Output = append(status.Output, fmt.Sprintf("error adding user %s with uid %s", u.Username, u.UID))
					return status, errors.Wrap(err, "group add")
				}
				status.Output = append(status.Output, fmt.Sprintf("added user %s with uid %s", u.Username, u.UID))
			default:
				status.Level = resource.StatusCantChange
				return status, fmt.Errorf("will not attempt add: user %s with uid %s", u.Username, u.UID)
			}
		}
	case StateAbsent:
		switch {
		case u.UID == "":
			_, nameNotFound := nameErr.(user.UnknownUserError)

			switch {
			case !nameNotFound && userByName != nil:
				err := u.system.DelUser(u.Username)
				if err != nil {
					status.Level = resource.StatusFatal
					status.Output = append(status.Output, fmt.Sprintf("error deleting user %s", u.Username))
					return status, errors.Wrap(err, "group delete")
				}
				status.Output = append(status.Output, fmt.Sprintf("deleted user %s", u.Username))
			default:
				status.Level = resource.StatusCantChange
				return status, fmt.Errorf("will not attempt delete: user %s", u.Username)
			}
		case u.UID != "":
			_, nameNotFound := nameErr.(user.UnknownUserError)
			_, uidNotFound := uidErr.(user.UnknownUserIdError)

			switch {
			case !nameNotFound && !uidNotFound && userByName != nil && userByID != nil && *userByName == *userByID:
				err := u.system.DelUser(u.Username)
				if err != nil {
					status.Level = resource.StatusFatal
					status.Output = append(status.Output, fmt.Sprintf("error deleting user %s with uid %s", u.Username, u.UID))
					return status, errors.Wrap(err, "group delete")
				}
				status.Output = append(status.Output, fmt.Sprintf("deleted user %s with uid %s", u.Username, u.UID))
			default:
				status.Level = resource.StatusCantChange
				return status, fmt.Errorf("will not attempt delete: user %s with uid %s", u.Username, u.UID)
			}
		}
	default:
		status.Level = resource.StatusFatal
		return status, fmt.Errorf("user: unrecognized state %s", u.State)
	}

	return status, nil
}

// SetAddUserOptions returns a AddUserOptions struct with the options
// specified in the configuration for adding a user
func SetAddUserOptions(u *User) *AddUserOptions {
	options := new(AddUserOptions)

	if u.UID != "" {
		options.UID = u.UID
	}

	switch {
	case u.GroupName != "":
		options.Group = u.GroupName
	case u.GID != "":
		options.Group = u.GID
	}

	if u.Name != "" {
		options.Comment = u.Name
	}

	if u.HomeDir != "" {
		options.Directory = u.HomeDir
	}

	return options
}

// SetModUserOptions returns a ModUserOptions struct with the options
// specified in the configuration for modifying a user
func SetModUserOptions(u *User) *ModUserOptions {
	options := new(ModUserOptions)

	if u.UID != "" {
		options.UID = u.UID
	}

	switch {
	case u.GroupName != "":
		options.Group = u.GroupName
	case u.GID != "":
		options.Group = u.GID
	}

	if u.Name != "" {
		options.Comment = u.Name
	}

	if u.HomeDir != "" {
		options.Directory = u.HomeDir
		if u.MoveDir {
			options.MoveDir = true
		}
	}

	return options
}

func noOptionsSet(u *User) bool {
	switch {
	case u.UID != "":
	case u.GroupName != "":
	case u.GID != "":
	case u.Name != "":
	case u.HomeDir != "":
		return false
	}
	return true
}
