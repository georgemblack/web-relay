FROM golang:1.15 as build-env
WORKDIR /go/src/app
ADD . /go/src/app
RUN go build ./main.go

FROM gcr.io/distroless/base-debian10
WORKDIR /app
COPY --from=build-env /go/src/app/main ./main
CMD ["/app/main"]

EXPOSE 8080