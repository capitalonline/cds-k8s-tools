FROM golang:1.18 as build
RUN mkdir /app
RUN mkdir /app/bin
COPY . /app/
RUN go env -w GO111MODULE=on
# RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go env

ARG bin_file

WORKDIR /app
RUN go mod tidy
RUN CGO_ENABLED=0 GOARCH="amd64" GOOS="linux" go build -ldflags " -s -w" -o bin/${bin_file}  ./cmd/${bin_file}.go


FROM alpine:3.16 as run

ARG bin_file

ENV TO_BIN_FILE ${bin_file}

COPY --from=build /app/bin/${bin_file} /app/${bin_file}

WORKDIR /app/

RUN chmod -R 777 /app/

ENTRYPOINT /app/${TO_BIN_FILE}

# docker build  --build-arg bin_file=cds-snat-configuration -t cds-snat-configuration .
# docker build --build-arg bin_file=cds-ha-configuration -t cds-ha-configuration .
# docker tag cds-snat-configuration:latest capitalonline/cds-snat-configuration:latest
# docker push capitalonline/cds-snat-configuration:latest