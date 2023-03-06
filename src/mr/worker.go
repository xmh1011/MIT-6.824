package mr

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// 使用ihash（key）% NReduce为Map发出的每个KeyValue选择reduce任务编号。
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go调用这个函数
//
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {

	// TODO: Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	CallExample()

}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
// call()函数用于发送RPC请求到coordinator，等待响应。
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock() // coordinatorSock()函数返回一个唯一的socket文件名
	// DialHTTP 连接到位于指定网络地址的 HTTP RPC 服务器，侦听默认 HTTP RPC 路径
	c, err := rpc.DialHTTP("unix", sockname)
	// c为*Client类型，err为error类型
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply) // Call()函数用于发送RPC请求到coordinator，等待响应
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
