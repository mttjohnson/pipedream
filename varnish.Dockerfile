# https://hub.docker.com/_/varnish
FROM varnish:7.4.3
ENV VARNISH_HTTP_PORT=9000
COPY default.vcl /etc/varnish/default.vcl

# Hack up a local docker container to fake DNS to run locally
USER root
RUN apt update && apt -y install curl net-tools vim iproute2 dnsutils procps iputils-ping
RUN apt -y install dnsmasq
RUN echo 'listen-address=127.0.0.1' >> /etc/dnsmasq.conf
RUN echo 'user=root' >> /etc/dnsmasq.conf
RUN echo 'server=8.8.8.8' >> /etc/dnsmasq.conf
RUN echo 'address=/changelog-2024-01-12.internal/172.18.0.2'  >> /etc/dnsmasq.conf
RUN echo 'address=/changelog-2024-01-12.internal/fd20:b007:398e::2'  >> /etc/dnsmasq.conf
RUN dnsmasq --test
USER varnish
# ENTRYPOINT ["sh", "-c", "/etc/init.d/dnsmasq systemd-exec && /usr/local/bin/docker-varnish-entrypoint"]
