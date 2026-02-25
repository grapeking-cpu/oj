# Python 3.11 Runner 镜像
FROM python:3.11-slim

LABEL maintainer="OJ Platform"
LABEL language="py311"

# 安装必要包
RUN pip install --no-cache-dir --break-system-packages \
    numpy==1.26.0 \
    scipy==1.11.0

# 创建工作用户
RUN useradd -m runner

WORKDIR /home/runner

USER runner

EXPOSE 8080

CMD ["python3"]
