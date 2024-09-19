FROM golang:1.22.6-alpine AS builder

ENV APP_DIR=/go/src/github.com/org/repo

ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG APP_VERSION=develop
ARG PACKAGE_COMMONS=github.com/reportportal/commons-go/v5
ARG REPO_NAME=reportportal/service-index
ARG BUILD_BRANCH
ARG BUILD_DATE

ADD . ${APP_DIR}
WORKDIR ${APP_DIR}

RUN echo "I am running on ${BUILDPLATFORM}, building for TargetOS: $TARGETOS and Targetarch: ${TARGETARCH}"

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -ldflags "-extldflags '"-static"' \
        -X ${PACKAGE_COMMONS}/commons.repo=${REPO_NAME} \
        -X ${PACKAGE_COMMONS}/commons.branch=${BUILD_BRANCH} \
        -X ${PACKAGE_COMMONS}/commons.buildDate=${BUILD_DATE} \
        -X ${PACKAGE_COMMONS}/commons.version=${APP_VERSION}" \
        -o app ./

FROM alpine:3.20.2
ENV DEPOLY_DIR=/app/service-index
RUN mkdir -p ${DEPOLY_DIR}
WORKDIR ${DEPOLY_DIR}

RUN chgrp -R 0 ${DEPOLY_DIR} && chmod -R g=u ${DEPOLY_DIR}

ENV APP_DIR=/go/src/github.com/org/repo
ARG APP_VERSION

LABEL maintainer="Andrei Varabyeu <andrei_varabyeu@epam.com>"
LABEL version=${APP_VERSION}

RUN apk --no-cache add --upgrade apk-tools
COPY --from=builder ${APP_DIR}/app .

EXPOSE 8080
ENTRYPOINT ["./app"]
