FROM alpine:3.11

LABEL maintainer="Andrei Varabyeu <andrei_varabyeu@epam.com>"
LABEL version=5.0.7

ENV APP_DOWNLOAD_URL https://dl.bintray.com/epam/reportportal/5.0.7/service-index_linux_amd64

ADD ${APP_DOWNLOAD_URL} /service-index

RUN chmod +x /service-index

EXPOSE 8080
ENTRYPOINT ["/service-index"]
