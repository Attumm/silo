FROM golang:alpine as builder
RUN mkdir /app 
ADD *.go /app/
WORKDIR /app 
RUN go build -o main .


FROM alpine:3.10
RUN adduser -S -D -H -h /app appuser
USER appuser
WORKDIR /app 
COPY --from=builder /app/main ./
COPY index.html ./
ENTRYPOINT ["/app/main"]
EXPOSE 8000
