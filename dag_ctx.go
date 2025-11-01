// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import (
	"sync"
	"time"
)

type dagCtx struct {
	wg    sync.WaitGroup
	pool  IPool
	begin time.Time
}

func newDagCtx(pool IPool) *dagCtx {
	return &dagCtx{
		begin: time.Now(),
		pool:  pool,
	}
}
