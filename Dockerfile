FROM alpine:3.14

RUN mkdir -p /app
RUN mkdir -p /app/www
WORKDIR /app

ADD ./templates .

ADD ./dist ./www/dist
ADD ./runner .

RUN chmod +x ./runner

ENTRYPOINT ["./runner"]
