// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import (
	"time"
)

type Processor[T any] func(node IRuntimeNode, params T) error

type NodeHookFunc[T any] func(node IRuntimeNode, params T)

type Node[T any] struct {
	// Name 节点名称，仅在err里展示用，建议 Name 保持唯一性
	Name string
	// Processor 节点方法，返回 nil 表示成功，返回 err 表示失败。超时后将无视该函数的返回值，并视为返回 TimeoutErr
	Processor Processor[T]
	// LocalTimeout 本地超时时间，在节点开始执行时开始计时，小于或等于0时表示无超时时
	LocalTimeout time.Duration
	// TotalTimeout 全局超时时间，在图开始执行时开始计时，小于或等于0时表示无超时时间
	TotalTimeout time.Duration
	// Dependencies 强依赖，依赖节点若出现 err（超时也是一种 err），当前节点不会运行
	Dependencies []*Node[T]
	// WeakDependencies 弱依赖，依赖节点若失败或超时，当前节点继续运行
	WeakDependencies []*Node[T]
	// MaxAttempts 最大重试次数，小于1时被视为1
	MaxAttempts uint
	// BackoffFunc 退避策略，即重试之间等待的时间间隔
	BackoffFunc BackoffFunc
	// 节点运行成功的钩子函数
	OnSuccess NodeHookFunc[T]
	// 节点运行失败的钩子函数
	OnFailure NodeHookFunc[T]
}

func (node *Node[T]) AddDependency(deps ...*Node[T]) {
	node.Dependencies = append(node.Dependencies, deps...)
}

func (node *Node[T]) AddWeakDependency(weekDeps ...*Node[T]) {
	node.WeakDependencies = append(node.WeakDependencies, weekDeps...)
}
