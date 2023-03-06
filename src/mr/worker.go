package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"time"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

type SortedKey []KeyValue

// Len 重写len,swap,less才能排序
func (k SortedKey) Len() int           { return len(k) }
func (k SortedKey) Swap(i, j int)      { k[i], k[j] = k[j], k[i] }
func (k SortedKey) Less(i, j int) bool { return k[i].Key < k[j].Key }

//
// 使用ihash（key）% NReduce为Map发出的每个KeyValue选择reduce任务编号。
// use ihash(key) % NReduce to choose to reduce
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

	// CallExample()
	keepFlag := true
	for keepFlag {
		task := GetTask()
		switch task.TaskType {
		case MapTask:
			{
				DoMapTask(mapf, &task)
				callDone(&task)
			}

		case WaittingTask:
			{
				// fmt.Println("All tasks are in progress, please wait...")
				time.Sleep(time.Second)
			}

		case ReduceTask:
			{
				DoReduceTask(reducef, &task)
				callDone(&task)
			}

		case ExitTask:
			{
				// fmt.Println("Task about :[", task.TaskId, "] is terminated...")
				keepFlag = false
			}

		}
	}

	// uncomment to send the Example RPC to the coordinator.

}

// GetTask 获取任务（需要知道是Map任务，还是Reduce）
func GetTask() Task {

	args := TaskArgs{}
	reply := Task{}
	ok := call("Coordinator.PollTask", &args, &reply)

	if ok {
		fmt.Println(reply)
	} else {
		fmt.Printf("call failed!\n")
	}
	return reply

}

// DoMapTask 参考wc.go和mrsequential.go，完成map函数
func DoMapTask(mapf func(string, string) []KeyValue, response *Task) {
	var intermediate []KeyValue

	// 遍历文件切片
	for _, filename := range response.FileSlice {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		// 通过io工具包获取content,作为mapf的参数
		content, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("cannot read %v", filename)
		}
		file.Close()
		// map返回一组KV结构体数组
		intermediate = mapf(filename, string(content))

		// initialize and loop over []KeyValue
		rn := response.ReducerNum
		// 创建一个长度为nReduce的二维切片
		HashedKV := make([][]KeyValue, rn)

		for _, kv := range intermediate {
			HashedKV[ihash(kv.Key)%rn] = append(HashedKV[ihash(kv.Key)%rn], kv)
		}
		for i := 0; i < rn; i++ {
			// 生成中间文件名 形式为mr-tmp-0-0
			oname := "mr-tmp-" + strconv.Itoa(response.TaskId) + "-" + strconv.Itoa(i)
			ofile, _ := os.Create(oname)
			enc := json.NewEncoder(ofile)
			for _, kv := range HashedKV[i] {
				enc.Encode(kv)
			}
			ofile.Close()
		}
	}

}

func DoReduceTask(reducef func(string, []string) string, response *Task) {
	reduceFileNum := response.TaskId
	intermediate := shuffle(response.FileSlice)
	dir, _ := os.Getwd()
	// tempFile, err := ioutil.TempFile(dir, "mr-tmp-*")
	tempFile, err := ioutil.TempFile(dir, "mr-tmp-*")
	if err != nil {
		log.Fatal("Failed to create temp file", err)
	}
	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		var values []string
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)
		fmt.Fprintf(tempFile, "%v %v\n", intermediate[i].Key, output)
		i = j
	}
	tempFile.Close()
	fn := fmt.Sprintf("mr-out-%d", reduceFileNum)
	os.Rename(tempFile.Name(), fn)
}

// 洗牌方法，得到一组排序好的kv数组
func shuffle(files []string) []KeyValue {
	var kva []KeyValue
	for _, filepath := range files {
		file, _ := os.Open(filepath)
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			kva = append(kva, kv)
		}
		file.Close()
	}
	sort.Sort(SortedKey(kva))
	return kva
}

// CallExample
// example function to show how to make an RPC call to the coordinator.
//
// The RPC argument and reply types are defined in rpc.go.
// 已给的示例函数, 用于展示如何向coordinator发送RPC请求。.
/*
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
*/

//
// send an RPC request to the coordinator, wait for the response.
// Usually returns true.
// Returns false if something goes wrong.
// call()函数用于发送RPC请求到coordinator，等待响应。.
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

// callDone Call RPC to mark the task as completed
func callDone(f *Task) Task {

	args := f
	reply := Task{}
	ok := call("Coordinator.MarkFinished", &args, &reply)

	if ok {
		// fmt.Println("worker finish :taskId[", args.TaskId, "]")
	} else {
		fmt.Printf("call failed!\n")
	}
	return reply

}
