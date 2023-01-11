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

const printStuff = true

// Only internal representation of a task (see rpc.go for communication)
type TaskData struct {
	filename    string
	stage       string // one of ["waiting", "running", "done"]
	startedTime time.Time
	isMap       bool
}

type Coordinator struct {
	mu      sync.Mutex // Allows for mutual exclusion of threads to avoid race conditions
	nReduce int
	nMap    int
	tasks   []TaskData
	done    bool
	mapDone bool
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
		// Only give reduce tasks after all map tasks are done
		if (task.isMap || c.mapDone) &&
			(task.stage == "waiting" ||
				(task.stage == "running" && runtime > maxTimeout)) {

			if printStuff {
				log.Println("Running ", task.filename)
			}

			// Reply with the filename to process and index in list
			reply.Filename = task.filename
			reply.TaskId = i
			reply.IsMap = task.isMap
			if task.isMap {
				reply.NMax = c.nReduce
			} else {
				reply.NMax = c.nMap
			}

			// Save the start-time and update stage of task
			c.tasks[i].startedTime = time.Now()
			c.tasks[i].stage = "running"
			return nil
		}
	}

	// If we are done, return error to exit otherwise wait for running tasks to finish
	if c.done {
		c.downloadResults()
		return fmt.Errorf("no next task, exiting")
	} else {
		time.Sleep(time.Second)
		return nil
	}
}

func (c *Coordinator) TaskDone(args *Task, reply *Task) error {
	if printStuff {
		log.Println("Done    ", args.Filename)
	}

	// Safely write to coordinator
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tasks[args.TaskId].stage = "done"

	// Check if all map tasks are done
	if !c.mapDone {
		for _, task := range c.tasks {
			if task.isMap && task.stage != "done" {
				return nil
			}
		}
		// If we get here all map tasks are done, continue with reduce tasks
		c.mapDone = true
	}

	// Check if all tasks are done
	for _, task := range c.tasks {
		if !task.isMap && task.stage != "done" {
			return nil
		}
	}
	// If we get here all map and reduce tasks are done
	c.done = true

	return nil
}

func (c *Coordinator) downloadResults() {
	for i := 0; i < c.nReduce; i++ {
		name := fmt.Sprintf("mr-out-%d", i)

		fmt.Println("Retriveing result:", name)
		data, err := DownloadFile(name)
		if err != nil {
			fmt.Println(err)
		}

		f, err := os.Create(name)
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()

		_, err = f.Write(data)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	//sockname := coordinatorSock()
	//os.Remove(sockname)
	//l, e := net.Listen("unix", sockname)
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
		nMap:    len(files),
		tasks:   make([]TaskData, len(files)+nReduce),
		done:    false,
		mapDone: false,
	}

	// Create mapping tasks
	for i, file := range files {
		c.tasks[i] = TaskData{
			filename: file,
			stage:    "waiting",
			isMap:    true,
		}
		UploadFile(file)
	}

	// Reducing jobs
	for i := 0; i < nReduce; i++ {
		c.tasks[len(files)+i] = TaskData{
			// Using filname here to store reduce number (last part of filename)
			// this is because there are multiple ways files are read and stored
			// in the reduce function
			filename: fmt.Sprint(i),
			stage:    "waiting",
			isMap:    false,
		}
	}

	c.server()
	return &c
}
