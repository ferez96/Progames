FROM python:alpine

WORKDIR /build
COPY . .

RUN /build/scripts/build.sh


CMD progames --help