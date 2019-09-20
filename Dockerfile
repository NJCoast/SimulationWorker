FROM golang:1.13.0
WORKDIR /go/src/github.com/NJCoast/SimulationWorker/
RUN mkdir -p $GOPATH/k8s.io/ && cd $GOPATH/k8s.io && git clone https://github.com/kubernetes/klog && cd klog
RUN go get -d k8s.io/client-go/... && cd $GOPATH/k8s.io/klog && git checkout a6a74fbce3a592242b0fc24cd93fd98a4cea0a98 && go install k8s.io/client-go/...
RUN go get github.com/gorilla/websocket
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o SimulationWorker .

FROM 234514569215.dkr.ecr.us-east-1.amazonaws.com/model:1.25
WORKDIR /root/
COPY convert.js .
COPY package.json .
COPY package-lock.json .
RUN apt-get update && apt-get install -y curl && curl -sL https://deb.nodesource.com/setup_10.x | bash - &&  apt-get install -y nodejs && npm ci 
COPY --from=0 /go/src/github.com/NJCoast/SimulationWorker .
CMD ["./SimulationWorker"] 