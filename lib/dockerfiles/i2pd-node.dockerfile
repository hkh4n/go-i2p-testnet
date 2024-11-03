FROM alpine:3.19

RUN apk add --no-cache i2pd

EXPOSE 7070

CMD ["i2pd"]
