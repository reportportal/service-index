FROM --platform=${BUILDPLATFORM} golang:1.19.1-alpine AS builder

ENV APP_DIR=/go/src/github.com/org/repo

ARG BUILDPLATFORM TARGETOS TARGETARCH
ARG APP_VERSION=develop
ARG PACKAGE_COMMONS=github.com/reportportal/commons-go/v5
ARG REPO_NAME=reportportal/service-index
ARG BUILD_BRANCH
ARG BUILD_DATE

ADD . ${APP_DIR}
WORKDIR ${APP_DIR}

RUN echo "I am running on $BUILDPLATFORM, building for TargetOS: $TARGETOS and Targetarch: $TARGETARCH"

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -ldflags "-extldflags '"-static"' \
        -X ${PACKAGE_COMMONS}/commons.repo=${REPO_NAME} \
        -X ${PACKAGE_COMMONS}/commons.branch=${BUILD_BRANCH} \
        -X ${PACKAGE_COMMONS}/commons.buildDate=${BUILD_DATE} \
        -X ${PACKAGE_COMMONS}/commons.version=${APP_VERSION}" \
        -o app ./

FROM --platform=$BUILDPLATFORM alpine:3.16.2
WORKDIR /root/

ENV APP_DIR=/go/src/github.com/org/repo
ARG APP_VERSION

LABEL maintainer="Andrei Varabyeu <andrei_varabyeu@epam.com>"
LABEL version=${APP_VERSION}

RUN apk --no-cache add --upgrade apk-tools
COPY --from=builder ${APP_DIR}/app .

EXPOSE 8080
ENTRYPOINT ["./app"]
