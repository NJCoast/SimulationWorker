FROM golang:1.10
WORKDIR /go/src/github.com/NJCoast/SimulationWorker/
RUN go get k8s.io/client-go/...
RUN go get github.com/gorilla/websocket
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o SimulationWorker .

FROM 234514569215.dkr.ecr.us-east-1.amazonaws.com/model:1.24
WORKDIR /root/
COPY convert.js .
COPY package.json .
COPY package-lock.json .
RUN apt-get update && apt-get install -y curl && curl -sL https://deb.nodesource.com/setup_10.x | bash - &&  apt-get install -y nodejs && npm ci 
COPY --from=0 /go/src/github.com/NJCoast/SimulationWorker .
CMD ["./SimulationWorker"] 