FROM alpine:3.6

RUN mkdir -p /home/app
WORKDIR /home/app

RUN addgroup -S appuser
RUN adduser -D -S -s /sbin/nologin -G appuser appuser

STOPSIGNAL SIGTERM


CMD ["./cube"]