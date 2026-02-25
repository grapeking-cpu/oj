# C++17 Runner 镜像
FROM alpine:3.18

LABEL maintainer="OJ Platform"
LABEL language="cpp17"

# 安装编译工具
RUN apk add --no-cache \
    g++=13.2.1-r3 \
    libc-dev=0.7.2-r3 \
    make=4.4.1-r1

# 创建工作目录
WORKDIR /app

# 创建非root用户
RUN adduser -D -s /bin/sh runner

# 设置权限
RUN chown -R runner:runner /app

# 切换用户
USER runner

# 暴露端口
EXPOSE 8080

# 默认命令
CMD ["sh"]
