# dummybox

DummyBox is a swiss knife tool which allows to mock container behaviours in a cluster. 

The goal is to use it for validating monitoring settings (logging messages, metrics, alerts, etc.) and cluster settings (connectivity, autoscaling, RBAC, etc.).

The tasks:
- Generate cpu/memory utilization
- Perform HTTP requests
  - with arbitrary HTTP response status codes
  - with delays
- Get information of the running container
- Kill the container with a specific status code
- Produce log messages in the console (stout and sterr)

## Usage

Install `ko`: 

```bash
go get github.com/google/ko/cmd/ko
```

Init the go module:
  
```bash
go mod init github.com/crlsmrls/dummybox
```

Build and publish the container image:

```bash
VERSION=$(cat VERSION) KO_DOCKER_REPO=crlsmrls ko publish -B -t $(cat VERSION) -t latest . 
VERSION=$(cat VERSION) KO_DOCKER_REPO=ko.local ko publish -B -t $(cat VERSION) -t latest . 
```

