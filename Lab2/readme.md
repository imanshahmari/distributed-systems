# TDA596 Distributed systems - Lab 2

### Group 22
- Davide Canci
- Iman Shahmari Chat Gieh
- Alexander Bodin


## Run go

``` go run Chord.go Networking.go cli.go -a localhost -p 1100 ```
``` go run Chord.go Networking.go cli.go -a localhost -p 1110 -ja localhost -jp 1100 ```
etc.


## Build and run server:
``` docker build --build-arg portNum=80 -t tda596-lab2 . ```
``` docker run -p 80:80 -v /Users/alex/Code/distributed-systems/Lab2:/usr/src/app -it --rm --name tda596-lab2-run tda596-lab2 ```

runs the docker container with
-p exposes port 80 from docker to port on computer
-v connects a volume on computer to docker container

port number in build-arg and -p must be the same!
go run . -p 1403 -jp 1400





### Known bugs (dont do this)
After replication of files the replication of the storing node is not correct
( ie the first bucket has the correctly updated storingNode but it is not replicated automatically to the next one)

Too short time interval for tff creates EOF errors

(fixed??) Killing nodes too quickly can result in missing predecessors/successors

If nodes are initialized with small ids the finger table does not work

Figer table wrong????