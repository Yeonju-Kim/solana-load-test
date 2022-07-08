ARG DOCKER_BASE_IMAGE=klaytn/build_base:latest

FROM ${DOCKER_BASE_IMAGE}

ENV PKG_DIR /locust-docker-pkg
ENV SRC_DIR /go/src/github.com/Yeonju-Kim/solana-load-test
ENV GOPATH /go

RUN mkdir -p $PKG_DIR/bin

ADD . $SRC_DIR

RUN cd $SRC_DIR/solanaslave && go build -ldflags "-linkmode external -extldflags -static"
RUN cp $SRC_DIR/solanaslave/solanaslave $PKG_DIR/bin
