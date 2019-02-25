FROM golang:1.10
WORKDIR /go/src/github.com/NJCoast/SimulationWorker/
RUN go get k8s.io/client-go/...
RUN go get github.com/gorilla/websocket
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o SimulationWorker .

FROM harbor.james-sweet.com/njcoast/model:1.21
WORKDIR /root/
COPY --from=0 /go/src/github.com/NJCoast/SimulationWorker .
CMD ["./SimulationWorker"] 