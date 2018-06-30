FROM golang:1.10-alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

RUN apk add --no-cache curl

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

COPY compile_cube.sh .
RUN chmod u=x,g=x,o=x ./compile_cube.sh

CMD ["./compile_cube.sh"]