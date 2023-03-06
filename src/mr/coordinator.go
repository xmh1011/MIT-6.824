package mr

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 定义为全局，worker之间访问coordinator时加锁
var mu sync.Mutex

type Coordinator struct {
	// TODO：our definitions here.
	ReducerNum        int            // 传入的参数决定需要多少个reducer
	TaskId            int            // 用于生成task的特殊id
	DistPhase         Phase          // 目前整个框架应该处于什么任务阶段
	ReduceTaskChannel chan *Task     // 使用chan保证并发安全
	MapTaskChannel    chan *Task     // 使用chan保证并发安全
	taskMetaHolder    TaskMetaHolder // 存着task
	files             []string       // 传入的文件数组
}

// TaskMetaHolder 保存全部任务的元数据
type TaskMetaHolder struct {
	MetaMap map[int]*TaskMetaInfo // 通过下标hash快速定位
	// taskMetaHolder为存放全部元信息(TaskMetaInfo)的map，当然用slice也行。
}

// TaskMetaInfo 保存任务的元数据
type TaskMetaInfo struct {
	state     State     // 任务的状态
	StartTime time.Time // 任务的开始时间，为crash做准备
	TaskAdr   *Task     // 传入任务的指针,为的是这个任务从通道中取出来后，还能通过地址标记这个任务已经完成
}

// TODO: Your code here -- RPC handlers for the worker to call.

//
//
// RPC 参数和回复类型在 rpc.go 中定义。
//
// 示例RPC处理函数
// func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
// 	reply.Y = args.X + 1
// 	return nil
// }

//
// Start a thread that listens for RPCs from worker.go
// 启动一个从 worker.go 侦听 RPC 的线程.
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
// 创造一个Coordinator.
// main/mrcoordinator.go调用这个函数.
// nReduce是要reduce的次数.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// TODO: Your code here.
	c := Coordinator{
		files:             files,
		ReducerNum:        nReduce,
		DistPhase:         MapPhase,
		MapTaskChannel:    make(chan *Task, len(files)),
		ReduceTaskChannel: make(chan *Task, nReduce),
		taskMetaHolder: TaskMetaHolder{
			MetaMap: make(map[int]*TaskMetaInfo, len(files)+nReduce), // 任务的总数应该是files + Reducer的数量
		},
	}
	c.makeMapTasks(files)

	c.server()

	go c.CrashDetector()

	return &c
}

// CrashDetector crash探测协程的实现
func (c *Coordinator) CrashDetector() {
	for {
		time.Sleep(time.Second * 2)
		mu.Lock()
		if c.DistPhase == AllDone {
			mu.Unlock()
			break
		}

		for _, v := range c.taskMetaHolder.MetaMap {
			if v.state == Working {
				// fmt.Println("task[", v.TaskAdr.TaskId, "] is working: ", time.Since(v.StartTime), "s")
			}

			if v.state == Working && time.Since(v.StartTime) > 9*time.Second {
				fmt.Printf("the task[ %d ] is crash,take [%d] s\n", v.TaskAdr.TaskId, time.Since(v.StartTime))

				switch v.TaskAdr.TaskType {
				case MapTask:
					c.MapTaskChannel <- v.TaskAdr
					v.state = Waiting
				case ReduceTask:
					c.ReduceTaskChannel <- v.TaskAdr
					v.state = Waiting

				}
			}
		}
		mu.Unlock()
	}

}

// 实现上方的makeMapTasks：将Map任务放到Map管道中，taskMetaInfo放到taskMetaHolder中。
// 对map任务进行处理,初始化map任务
func (c *Coordinator) makeMapTasks(files []string) {

	for _, v := range files {
		id := c.generateTaskId()
		task := Task{
			TaskType:   MapTask,
			TaskId:     id,
			ReducerNum: c.ReducerNum,
			FileSlice:  append([]string{}, v),
		}

		// 保存任务的初始状态
		taskMetaInfo := TaskMetaInfo{
			state:   Waiting, // 任务等待被执行
			TaskAdr: &task,   // 保存任务的地址
		}
		c.taskMetaHolder.acceptMeta(&taskMetaInfo)

		fmt.Println("make a map task :", &task)
		c.MapTaskChannel <- &task
	}
}

// 实现上方的makeReduceTasks：将Reduce任务放到Reduce管道中，taskMetaInfo放到taskMetaHolder中。
func (c *Coordinator) makeReduceTasks() {
	for i := 0; i < c.ReducerNum; i++ {
		id := c.generateTaskId()
		task := Task{
			TaskId:    id,
			TaskType:  ReduceTask,
			FileSlice: selectReduceName(i),
		}

		// 保存任务的初始状态
		taskMetaInfo := TaskMetaInfo{
			state:   Waiting, // 任务等待被执行
			TaskAdr: &task,   // 保存任务的地址
		}
		c.taskMetaHolder.acceptMeta(&taskMetaInfo)

		// fmt.Println("make a reduce task :", &task)
		c.ReduceTaskChannel <- &task
	}
}

func selectReduceName(reduceNum int) []string {
	var s []string
	path, _ := os.Getwd()
	files, _ := ioutil.ReadDir(path)
	for _, fi := range files {
		// 匹配对应的reduce文件
		if strings.HasPrefix(fi.Name(), "mr-tmp") && strings.HasSuffix(fi.Name(), strconv.Itoa(reduceNum)) {
			s = append(s, fi.Name())
		}
	}
	return s
}

// 通过结构体的TaskId自增来获取唯一的任务id
func (c *Coordinator) generateTaskId() int {

	res := c.TaskId
	c.TaskId++
	return res
}

// 将接受taskMetaInfo储存进MetaHolder里
func (t *TaskMetaHolder) acceptMeta(TaskInfo *TaskMetaInfo) bool {
	taskId := TaskInfo.TaskAdr.TaskId
	meta, _ := t.MetaMap[taskId]
	if meta != nil {
		fmt.Println("meta contains task which id = ", taskId)
		return false
	} else {
		t.MetaMap[taskId] = TaskInfo
	}
	return true
}

// 分配任务中转换阶段的实现
func (c *Coordinator) toNextPhase() {
	if c.DistPhase == MapPhase {
		c.makeReduceTasks()
		c.DistPhase = ReducePhase
	} else if c.DistPhase == ReducePhase {
		c.DistPhase = AllDone
	}
}

// 检查多少个任务做了包括（map、reduce）,
func (t *TaskMetaHolder) checkTaskDone() bool {

	var (
		mapDoneNum      = 0
		mapUnDoneNum    = 0
		reduceDoneNum   = 0
		reduceUnDoneNum = 0
	)

	// 遍历储存task信息的map
	for _, v := range t.MetaMap {
		// 首先判断任务的类型
		if v.TaskAdr.TaskType == MapTask {
			// 判断任务是否完成,下同
			if v.state == Done {
				mapDoneNum++
			} else {
				mapUnDoneNum++
			}
		} else if v.TaskAdr.TaskType == ReduceTask {
			if v.state == Done {
				reduceDoneNum++
			} else {
				reduceUnDoneNum++
			}
		}

	}
	// fmt.Printf("map tasks  are finished %d/%d, reduce task are finished %d/%d \n",
	// mapDoneNum, mapDoneNum+mapUnDoneNum, reduceDoneNum, reduceDoneNum+reduceUnDoneNum)

	// 如果某一个map或者reduce全部做完了，代表需要切换下一阶段，返回true

	// Map
	if (mapDoneNum > 0 && mapUnDoneNum == 0) && (reduceDoneNum == 0 && reduceUnDoneNum == 0) {
		return true
	} else {
		if reduceDoneNum > 0 && reduceUnDoneNum == 0 {
			return true
		}
	}

	return false

}

// 判断给定任务是否在工作，并修正其目前任务信息状态，如果任务不在工作的话返回true
func (t *TaskMetaHolder) judgeState(taskId int) bool {
	taskInfo, ok := t.MetaMap[taskId]
	if !ok || taskInfo.state != Waiting {
		return false
	}
	taskInfo.state = Working
	return true
}

func (c *Coordinator) MarkFinished(args *Task, reply *Task) error {
	mu.Lock()
	defer mu.Unlock()
	switch args.TaskType {
	case MapTask:
		meta, ok := c.taskMetaHolder.MetaMap[args.TaskId]

		// prevent a duplicated work, which returned from another worker
		if ok && meta.state == Working {
			meta.state = Done
			fmt.Printf("Map task Id[%d] is finished.\n", args.TaskId)
		} else {
			fmt.Printf("Map task Id[%d] is finished,already ! ! !\n", args.TaskId)
		}
		break

	default:
		panic("The task type undefined ! ! !")
	}
	return nil

}

//
// main/mrcoordinator.go周期性调用Done()函数，来检查整个工作是否已经完成。
//

// 如果map任务全部实现完（暂且略过reduce）阶段为AllDone那么Done方法应该返回true,使协调者能够exit程序。
// Done 主函数mr调用，如果所有task完成mr会通过此方法退出
func (c *Coordinator) Done() bool {
	// TODO: Your code here.
	mu.Lock()
	defer mu.Unlock()
	if c.DistPhase == AllDone {
		fmt.Printf("All tasks are finished,the coordinator will be exit! !")
		return true
	} else {
		return false
	}

}

// 将map任务管道中的任务取出，如果取不出来，说明任务已经取尽，那么此时任务要么就已经完成，要么就是正在进行。
// 判断任务map任务是否先完成，如果完成那么应该进入下一个任务处理阶段（ReducePhase），
// 因为此时我们先验证map则直接跳过reduce直接allDone全部完成。
// 分发任务
func (c *Coordinator) PollTask(args *TaskArgs, reply *Task) error {
	// 分发任务应该上锁，防止多个worker竞争，并用defer回退解锁
	mu.Lock()
	defer mu.Unlock()

	// 判断任务类型存任务
	switch c.DistPhase {
	case MapPhase:
		{
			if len(c.MapTaskChannel) > 0 {
				*reply = *<-c.MapTaskChannel
				if !c.taskMetaHolder.judgeState(reply.TaskId) {
					fmt.Printf("Map-taskid[ %d ] is running\n", reply.TaskId)
				}
			} else {
				reply.TaskType = WaittingTask // 如果map任务被分发完了但是又没完成，此时就将任务设为Waiting
				if c.taskMetaHolder.checkTaskDone() {
					c.toNextPhase()
				}
				return nil
			}
		}

	case ReducePhase:
		{
			if len(c.ReduceTaskChannel) > 0 {
				*reply = *<-c.ReduceTaskChannel
				if !c.taskMetaHolder.judgeState(reply.TaskId) {
					fmt.Printf("Reduce-taskid[ %d ] is running\n", reply.TaskId)
				}
			} else {
				reply.TaskType = WaittingTask // 如果map任务被分发完了但是又没完成，此时就将任务设为Waiting
				if c.taskMetaHolder.checkTaskDone() {
					c.toNextPhase()
				}
				return nil
			}
		}

	case AllDone:
		{
			reply.TaskType = ExitTask
		}
	default:
		panic("The phase undefined ! ! !")

	}

	return nil
}
