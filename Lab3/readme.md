

## Build and run

First go into the correct folder
``` cd 6.824/src/main ```

Build plugin ws.so (IMPORTANT must build after every time we save, otherwise we get an error (cannot load plugin wc.so))
``` go build -race -buildmode=plugin ../mrapps/wc.go ```

Run the coordinator and then the worker(s)
``` go run -race mrcoordinator.go 1.txt 2.txt 3.txt ```
``` go run -race mrworker.go wc.so ```