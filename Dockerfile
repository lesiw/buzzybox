FROM golang:1.22
WORKDIR /work
COPY . .
RUN go build .

FROM scratch
COPY --from=0 /work/buzzybox /buzzybox
ENTRYPOINT ["/buzzybox"]
