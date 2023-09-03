FROM golang:1.20 AS build

WORKDIR /src  

COPY go.mod ./
COPY go.sum ./

RUN go mod download && go mod verify

COPY *.go ./

RUN go build -o /samplebox

FROM gcr.io/distroless/base-debian12
# ARG VERSION
# ENV VERSION $VERSION

COPY --from=build /samplebox /samplebox

ENTRYPOINT ["/samplebox"]