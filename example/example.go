package main

import (
	"fmt"
	"github.com/china-tjj/easy-dag"
	"time"
)

type Params struct {
	v1 int
	v2 int
	v3 int
}

func processor1(node easydag.IRuntimeNode, params *Params) error {
	params.v1 = 1
	fmt.Println("node1 success")
	return nil
}

func processor2(node easydag.IRuntimeNode, params *Params) error {
	// 模拟远程调用
	time.Sleep(2 * time.Millisecond)
	ok := node.DoIfRunning(func() {
		params.v2 = 10
	})
	if ok {
		fmt.Println("node2 success")
	} else {
		fmt.Println("node2 timeout")
	}
	return nil
}

func processor3(node easydag.IRuntimeNode, params *Params) error {
	params.v3 = params.v1 + params.v2
	fmt.Println("node3 success")
	return nil
}

func main() {
	node1 := &easydag.Node[*Params]{
		Name:      "node1",
		Processor: processor1,
	}
	node2 := &easydag.Node[*Params]{
		Name:         "node2",
		LocalTimeout: 1 * time.Millisecond,
		Processor:    processor2,
		Dependencies: []*easydag.Node[*Params]{node1},
	}
	node3 := &easydag.Node[*Params]{
		Name:             "node3",
		Processor:        processor3,
		WeakDependencies: []*easydag.Node[*Params]{node2},
	}
	dag, err := easydag.NewDAG(node3)
	if err != nil {
		panic(err)
	}
	err = dag.SaveAsMermaid("./example/example.mermaid")
	if err != nil {
		panic(err)
	}
	params := &Params{}
	dag.Run(params)
	fmt.Printf("%+v\n", params)
}
