# Java 17 Runner 镜像
FROM eclipse-temurin:17-jre-alpine

LABEL maintainer="OJ Platform"
LABEL language="java17"

RUN addgroup -S runner && adduser -S runner -G runner

WORKDIR /home/runner

USER runner

EXPOSE 8080

CMD ["java"]
