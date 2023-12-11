FROM alpine:3.14

RUN mkdir -p /app
RUN mkdir -p /app/www
WORKDIR /app

ADD ./dist ./www/dist
ADD ./runner .

ENTRYPOINT ["./runner"]
