FROM scratch

LABEL maintainer="Andrei Varabyeu <andrei_varabyeu@epam.com>"
LABEL version="@version@"
LABEL description="@description@"

ENV APP_DOWNLOAD_URL https://dl.bintray.com/epam/reportportal/com/epam/reportportal/service-index-temp/3.1.1/service-index_linux_amd64

ADD ${APP_DOWNLOAD_URL} /

EXPOSE 8080
ENTRYPOINT ["/service-index"]
