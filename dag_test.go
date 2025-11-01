package easydag

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

func TestCycle(t *testing.T) {
	node1 := &Node[struct{}]{Name: "node1"}
	node2 := &Node[struct{}]{Name: "node2"}
	node3 := &Node[struct{}]{Name: "node3"}
	node1.AddDependency(node3)
	node2.AddDependency(node1)
	node3.AddDependency(node2)
	_, err := NewDAG(node3)
	if err == nil {
		t.Fatal("cycle detect err")
	}
	if err.Error() != "cyclic dependency detected: node3 -> node2 -> node1 -> node3" {
		t.Fatal("cycle detect err:", err.Error())
	}
}

func BenchmarkPool(b *testing.B) {
	var simpleFib func(i int) int
	simpleFib = func(i int) int {
		if i <= 1 {
			return 1
		}
		return simpleFib(i-1) + simpleFib(i-2)
	}
	process := func(node IRuntimeNode, _ struct{}) error {
		simpleFib(10)
		return nil
	}
	var nodes []*Node[struct{}]
	for i := 0; i < 30; i++ {
		node := &Node[struct{}]{
			Name:      fmt.Sprintf("node-%d", i),
			Processor: process,
		}
		node.AddDependency(nodes...)
		nodes = append(nodes, node)
	}
	dag, err := NewDAG[struct{}](nodes...)
	if err != nil {
		b.Fatal(err)
	}

	defaultPool := NewPool(math.MaxInt)
	b.Run("nopool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			for j := 0; j < 1000; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					dag.Run(struct{}{})
				}()
			}
			wg.Wait()
		}
	})
	b.Run("pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			for j := 0; j < 1000; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					dag.RunWithPool(defaultPool, struct{}{})
				}()
			}
			wg.Wait()
		}
	})
}
