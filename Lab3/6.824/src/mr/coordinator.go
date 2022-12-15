package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

const testing = false

type TaskData struct {
	filename    string
	stage       string // one of ["waiting", "running", "done"]
	workerId    int    // maybe not needed
	startedTime time.Time
}

type Coordinator struct {
	// Your definitions here.
	mu      sync.Mutex // Allows for mutual exclusion of threads to avoid race conditions
	nReduce int
	tasks   []TaskData
	done    bool
}

var (
	maxTimeout time.Duration = time.Second * 10
)

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) NextTask(args *ExampleArgs, reply *Task) error {
	// Safely read/write to coordinator
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, task := range c.tasks {
		var runtime time.Duration = time.Since(task.startedTime)

		// Give first task that is unassigned or expired (too long runtime)
		// TODO for debugging, remove testing here before publishing
		if testing || task.stage == "waiting" ||
			(task.stage == "running" && runtime > maxTimeout) {
			log.Println("Running ", task.filename)

			// Reply with the filename to process and index in list
			reply.Filename = task.filename
			reply.TaskId = i

			// Save the start-time and update stage of task
			c.tasks[i].startedTime = time.Now()
			c.tasks[i].workerId = args.X
			c.tasks[i].stage = "running"
			return nil
		}
	}

	c.done = true
	return fmt.Errorf("no next task, exiting")
}

func (c *Coordinator) TaskDone(args *Task, reply *Task) error {
	log.Println("Done    ", args.Filename)

	// Safely write to coordinator
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tasks[args.TaskId].stage = "done"
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	// Your code here.

	// Safely read from coordinator
	c.mu.Lock()
	defer c.mu.Unlock()

	//for _, task := range c.tasks {
	//	fmt.Print(task.filename, " ", task.stage, "\t")
	//}
	//fmt.Print("\n")
	return c.done
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// Your code here.

	c := Coordinator{
		nReduce: nReduce,
		tasks:   make([]TaskData, len(files)),
		done:    false,
	}

	for i, file := range files {
		c.tasks[i] = TaskData{
			filename: file,
			stage:    "waiting",
		}
	}

	c.server()
	return &c
}
