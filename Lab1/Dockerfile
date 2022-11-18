FROM golang:1.19

# Specify which port to use for the server with --build-args
ARG portNum

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app ./...

EXPOSE ${portNum}

# Convert build variable to use for runtime
ENV portNumEnv=${portNum}

CMD "app" ${portNumEnv}