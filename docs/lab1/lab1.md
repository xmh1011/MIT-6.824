### 任务分析
如上文所述，在 main/mrsequential.go 中我们可以找到初始代码预先提供的 单进程串行 的 MapReduce 参考实现，而我们的任务是实现一个 单机多进程并行 的版本。

通过阅读 Lab 文档 http://nil.csail.mit.edu/6.824/2021/labs/lab-mr.html 以及初始代码，可知信息如下：

  - Coordinator 进程与 Worker 进程间通过本地 Socket 进行 Golang RPC 通信
  - 由 Coordinator 协调整个 MR 计算的推进，并分配 Task 到 Worker 上运行
  - 在启动 Coordinator 进程时指定 输入文件名 及 Reduce Task 数量
  - 在启动 Worker 进程时指定所用的 MR APP 动态链接库文件
  - Coordinator 需要留意 Worker 可能无法在合理时间内完成收到的任务（Worker 卡死或宕机），在遇到此类问题时需要重新派发任务
  - Coordinator 进程的入口文件为 main/mrcoordinator.go
  - Worker 进程的入口文件为 main/mrworker.go
  - 我们需要补充实现 mr/coordinator.go、mr/worker.go、mr/rpc.go 这三个文件

基于此，我们不难设计出，Coordinator 需要有以下功能：

- 在启动时根据指定的输入文件数及 Reduce Task 数，生成 Map Task 及 Reduce Task
- 响应 Worker 的 Task 申请 RPC 请求，分配可用的 Task 给到 Worker 处理
- 追踪 Task 的完成情况，在所有 Map Task 完成后进入 Reduce 阶段，开始派发 Reduce Task；在所有 Reduce Task 完成后标记作业已完成并退出

而 Worker 的功能则相对简单，只需要保证在空闲时通过 RPC 向 Coordinator 申请 Task 并运行，再不断重复该过程即可。

此外 Lab 要求我们考虑 Worker 的 Failover，即 Worker 获取到 Task 后可能出现宕机和卡死等情况。这两种情况在 Coordinator 的视角中都是相同的，就是该 Worker 长时间不与 Coordinator 通信了。为了简化任务，Lab 说明中明确指定了，设定该超时阈值为 10s 即可。为了支持这一点，我们的实现需要支持到：

- Coordinator 追踪已分配 Task 的运行情况，在 Task 超出 10s 仍未完成时，将该 Task 重新分配给其他 Worker 重试
- 考虑 Task 上一次分配的 Worker 可能仍在运行，重新分配后会出现两个 Worker 同时运行同一个 Task 的情况。要确保只有一个 Worker 能够完成结果数据的最终写出，以免出现冲突，导致下游观察到重复或缺失的结果数据

第一点比较简单，而第二点会相对复杂些，不过在 Lab 文档中也给出了提示 —— 实际上也是参考了 Google MapReduce 的做法，Worker 在写出数据时可以先写出到临时文件，最终确认没有问题后再将其重命名为正式结果文件，区分开了 Write 和 Commit 的过程。Commit 的过程可以是 Coordinator 来执行，也可以是 Worker 来执行：

Coordinator Commit：Worker 向 Coordinator 汇报 Task 完成，Coordinator 确认该 Task 是否仍属于该 Worker，是则进行结果文件 Commit，否则直接忽略
Worker Commit：Worker 向 Coordinator 汇报 Task 完成，Coordinator 确认该 Task 是否仍属于该 Worker 并响应 Worker，是则 Worker 进行结果文件 Commit，再向 Coordinator 汇报 Commit 完成
这里两种方案都是可行的，各有利弊。我在我的实现中选择了 Coordinator Commit，因为它可以少一次 RPC 调用，在编码实现上会更简单，但缺点是所有 Task 的最终 Commit 都由 Coordinator 完成，在极端场景下会让 Coordinator 变成整个 MR 过程的性能瓶颈。

### 代码设计与实现
代码的设计及实现主要是三个部分：

- Coordinator 与 Worker 间的 RPC 通信，对应 mr/rpc.go 文件
- Coordinator 调度逻辑，对应 mr/coordinator.go 文件
- Worker 计算逻辑，对应 mr/worker.go 文件

### RPC 通信
Coordinator 与 Worker 间的需要进行的通信主要有两块：

- Worker 在空闲时向 Coordinator 发起 Task 请求，Coordinator 响应一个分配给该 Worker 的 Task
- Worker 在上一个 Task 运行完成后向 Coordinator 汇报

考虑到上述两个过程总是交替进行的，且 Worker 在上一个 Task 运行完成后总是立刻会需要申请一个新的 Task，在实现上这里我把它们合并为了一个 ApplyForTask RPC 调用：

- 由 Worker 向 Coordinator 发起，申请一个新的 Task，同时汇报上一个运行完成的 Task（如有）
- Coordinator 接收到 RPC 请求后将同步阻塞，直到有可用的 Task 分配给该 Worker 或整个 MR 作业已运行完成