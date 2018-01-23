FROM nginx

ADD https://github.com/kvhnuke/etherwallet/releases/download/v3.11.3.0/etherwallet-v3.11.3.0.zip /mew.zip
RUN apt update && apt install unzip && unzip /mew.zip -d / && \
    mkdir -p /usr/share/nginx/html && \
    mv /etherwallet-v3.11.3.0/* /usr/share/nginx/html
