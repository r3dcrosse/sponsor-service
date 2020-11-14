FROM golang:1.15-alpine
RUN mkdir /sponsor-service
ADD . /sponsor-service
WORKDIR /sponsor-service

RUN go mod download
RUN go build -o main .

ENV RABBITMQ_IP "localhost:5672"

CMD ["sh", "-c", "/sponsor-service/main -rabbit=${RABBITMQ_IP}"]