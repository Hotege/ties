FROM ubuntu:26.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        wget \
        g++ \
        libboost-filesystem1.90-dev \
    && rm -rf /var/lib/apt/lists/*

ENV GOROOT=/usr/local/go
ENV PATH=$GOROOT/bin:$PATH
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

RUN wget -q https://go.dev/dl/go1.27.3.linux-amd64.tar.gz -O /tmp/go.tgz && \
    tar -C /usr/local -xzf /tmp/go.tgz && \
    rm /tmp/go.tgz && \
    go version

WORKDIR /app
COPY . /app

RUN chmod +x ./projects/build.sh ./projects/run.sh && ./projects/build.sh

RUN ldd libs/libaddons.so libs/hello.tie outputs/ties || true


FROM ubuntu:26.04 AS runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        libboost-filesystem1.90.0 \
        libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/libs    ./libs
COPY --from=builder /app/outputs ./outputs
COPY --from=builder /app/projects/run.sh  ./projects/run.sh

CMD ["./projects/run.sh"]