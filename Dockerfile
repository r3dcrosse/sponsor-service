FROM golang:1.15-alpine
RUN mkdir /sponsor-service
ADD . /sponsor-service
WORKDIR /sponsor-service

RUN go mod download
RUN go build -o main .

EXPOSE 8000

ENV RABBITMQ_IP "localhost:5672"
ENV PG_IP "localhost"
ENV PG_PORT "5432"
ENV PG_USER "user"
ENV PG_PASS "hey"
ENV PG_DB_NAME "postgres"
ENV PG_SSL "disable"

CMD ["sh", "-c", "/sponsor-service/main -rabbit=${RABBITMQ_IP} -pg_ip=${PG_IP} -pg_port=${PG_PORT} -pg_user=${PG_USER} -pg_password=${PG_PASS} -pg_dbname=${PG_DB_NAME} -pg_ssl=${PG_SSL}"]