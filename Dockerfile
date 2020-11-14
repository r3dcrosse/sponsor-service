FROM golang:1.15-alpine
RUN mkdir /sponsor-service
ADD . /sponsor-service
WORKDIR /sponsor-service

RUN go mod download
RUN go build -o main .

CMD ["/sponsor-service/main"]