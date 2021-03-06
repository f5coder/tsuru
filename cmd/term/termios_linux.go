// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package term

import (
	"syscall"
)

type Termios syscall.Termios

var (
	TCGETS = syscall.TCGETS
	TCSETS = syscall.TCSETS
)
