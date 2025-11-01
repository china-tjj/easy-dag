// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

type strErr string

func (e strErr) Error() string {
	return string(e)
}

const TimeoutErr = strErr("timeout")
