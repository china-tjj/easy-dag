// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import (
	"time"
)

type NodeResult struct {
	Status   int
	Err      error
	Begin    time.Time
	Cost     time.Duration // 节点执行耗时，
	Attempts uint
}
