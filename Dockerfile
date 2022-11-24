FROM alpine:3.16 AS build

RUN apk update && apk add go git make

ENV PRJ_DIR /tmp_project

WORKDIR ${PRJ_DIR}

COPY go.mod ${PRJ_DIR}
COPY go.sum ${PRJ_DIR}
RUN go mod download

ARG bin_file

COPY . ${PRJ_DIR}

RUN make container-binary BIN_FILE=${bin_file} MAIN_FILE=${bin_file}.go && \
    cp bin/${bin_file} /bin/${bin_file}


FROM alpine:3.16 as run

ARG bin_file

ENV TO_BIN_FILE ${bin_file}

COPY --from=build /bin/${bin_file} /app/${bin_file}

WORKDIR /app/

RUN chmod -R 777 /app/

ENTRYPOINT /app/${TO_BIN_FILE}

# docker build  --build-arg bin_file=cds-snat-configuration -t cds-snat-configuration .
# docker tag cds-snat-configuration:latest capitalonline/cds-snat-configuration:latest
# docker push capitalonline/cds-snat-configuration:latest