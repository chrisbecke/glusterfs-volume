FROM ubuntu:22.04 AS ubuntu-base

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update 
RUN apt-get install -y glusterfs-client ca-certificates tini

FROM ubuntu-base AS ubuntu-builder

RUN apt-get install -y golang-go pkg-config libglusterfs-dev uuid-dev ca-certificates

FROM fedora:38 AS fedora-base

RUN dnf update -y
RUN dnf install -y glusterfs-client tini

FROM fedora-base AS fedora-builder

RUN dnf install -y pkg-config glusterfs-api-devel golang

FROM ubuntu-builder AS builder
FROM ubuntu-base AS base

FROM builder AS build

WORKDIR /src
COPY src .
RUN go build

FROM base AS plugin

COPY --from=build /src/glusterfs-plugin /usr/local/bin/glusterfs-plugin
