FROM alpine:3.19

RUN apk add --no-cache i2pd
RUN apk add --no-cache rsync
EXPOSE 7070

CMD ["i2pd", "--loglevel=debug", "--conf=/var/lib/i2pd/i2pd.conf"]
CMD ["sleep", "10m"]