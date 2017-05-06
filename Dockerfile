FROM scratch

MAINTAINER Andrei Varabyeu <andrei_varabyeu@epam.com>

ADD ./bin/service-index /

EXPOSE 8080
ENTRYPOINT ["/service-index"]
