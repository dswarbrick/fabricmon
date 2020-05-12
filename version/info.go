// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

package version

import (
	"fmt"
	"runtime"
)

// Build information, set via "-X" ldflags
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
)

func BuildContext() string {
	return fmt.Sprintf("(go=%s, user=%s, date=%s)", runtime.Version(), BuildUser, BuildDate)
}

func Info() string {
	return fmt.Sprintf("(version=%s, branch=%s, revision=%s)", Version, Branch, Revision)
}
