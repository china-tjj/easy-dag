// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import "time"

// nodeMetadata 记录Node的元信息，作用如下：
// 1.避免创建dag后节点信息被用户修改，造成不符合预期的结果
// 2.把依赖节点的指针换为下标，储存dag时便可以把map换为slice，减少内存占用，加快查询速度
type nodeMetadata[T any] struct {
	name         string
	processor    Processor[T]
	localTimeout time.Duration
	totalTimeout time.Duration
	depCnt       int32
	children     []int
	weakChildren []int
	maxAttempts  uint
	backoffFunc  BackoffFunc
	onSuccess    NodeHookFunc[T]
	onFailure    NodeHookFunc[T]
}

func newNodeMetadata[T any](node *Node[T]) *nodeMetadata[T] {
	metaData := &nodeMetadata[T]{
		name:         node.Name,
		processor:    node.Processor,
		localTimeout: node.LocalTimeout,
		totalTimeout: node.TotalTimeout,
		maxAttempts:  node.MaxAttempts,
		backoffFunc:  node.BackoffFunc,
		onSuccess:    node.OnSuccess,
		onFailure:    node.OnFailure,
	}
	if metaData.name == "" {
		metaData.name = "noname"
	}
	return metaData
}
