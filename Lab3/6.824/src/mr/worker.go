package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"time"
	//"sync"
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
	//var wg sync.WaitGroup

	for {
		// Get a new task
		task, err := AskForTask()
		if err != nil {
			if printStuff {
				fmt.Println(err)
			}
			return
		}

		// If reply is empty but not failed, means we are waiting for tasks to finish (can't exit as a task might timeout on other worker)
		if task.Filename == "" {
			time.Sleep(time.Second)
			continue
		}

		if task.IsMap {
			err = mapWorker(task, mapf)
		} else {
			err = reduceWorker(task, reducef)
		}
		if printStuff && err != nil {
			fmt.Println(err)
		}
	}
}

func mapWorker(task *Task, mapf func(string, string) []KeyValue) error {
	data, err := os.ReadFile(task.Filename)
	if err != nil {
		return err
	}

	// Map (for ws: create a dictionary of words (with value 1))
	kva := mapf(task.Filename, string(data))

	// Initialize array of data to print to file
	tempFileData := make([][]KeyValue, task.NMax)
	for i := 0; i < task.NMax; i++ {
		tempFileData[i] = make([]KeyValue, 0, 10)
	}

	// Add each keyvalue to its hashed reduce index
	for _, keyVal := range kva {
		i := ihash(keyVal.Key) % task.NMax
		tempFileData[i] = append(tempFileData[i], keyVal)
	}

	// Save each reduce to a file with name of format "out-[map nr]-[reduce nr]"
	for i, kva := range tempFileData {

		filename := fmt.Sprintf("mr-%d-%d", task.TaskId, i)
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		data, err := json.Marshal(kva)
		if err != nil {
			return err
		}

		_, err = f.Write(data)
		if err != nil {
			return err
		}

	}

	FinishedTask(task)
	return nil
}

func reduceWorker(task *Task, reducef func(string, []string) string) error {
	// Read the intermediate files

	var kva []KeyValue

	for i := 0; i < task.NMax; i++ {
		data, err := os.ReadFile(fmt.Sprintf("mr-%d-%s", i, task.Filename))
		if err != nil {
			return err
		}

		var kvaTemp []KeyValue
		err = json.Unmarshal(data, &kvaTemp)
		if err != nil {
			return err
		}

		// Combine all kva's
		kva = append(kva, kvaTemp...)
	}

	// Reduce (for ws: count all occurances of word)
	// Create outputfile
	//reduceId := 0
	filename := fmt.Sprintf("mr-out-%s", task.Filename) //, reduceId)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
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
	return nil
}

func AskForTask() (*Task, error) {
	args := ExampleArgs{99}
	reply := Task{}

	ok := call("Coordinator.NextTask", &args, &reply)
	if ok {
		if printStuff {
			fmt.Printf("reply.filename %v\n", reply.Filename)
		}
	} else {
		//fmt.Printf("call failed!\n")
		return &reply, fmt.Errorf("failed or no more tasks, quitting")
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

	if printStuff || fmt.Sprint(err) != "no next task, exiting" {
		fmt.Println(err)
	}
	return false
}
