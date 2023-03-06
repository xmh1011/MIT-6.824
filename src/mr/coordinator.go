package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

type Coordinator struct {
	// TODO：our definitions here.

}

// TODO: Your code here -- RPC handlers for the worker to call.

//
//
// RPC 参数和回复类型在 rpc.go 中定义。
//
// 示例RPC处理函数
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
// 启动一个从 worker.go 侦听 RPC 的线程
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	// l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go周期性调用Done()函数，来检查整个工作是否已经完成。
//
func (c *Coordinator) Done() bool {
	ret := false

	// TODO: Your code here.

	return ret
}

//
// 创造一个Coordinator.
// main/mrcoordinator.go调用这个函数.
// nReduce是要reduce的次数.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// TODO: Your code here.

	c.server()
	return &c
}
