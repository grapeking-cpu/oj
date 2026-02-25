# Go 1.22 Runner 镜像
FROM alpine:3.18

LABEL maintainer="OJ Platform"
LABEL language="go122"

# 安装 Go
RUN apk add --no-cache go=1.22.1-r0

# 创建工作目录
WORKDIR /app

RUN adduser -D -s /bin/sh runner
RUN chown -R runner:runner /app

USER runner

EXPOSE 8080

CMD ["sh"]
