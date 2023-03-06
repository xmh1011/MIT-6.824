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

// TODO: example to show how to declare the arguments and reply for an RPC.
// 示例，展示如何声明RPC的参数
type ExampleArgs struct {
	X int
}

// 示例，展示如何声明RPC的返回值
type ExampleReply struct {
	Y int
}

// TODO: Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
// 在 /var/tmp 中为协调器准备一个唯一的 UNIX 域套接字名称。
// 无法使用当前目录, 因为 Athena AFS 不支持 UNIX 域套接字。
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	// Getuid() 函数返回调用者的用户 ID
	s += strconv.Itoa(os.Getuid()) // strconv.Itoa() 函数将整型转换为字符串
	// 返回的s的形式为：/var/tmp/824-mr-1000
	return s
}
