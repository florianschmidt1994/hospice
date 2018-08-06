FROM golang:onbuild

RUN go get github.com/dyatlov/go-htmlinfo/htmlinfo
RUN go get github.com/gorilla/feeds
RUN go get github.com/mattn/go-sqlite3

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN go build -o main .

CMD ["./main"]

