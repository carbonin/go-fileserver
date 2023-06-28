FROM registry.ci.openshift.org/openshift/release:golang-1.16 as builder

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOFLAGS="" GO111MODULE=on go build -o /test-service main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5

ARG DATA_DIR=/data
RUN mkdir $DATA_DIR && chmod 775 $DATA_DIR
VOLUME $DATA_DIR
ENV DATA_DIR=$DATA_DIR

COPY --from=builder /test-service /test-service
CMD ["/test-service"]
