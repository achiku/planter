FROM golang:alpine

COPY . /src/
RUN cd /src/ && go get .
ENTRYPOINT ["/go/bin/planter"]
