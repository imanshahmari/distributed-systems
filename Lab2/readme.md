# TDA596 Distributed systems - Lab 2

### Group 22
- Davide Canci
- Iman Shahmari Chat Gieh
- Alexander Bodin



## Build and run server:
``` docker build --build-arg portNum=80 -t tda596-lab2 . ```
``` docker run -p 80:80 -v /Users/alex/Code/distributed-systems/Lab2:/usr/src/app -it --rm --name tda596-lab2-run tda596-lab2 ```

runs the docker container with
-p exposes port 80 from docker to port on computer
-v connects a volume on computer to docker container

port number in build-arg and -p must be the same!
