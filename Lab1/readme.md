# TDA596 Distributed systems - Lab 1

### Group 22
- Davide Canci
- Iman Shahmari
- Alexander Bodin



## BUILD:
``` docker build --build-arg portNum=81 -t tda596-lab1 . ```

builds a docker container with the argument portNumber


## RUN:
``` docker run -p 81:81 -v /PATH/TO/CODE/Lab1:/usr/src/app -it --rm --name tda596-lab1-run tda596-lab1 ```

Ex:
``` docker run -p 81:81 -v /Users/alex/Code/distributed-systems/Lab1:/usr/src/app -it --rm --name tda596-lab1-run tda596-lab1 ```

runs the docker container with
-p exposes port from docker to port on computer
-v connects a volume on computer to docker container


``` docker build --build-arg portNum=80 --build-arg proxyUrl=172.17.0.1:81 -t tda596-lab1-proxy . ```

``` docker run -p 80:80 -v /Users/alex/Code/distributed-systems/Lab1/proxy:/usr/src/app -it --rm --name tda596-lab1-proxy-run tda596-lab1-proxy ```
