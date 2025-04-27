FROM golang:1.23.1-bullseye AS builder
WORKDIR /movieSpider
ENV GOPROXY https://goproxy.cn,direct
COPY . .
RUN CGO_ENABLED=0 go build -o /app/image_guard

FROM  hairyhenderson/upx AS upx
COPY --from=builder /app/image_guard /app/image_guard
RUN upx /app/image_guard

FROM frolvlad/alpine-glibc
MAINTAINER YCD "hnyoucd@gmail.com"
WORKDIR /app
ENV PATH=/app:$PATH
ENV TZ Asia/Shanghai
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories &&\
    apk add -U tzdata --no-cache &&\
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime &&\
    echo $TZ > /etc/timezone
COPY --from=upx /app/image_guard /app/image_guard
ENTRYPOINT ["./image_guard"]

