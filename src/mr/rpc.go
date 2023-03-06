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

// Task worker向coordinator获取task的结构体
type Task struct {
	TaskType   TaskType // 任务类型判断到底是map还是reduce
	TaskId     int      // 任务的id
	ReducerNum int      // 传入的reducer的数量，用于hash
	FileSlice  []string // 输入文件的切片，map一个文件对应一个文件，reduce是对应多个temp中间值文件
}

// TaskArgs rpc应该传入的参数，可实际上应该什么都不用传,因为只是worker获取一个任务
type TaskArgs struct{}

// TaskType 对于下方枚举任务的父类型
type TaskType int

// Phase 对于分配任务阶段的父类型
type Phase int

// State 任务的状态的父类型
type State int

// 枚举任务的类型
const (
	MapTask TaskType = iota
	ReduceTask
	WaittingTask // Waittingen任务代表此时为任务都分发完了，但是任务还没完成，阶段未改变
	ExitTask     // exit
)

// 枚举阶段的类型
const (
	MapPhase    Phase = iota // 此阶段在分发MapTask
	ReducePhase              // 此阶段在分发ReduceTask
	AllDone                  // 此阶段已完成
)

// 任务状态类型
const (
	Working State = iota // 此阶段在工作
	Waiting              // 此阶段在等待执行
	Done                 // 此阶段已经做完
)

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
