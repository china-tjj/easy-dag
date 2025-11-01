// Copyright © 2025 tjj
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package easydag

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type DAG[T any] struct {
	metaNodes []*nodeMetadata[T]
	rootNodes []int
}

// NewDAG 根据节点定义生成图，会进行环形依赖检测。至少需要传入叶子节点，会通过 dfs 扫描所有节点。
func NewDAG[T any](nodes ...*Node[T]) (*DAG[T], error) {
	return newDagBuilder(nodes).build()
}

func (dag *DAG[T]) Run(params T) []*NodeResult {
	return dag.RunWithPool(nil, params)
}

func (dag *DAG[T]) RunWithPool(pool IPool, params T) []*NodeResult {
	ctx := newDagCtx(pool)
	runtimeNodes := make([]*runtimeNode[T], len(dag.metaNodes))
	for i, node := range dag.metaNodes {
		runtimeNodes[i] = newRuntimeNode(node, ctx)
	}
	for _, node := range runtimeNodes {
		node.children = make([]*runtimeNode[T], len(node.nodeMetadata.children))
		for i, childIdx := range node.nodeMetadata.children {
			node.children[i] = runtimeNodes[childIdx]
		}
		node.weakChildren = make([]*runtimeNode[T], len(node.nodeMetadata.weakChildren))
		for i, weakChildIdx := range node.nodeMetadata.weakChildren {
			node.weakChildren[i] = runtimeNodes[weakChildIdx]
		}
	}
	for _, idx := range dag.rootNodes {
		runtimeNodes[idx].start(params)
	}
	ctx.wg.Wait()
	results := make([]*NodeResult, len(runtimeNodes))
	for i, node := range runtimeNodes {
		results[i] = node.getResult()
	}
	return results
}

func (dag *DAG[T]) ToMermaid() string {
	var str strings.Builder
	_ = dag.WriteAsMermaid(&str)
	return str.String()
}

func (dag *DAG[T]) WriteAsMermaid(writer io.StringWriter) error {
	_, err := writer.WriteString("graph TB\n")
	if err != nil {
		return err
	}
	for i, node := range dag.metaNodes {
		_, err = writer.WriteString(fmt.Sprintf("    %d(%s)\n", i, node.name))
		if err != nil {
			return err
		}
	}
	for i, node := range dag.metaNodes {
		for _, childIdx := range node.children {
			_, err = writer.WriteString(fmt.Sprintf("    %d --> %d\n", i, childIdx))
			if err != nil {
				return err
			}
		}
		for _, weakChildIdx := range node.weakChildren {
			_, err = writer.WriteString(fmt.Sprintf("    %d -.-> %d\n", i, weakChildIdx))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (dag *DAG[T]) SaveAsMermaid(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return dag.WriteAsMermaid(file)
}
