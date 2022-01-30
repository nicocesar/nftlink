FROM node:17.4.0-alpine3.15 as build-env

ENV CGO_ENABLED=0
# Important env variables: DB_URL PORT GO_ENV GRAPHQL_URI
RUN apk add gcompat
COPY --from=golang:1.17-alpine /usr/local/go/ /usr/local/go/
 
ENV PATH="/usr/local/go/bin:${PATH}"


RUN mkdir /app
# do this first to improve layer catching with go mod (save us ~200s on development cycle)
COPY go.mod go.sum /app/ 
WORKDIR /app
RUN go mod download

## test+compile the web app
COPY web/ /app/web/
WORKDIR /app/web
RUN yarn install
## yarn needs --watchAll=false otherwise it gets stuck with watchman for some reason with docker
#RUN yarn test --watchAll=false
RUN yarn build 

## BENCHMARK between having go:embed and files in docker container.
WORKDIR /app
COPY *.go /app/
COPY ./lib /app/lib/
#COPY ./config /app/config/
#RUN go test
RUN GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' --installsuffix cgo -mod=readonly -o nftlink


FROM scratch
COPY --from=build-env /app/nftlink /nftlink
#COPY --from=build-env /app/config /config/
## TODO: BENCHMARK between having go:embed and files in docker container.
COPY --from=build-env /app/web/build/ /web/build/ 
## Certificate are needed for https to magic.link
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /
ENTRYPOINT ["/nftlink"]
