FROM golang:1.13 AS builder
COPY . /build
WORKDIR /build
RUN go mod download
RUN CGO_ENABLED=0 go build

FROM hashicorp/terraform:latest
COPY --from=builder /build/terraform-provider-zabbix /root/.terraform.d/plugins/linux_amd64/terraform-provider-zabbix
