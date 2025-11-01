// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import (
	"sync"
)

type IPool interface {
	Submit(func())
}

type Pool struct {
	mu         sync.Mutex
	head       *task
	tail       *task
	len        int
	maxWorkers int
	workers    int
}

type task struct {
	f    func()
	next *task
}

func NewPool(maxWorkers int) *Pool {
	t := &task{}
	t.next = t
	return &Pool{
		maxWorkers: maxWorkers,
		head:       t,
		tail:       t,
	}
}

func (p *Pool) Submit(f func()) {
	if f == nil {
		return
	}
	p.mu.Lock()
	if p.workers < p.maxWorkers {
		p.workers++
		p.mu.Unlock()
		go p.work(f)
		return
	}
	newTail := &task{f: f}
	p.tail.next = newTail
	p.tail = newTail
	p.len++
	p.mu.Unlock()
}

func (p *Pool) work(f func()) {
	for {
		f()
		p.mu.Lock()
		if p.len > 0 {
			f = p.head.f
			p.head = p.head.next
			p.len--
			p.mu.Unlock()
		} else {
			p.workers--
			p.mu.Unlock()
			return
		}
	}
}
