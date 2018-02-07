FROM alpine
ADD imgresolver /imgresolver
CMD ["/imgresolver", "--es", "http://elasticsearch:9200"]
