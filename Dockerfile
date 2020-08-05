FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY ./src ./src

WORKDIR /app/src/cmd/imager

RUN go build -o imager .

EXPOSE 8080

CMD ["./imager"]
