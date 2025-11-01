// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import (
	"errors"
	"slices"
	"strings"
)

type dagBuilder[T any] struct {
	nodes     []*Node[T]         // 用户输入的节点
	metaNodes []*nodeMetadata[T] // 所有节点的元数据
	index     map[*Node[T]]int   // 用户节点 -> 元数据下标
	visited   []bool             // 环检测：是否已访问
	next      []int              // 环检测：DFS实时搜索路径
}

func newDagBuilder[T any](nodes []*Node[T]) *dagBuilder[T] {
	return &dagBuilder[T]{
		nodes:     nodes,
		index:     make(map[*Node[T]]int, len(nodes)),
		metaNodes: make([]*nodeMetadata[T], 0, len(nodes)),
	}
}

func (b *dagBuilder[T]) build() (*DAG[T], error) {
	for _, node := range b.nodes {
		if node == nil {
			continue
		}
		b.add(node)
	}
	b.visited = make([]bool, len(b.metaNodes))
	b.next = make([]int, len(b.metaNodes))
	for idx := range b.next {
		b.next[idx] = -1
	}
	for idx := range b.metaNodes {
		if err := b.detectCycle(idx); err != nil {
			return nil, err
		}
	}
	dag := &DAG[T]{metaNodes: b.metaNodes}
	for idx, node := range b.metaNodes {
		if node.depCnt == 0 {
			dag.rootNodes = append(dag.rootNodes, idx)
		}
	}
	return dag, nil
}

func (b *dagBuilder[T]) add(node *Node[T]) int {
	if idx, exist := b.index[node]; exist {
		return idx
	}
	idx := len(b.metaNodes)
	b.index[node] = idx
	medaData := newNodeMetadata(node)
	b.metaNodes = append(b.metaNodes, medaData)
	for _, dep := range node.Dependencies {
		if dep == nil {
			continue
		}
		depIdx := b.add(dep)
		b.metaNodes[depIdx].children = append(b.metaNodes[depIdx].children, idx)
		medaData.depCnt++
	}
	for _, weakDep := range node.WeakDependencies {
		if weakDep == nil {
			continue
		}
		weakDepIdx := b.add(weakDep)
		b.metaNodes[weakDepIdx].weakChildren = append(b.metaNodes[weakDepIdx].weakChildren, idx)
		medaData.depCnt++
	}
	return idx
}

func (b *dagBuilder[T]) detectCycle(idx int) error {
	// 已经搜过时，若在搜索路径内，说明有环
	if b.next[idx] != -1 {
		cycle := []string{b.metaNodes[idx].name}
		for cur := b.next[idx]; cur != idx; cur = b.next[cur] {
			cycle = append(cycle, b.metaNodes[cur].name)
		}
		cycle = append(cycle, b.metaNodes[idx].name)
		slices.Reverse(cycle)
		return errors.New("cyclic dependency detected: " + strings.Join(cycle, " -> "))
	}
	if b.visited[idx] {
		return nil
	}
	b.visited[idx] = true
	for _, child := range b.metaNodes[idx].children {
		b.next[idx] = child
		if err := b.detectCycle(child); err != nil {
			return err
		}
	}
	for _, weakChild := range b.metaNodes[idx].weakChildren {
		b.next[idx] = weakChild
		if err := b.detectCycle(weakChild); err != nil {
			return err
		}
	}
	b.next[idx] = -1
	return nil
}
