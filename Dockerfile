FROM alpine:3.16.2

LABEL maintainer="Andrei Varabyeu <andrei_varabyeu@epam.com>"
LABEL version=5.1.12-BETA-2

ENV APP_DOWNLOAD_URL https://github.com/reportportal/service-index/releases/download/v5.1.12-BETA-2/service-index_linux_amd64
RUN apk --no-cache add --upgrade apk-tools

ADD ${APP_DOWNLOAD_URL} /service-index

RUN chmod +x /service-index

EXPOSE 8080
ENTRYPOINT ["/service-index"]
