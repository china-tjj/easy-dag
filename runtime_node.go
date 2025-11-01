// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// IRuntimeNode 节点运行时对外暴露的交互接口
type IRuntimeNode interface {
	// GetName 获取节点名称
	GetName() string
	// DoIfRunning 正在运行时（即未超时时）才执行，返回是否成功执行；若成功开始执行，在执行完成之前不会触发超时（超时推迟到执行完成后发生）。
	// 最佳实践：节点仅在未超时时往数据总线写入数据，主流程在图执行结束后再操作数据总线，主流程无需加锁。
	// 该方法锁的粒度较小，仅与超时处理互斥，并发访问数据总线需自行加锁。
	DoIfRunning(fn func()) bool
	// GetDDL 获取节点的最终截止时间（ddl）、是否获取成功
	GetDDL() (time.Time, bool)
	// GetCost 获取节点执行耗时，包括多次重试的总时间、重试的退避时间、超时后继续执行的时间
	GetCost() time.Duration
	// GetAttempts 获取节点运行次数
	GetAttempts() uint
}

// runtimeNode dag每次运行时创建的节点，是有状态的
type runtimeNode[T any] struct {
	*nodeMetadata[T]
	ctx          *dagCtx
	doneDepCnt   atomic.Int32
	children     []*runtimeNode[T]
	weakChildren []*runtimeNode[T]
	status       atomic.Int32
	done         chan struct{}
	err          error
	// mu 与超时控制互斥，故仅在超时时加写锁（排他锁），其余情况加读锁（共享锁）
	mu       sync.RWMutex
	begin    time.Time
	ddl      time.Time
	cost     atomic.Int64
	attempts uint
}

func newRuntimeNode[T any](metaData *nodeMetadata[T], ctx *dagCtx) *runtimeNode[T] {
	return &runtimeNode[T]{
		nodeMetadata: metaData,
		ctx:          ctx,
		children:     make([]*runtimeNode[T], 0, len(metaData.children)),
		weakChildren: make([]*runtimeNode[T], 0, len(metaData.weakChildren)),
		done:         make(chan struct{}),
	}
}

func (node *runtimeNode[T]) GetName() string {
	return node.name
}

func (node *runtimeNode[T]) DoIfRunning(fn func()) bool {
	node.mu.RLock()
	defer node.mu.RUnlock()
	if node.status.Load() != Running {
		return false
	}
	fn()
	return true
}

func (node *runtimeNode[T]) GetDDL() (time.Time, bool) {
	if node.localTimeout <= 0 && node.totalTimeout <= 0 {
		return time.Time{}, false
	}
	return node.ddl, true
}

func (node *runtimeNode[T]) GetCost() time.Duration {
	node.mu.RLock()
	defer node.mu.RUnlock()
	select {
	case <-node.done:
		return time.Duration(node.cost.Load())
	default:
		return time.Since(node.begin)
	}
}

func (node *runtimeNode[T]) GetAttempts() uint {
	return node.attempts
}

func (node *runtimeNode[T]) start(params T) {
	if !node.status.CompareAndSwap(Waiting, Running) {
		return
	}
	node.ctx.wg.Add(1)
	if node.ctx.pool == nil {
		go node.run(params)
	} else {
		node.ctx.pool.Submit(func() {
			node.run(params)
		})
	}
}

func (node *runtimeNode[T]) run(params T) {
	defer node.ctx.wg.Done()
	if node.totalTimeout > 0 && time.Now().After(node.ctx.begin.Add(node.totalTimeout)) {
		node.fail(params, TimeoutErr)
	} else if node.processor == nil {
		node.success(params)
	} else if node.localTimeout <= 0 && node.totalTimeout <= 0 {
		node.processWithoutTimeout(params)
	} else {
		node.processWithTimeout(params)
	}
	if node.status.Load() == Succeeded {
		for _, child := range node.children {
			child.onDepDone(params)
		}
	}
	for _, child := range node.weakChildren {
		child.onDepDone(params)
	}
}

func (node *runtimeNode[T]) process(params T) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("recover panic over node %s: %v", node.name, e)
		}
	}()
	return node.processor(node, params)
}

func (node *runtimeNode[T]) processWithRetry(params T) {
	var err error
	defer func() {
		node.cost.Store(int64(time.Since(node.begin)))
		close(node.done)
		if err == nil {
			node.success(params)
		} else {
			node.fail(params, err)
		}
	}()
	maxAttempts := maxUint(1, node.maxAttempts)
	for node.attempts < maxAttempts {
		ok := node.DoIfRunning(func() {
			node.attempts++
		})
		// 避免超时后继续重跑
		if !ok {
			return
		}
		err = node.process(params)
		if err == nil {
			return
		}
		if node.attempts != maxAttempts && node.backoffFunc != nil {
			// 避免超时后无效等待
			if node.status.Load() != Running {
				return
			}
			time.Sleep(node.backoffFunc(node.attempts))
		}
	}
	return
}

func (node *runtimeNode[T]) processWithoutTimeout(params T) {
	node.begin = time.Now()
	node.processWithRetry(params)
}

func (node *runtimeNode[T]) processWithTimeout(params T) {
	started := make(chan struct{})
	process := func() {
		node.begin = time.Now()
		timeout := time.Duration(math.MaxInt64)
		if node.localTimeout > 0 {
			timeout = minDuration(timeout, node.localTimeout)
		}
		if node.totalTimeout > 0 {
			timeout = minDuration(timeout, node.ctx.begin.Add(node.totalTimeout).Sub(node.begin))
		}
		node.ddl = node.begin.Add(timeout)
		close(started)
		node.processWithRetry(params)
	}
	if node.ctx.pool == nil {
		go process()
	} else {
		node.ctx.pool.Submit(process)
	}
	<-started
	select {
	case <-node.done:
		break
	case <-time.After(time.Until(node.ddl)):
		// 在超时时，可能processor正在调用DoIfRunning，需要加锁，其余情况无并发冲突，无需加锁
		node.mu.Lock()
		node.fail(params, TimeoutErr)
		node.mu.Unlock()
	}
}

func (node *runtimeNode[T]) onDepDone(params T) {
	if node.doneDepCnt.Add(1) == node.depCnt {
		node.start(params)
	}
}

func (node *runtimeNode[T]) success(params T) {
	if !node.status.CompareAndSwap(Running, Succeeded) {
		return
	}
	if node.onSuccess != nil {
		node.onSuccess(node, params)
	}
}

func (node *runtimeNode[T]) fail(params T, err error) {
	if !node.status.CompareAndSwap(Running, Failed) {
		return
	}
	node.err = err
	if node.onFailure != nil {
		node.onFailure(node, params)
	}
}

func (node *runtimeNode[T]) getResult() *NodeResult {
	return &NodeResult{
		Status:   int(node.status.Load()),
		Err:      node.err,
		Begin:    node.begin,
		Cost:     node.GetCost(),
		Attempts: node.attempts,
	}
}
