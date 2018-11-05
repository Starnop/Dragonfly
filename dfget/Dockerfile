FROM golang:1.10

WORKDIR /go/src/github.com/alibaba

# install dfget
RUN git clone https://github.com/alibaba/Dragonfly.git \
    && cd Dragonfly && ./build/build.sh client

# set the directory of df-client to $PATH environment variable
ENV PATH=$GOPATH/bin:/root/.dragonfly/df-client:$PATH