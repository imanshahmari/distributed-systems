# TDA596 Distributed systems - Lab 2

### Group 22
- Alexander Bodin
- Davide Canci
- Iman Shahmari Chat Gieh


## Run go
``` go run . -a localhost -p 1100 ```
``` go run . -a localhost -p 1110 -ja localhost -jp 1100 ```
etc.


## Features

Create network
Join network ( automatically updates pred and succ )

Store file with secure fault-tolerance replication of both file data and bucket "pointers"

Node failure updates to correct nodes and redo replication to keep files secure and fult-tolerant

Lookup files