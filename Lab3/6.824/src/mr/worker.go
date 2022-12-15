package mr

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"sort"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	for {
		// Get a new task
		task, err := AskForTask()
		if err != nil {
			fmt.Println(err)
			return
		}

		//time.Sleep(time.Second * 5)

		data, err := os.ReadFile(task.Filename)
		if err != nil {
			log.Println(err)
			continue
		}

		// Map (for ws: create a dictionary of words (with value 1))
		kva := mapf(task.Filename, string(data))

		sort.Sort(ByKey(kva))

		// Reduce (for ws: count all occurances of word)
		// Create outputfile
		//reduceId := 0
		filename := fmt.Sprintf("mr-out-%d", task.TaskId) //, reduceId)
		file, _ := os.Create(filename)
		defer file.Close()

		//reduce(task.Filename, kva, reducef)
		temp := make(map[string]([]string))
		for _, keyVal := range kva {
			temp[keyVal.Key] = append(temp[keyVal.Key], keyVal.Value)
		}

		for key, list := range temp {
			value := reducef(key, list)
			fmt.Fprintf(file, "%v %v\n", key, value)
		}

		// Send that we finished task
		FinishedTask(task)

		// TODO for debugging, remove before publishing
		if testing {
			return
		}
	}
}

func reduce(filename string, kva []KeyValue,
	reducef func(string, []string) string) {
}

func AskForTask() (*Task, error) {
	args := ExampleArgs{99}
	reply := Task{}

	ok := call("Coordinator.NextTask", &args, &reply)
	if ok {
		fmt.Printf("reply.filename %v\n", reply.Filename)
	} else {
		//fmt.Printf("call failed!\n")
		return &reply, fmt.Errorf("failed or no more tasks, quitting!")
	}
	return &reply, nil
}

func FinishedTask(task *Task) {
	call("Coordinator.TaskDone", task, &Task{})
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
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

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
