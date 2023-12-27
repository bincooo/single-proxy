FROM node:19.1.0-alpine3.16
#FROM debian:bullseye-slim
#RUN unlink /etc/localtime && ln -s /usr/share/zoneinfo/Etc/GMT-8 /etc/localtime
ADD . /app
WORKDIR /app
ADD .env .env
RUN chmod +x "./server"
CMD ["./server"]