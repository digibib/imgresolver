FROM alpine
ADD imgresolver /imgresolver
CMD ["/imgresolver", "--esAdr", "elasticsearch:9200"]
