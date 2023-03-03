package mr

//
// RPC definitions.
//
// remember to capitalize all names.
// 所有的RPC都是大写的
//

import (
	"os"
	"strconv"
)

// example to show how to declare the arguments
// and reply for an RPC.
//
// 示例，展示如何声明RPC的参数
type ExampleArgs struct {
	X int
}

// 示例，展示如何声明RPC的返回值
type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
