FROM alpine
RUN set -ex \
    && apk add --no-cache \
    ca-certificates
ADD imgresolver /imgresolver
CMD ["/imgresolver", "--es", "http://elasticsearch:9200"]
