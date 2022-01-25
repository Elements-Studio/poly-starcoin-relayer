FROM golang:1.16 AS build

ARG GIT_COMMIT
ARG GIT_DIRTY

ENV GIT_COMMIT=$GIT_COMMIT \
    GIT_DIRTY=$GIT_DIRTY \
    goos="linux" \
    goArch="amd64"

ENV BIN_NAME="poly-starcoin-relayer_${goos}_${goArch}"

WORKDIR /poly-starcoin-relayer
COPY ./ .
RUN go install -tags testnet
RUN GOOS=$goos GOARCH=$goArch go build -tags testnet -ldflags "-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY}" -o $BIN_NAME
# TODO: just use testnet for TEST now!!! 
RUN ls -la /poly-starcoin-relayer

FROM golang:1.16

ENV RELEASE_PATH="/poly-starcoin-relayer"

WORKDIR /data/poly-starcoin-relayer
# COPY --from=build /poly-starcoin-relayer/poly-starcoin-relayer_linux_amd64 \
#      /poly-starcoin-relayer/docker/entrypoint.sh \
#      ./
# RUN chmod 755 /data/poly-starcoin-relayer/entrypoint.sh

COPY --from=build /poly-starcoin-relayer/poly-starcoin-relayer_linux_amd64 \
     /poly-starcoin-relayer/config-testnet.json \
     ./
# TODO: just use testnet for TEST now!!! 

#RUN mkdir /data/poly-starcoin-relayer/config
#COPY --from=build /poly-starcoin-relayer/config /data/poly-starcoin-relayer/config

RUN ls -la /data/poly-starcoin-relayer

# ENTRYPOINT ["/data/poly-starcoin-relayer/entrypoint.sh"]
ENTRYPOINT ["/data/poly-starcoin-relayer/poly-starcoin-relayer_linux_amd64", "--cliconfig", "/data/poly-starcoin-relayer/config-testnet.json"]
# TODO: just use testnet for TEST now!!!
