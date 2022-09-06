FROM node:16.13.2-alpine3.15 as build-env

ENV CGO_ENABLED=0
# Important env variables: DB_URL PORT GO_ENV GRAPHQL_URI
RUN apk add gcompat
COPY --from=golang:1.18-alpine /usr/local/go/ /usr/local/go/
 
ENV PATH="/usr/local/go/bin:${PATH}"


RUN mkdir /app
# do this first to improve layer catching with go mod (save us ~200s on development cycle)
COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download

## test+compile the web app
COPY web/ /app/web/
WORKDIR /app/web
## RUN npm run test ## FIXME: add tests to the frontend
RUN npm run build

## BENCHMARK between having go:embed and files in docker container.
WORKDIR /app
COPY *.go /app/
COPY ./lib /app/lib/
#COPY ./config /app/config/
RUN go test
RUN GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' --installsuffix cgo -mod=readonly -o nftlink


FROM scratch
COPY --from=build-env /app/nftlink /nftlink
## Certificate are needed for https to sites like magic.link or infura
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /
ENTRYPOINT ["/nftlink"]
