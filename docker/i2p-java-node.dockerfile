FROM eclipse-temurin:17-jre-alpine

# Install I2P package
RUN apk add --no-cache i2p

# Expose default ports
EXPOSE 7657 # Web console
EXPOSE 7654 # I2CP

CMD ["i2prouter", "start"]
