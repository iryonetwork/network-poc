FROM ubuntu:18.04

RUN apt-get update && apt-get install -y curl libicu60 libusb-1.0-0 libcurl3-gnutls

ENV eos_version=1.7.3 \
    cdt_version=1.6.1 \
    contracts_version=1.6.0

RUN curl -LO https://github.com/EOSIO/eos/releases/download/v${eos_version}/eosio_${eos_version}-1-ubuntu-18.04_amd64.deb \
    && dpkg -i eosio_${eos_version}-1-ubuntu-18.04_amd64.deb

RUN curl -LO https://github.com/EOSIO/eosio.cdt/releases/download/v${cdt_version}/eosio.cdt_${cdt_version}-1_amd64.deb \
    && dpkg -i eosio.cdt_${cdt_version}-1_amd64.deb

RUN curl -LO https://github.com/EOSIO/eosio.cdt/archive/v${cdt_version}.tar.gz && tar -xvzf v${cdt_version}.tar.gz --one-top-level=eosio.cdt --strip-components 1

RUN cd /eosio.cdt/ && curl -LO https://github.com/EOSIO/eosio.contracts/archive/v${contracts_version}.tar.gz && tar -xvzf v${contracts_version}.tar.gz --one-top-level=eosio.contracts --strip-components 1
