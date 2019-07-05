# Thanks to https://blog.codeship.com/building-minimal-docker-containers-for-go-applications/
FROM scratch
ADD xlsx2json-api /
ENV DEBUG=1
ENV API_HOST=0.0.0.0
EXPOSE 8000/tcp
CMD ["/xlsx2json-api"]