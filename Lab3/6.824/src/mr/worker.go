package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

var (
	coordinatorIp string = "localhost"
)

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

	coordinatorIp1, err := DownloadFile("coordinatorIp.txt")
	if err != nil {
		fmt.Println(err)
	}
	coordinatorIp = string(coordinatorIp1)

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

		// Is this task a map or reduce
		if task.IsMap {
			err = mapWorker(task, mapf)
		} else {
			err = reduceWorker(task, reducef)
		}
		// Only print error from MapReduce if we want to print stuff
		if printStuff && err != nil {
			fmt.Println(err)
		}
	}
}

func mapWorker(task *Task, mapf func(string, string) []KeyValue) error {
	// Read textfile
	data, err := DownloadFile(task.Filename)
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

	for i, kva := range tempFileData {
		// Convert keyvalue list to json
		data, err := json.Marshal(kva)
		if err != nil {
			return err
		}

		// Save to a file with name of format "out-[map nr]-[reduce nr]"
		filename := fmt.Sprintf("mr-%d-%d", task.TaskId, i)

		// Use tempfile to avoid reading incomplete files
		f, err := ioutil.TempFile("", filename+"-*")
		// f, err := os.Create(filename)
		if err != nil {
			return err
		}

		// Write to intermediate file
		_, err = f.Write(data)
		if err != nil {
			f.Close()
			return err
		}

		// Get name of file
		tempfilename := f.Name()
		f.Close()

		// Rename to the correct name
		err = os.Rename(tempfilename, filename)
		if err != nil {
			return err
		}

		UploadFile(filename)

	}

	// Send that we finished map task
	FinishedTask(task)
	return nil
}

func reduceWorker(task *Task, reducef func(string, []string) string) error {
	var kva []KeyValue

	for i := 0; i < task.NMax; i++ {
		// Read all intermediate map files for this reduce
		fmt.Println("Downloading:", fmt.Sprintf("mr-%d-%s", i, task.Filename))
		data, err := DownloadFile(fmt.Sprintf("mr-%d-%s", i, task.Filename))
		if err != nil {
			return err
		}

		// Convert json to kva list
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
	filename := fmt.Sprintf("mr-out-%s", task.Filename)

	// Use tempfile to avoid reading incomplete files
	f, err := ioutil.TempFile("", filename+"-*")
	//file, err := os.Create(filename)
	if err != nil {
		return err
	}

	// Make a list per key for the reduce to count
	temp := make(map[string]([]string))
	for _, keyVal := range kva {
		temp[keyVal.Key] = append(temp[keyVal.Key], keyVal.Value)
	}

	// Reduce and append line to file
	for key, list := range temp {
		value := reducef(key, list)
		fmt.Fprintf(f, "%v %v\n", key, value)
	}

	// Get name of file
	tempfilename := f.Name()
	f.Close()

	// Rename to the correct name
	err = os.Rename(tempfilename, filename)
	if err != nil {
		return err
	}

	err = UploadFile(filename)
	if err != nil {
		return err
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

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", coordinatorIp)
	//sockname := coordinatorSock()
	//c, err := rpc.DialHTTP("unix", sockname)
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
