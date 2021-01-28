FROM swr.ap-southeast-1.myhuaweicloud.com/opensourceway/prow/bazel2-env:latest as builder
LABEL maintainer="Xie Weizhi" description="The prow module shares image"
ARG BPATH=prow/cmd/gitee-hook
WORKDIR test-infra
COPY . .
RUN bazel build ${BPATH}

FROM alpine:latest
ARG BPATH=prow/cmd/gitee-hook
ARG BMOUDLE=gitee-hook
ARG BPORT=8080
WORKDIR /root/
COPY --from=builder /usr/local/test-infra/bazel-bin/${BPATH}/linux_amd64_pure_stripped/${BMOUDLE} ./app
ENTRYPOINT ["./app"]
EXPOSE ${BPORT}