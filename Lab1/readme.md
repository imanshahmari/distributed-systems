# TDA596 Distributed systems - Lab 1

### Group 22
- Davide Canci
- Iman Shahmari
- Alexander Bodin



## Build and run server:
``` docker build --build-arg portNum=80 -t tda596-lab1 . ```
``` docker run -p 80:80 -v /Users/alex/Code/distributed-systems/Lab1:/usr/src/app -it --rm --name tda596-lab1-run tda596-lab1 ```

runs the docker container with
-p exposes port 80 from docker to port on computer
-v connects a volume on computer to docker container

port number in build-arg and -p must be the same!

## Build and run proxy
``` cd proxy ```
``` docker build --build-arg portNum=81 --build-arg proxyUrl=172.17.0.1 -t tda596-lab1-proxy . ```
``` docker run -p 81:81 -v /Users/alex/Code/distributed-systems/Lab1/proxy:/usr/src/app -it --rm --name tda596-lab1-proxy-run tda596-lab1-proxy ```

the build-arg proxyUrl is the url to where the proxy should redirect all traffic
172.17.0.1 is the docker containers ip to the host (my computer)