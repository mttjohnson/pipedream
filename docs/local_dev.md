# Wild experiments testing this locally with Docker

This is somewhat of a continuation of my "[Digital Resin Experimentation](https://github.com/mttjohnson/changelog.com/blob/james-and-gerhard-build-jerods-pipedream-adam-helps/fly.io/cdn-2024-01-26/VARNISH_TESTING.md)" but hopefully a bit more complete and repeatable for others to also be able to use.

Here I wanted to try and run everything locally and be able to run the hurl test suite against all the local stuff. I got it all working for the most part with a slight amount of what feels like dark magic dns manipulation, though there are still a few (3) assertions failing on the `test/admin.hurl` portion.

## Setup Requirements

This stuff all expects IPv6 to be there and just work (even if your local network is all IPv4).

You might not have an IPv6 network for Docker to use, and if not you would need to create one
```bash
docker network ls
docker network create --ipv6 ip6net
--network ip6net
```

## Build containers

Build some containers using Docker for experimenting with locally
```bash
NGINX_APP_NAME="nginx"
NGINX_IMAGE="${NGINX_APP_NAME}:$(date +'%F.%H-%M-%S')"
rg FROM nginx.Dockerfile | awk '{ print $2 }'
docker buildx build -f nginx.Dockerfile . --tag "${NGINX_IMAGE}" --tag "${NGINX_APP_NAME}:latest"

VARNISH_APP_NAME="varnish"
VARNISH_IMAGE="${VARNISH_APP_NAME}:$(date +'%F.%H-%M-%S')"
rg FROM Dockerfile | awk '{ print $2 }'
docker buildx build -f varnish.Dockerfile . --tag "${VARNISH_IMAGE}" --tag "${VARNISH_APP_NAME}:latest"

echo "
NGINX_APP_NAME=${NGINX_APP_NAME}
NGINX_IMAGE=${NGINX_IMAGE}
VARNISH_APP_NAME=${VARNISH_APP_NAME}
VARNISH_IMAGE=${VARNISH_IMAGE}
"
```

## Run containers

Run each of the containers in separate shells to watch log output
```bash

# zsh command to ignore comments
setopt INTERACTIVE_COMMENTS

# Start up nginx connecting it to the ipv6 brdige
(docker container stop nginx || true) && (docker container rm nginx || true) && docker run -d --network ip6net -p 80:80 -p 443:443 -p 4000:4000 -v ${PWD}/nginx.conf.template:/etc/nginx/templates/default.conf.template --name nginx nginx:latest

# Get the nginx container IPv4 and IPv6 addresses
nginx_container_ip4=$(docker container inspect nginx | jq -r '.[0].NetworkSettings.Networks.ip6net.IPAddress')
nginx_container_ip6=$(docker container inspect nginx | jq -r '.[0].NetworkSettings.Networks.ip6net.GlobalIPv6Address')
inject_host_entry_ip4="changelog-2024-01-12.fly.dev:${nginx_container_ip4}"
inject_host_entry_ip6="changelog-2024-01-12.fly.dev:${nginx_container_ip6}"

# Run varnish container with `--add-host` to inject host entries 
# redirecting backend to nginx container (mock backend)
# also change dns resolver to use localhost (expecting dnsmasq to start following)
# NOTE: turns out the vmod dynamic thing in Varnish doesn't get stuff from /etc/hots 
#       file which is why using dnsmasq ended up being necessary.
(docker container stop varnish || true) && (docker container rm varnish || true) && docker run -d --network ip6net -p 9000:9000 -v ${PWD}/default.vcl:/etc/varnish/default.vcl --add-host "${inject_host_entry_ip4}" --add-host "${inject_host_entry_ip6}" --dns "127.0.0.1" --name varnish varnish:latest
# Start up dnsmasq operating at 127.0.0.1 to override dns resolution
docker exec --user root varnish /bin/bash -c "/etc/init.d/dnsmasq systemd-exec"

# view initial varnish logs and then watch the nginx logs
docker logs varnish
docker logs -f nginx

```

## Interacting with containers

Exec into each of the container to run commands locally
```bash
docker exec -it --user root varnish bash
docker exec -it nginx bash
```

If you need some additional tools inside the container
```bash
apt update && apt -y install curl net-tools vim iproute2 dnsutils procps iputils-ping
```

Useful commands to use inside the Varnish container
```bash
# swap out VCL configs (https://ma.ttias.be/reload-varnish-vcl-without-losing-cache-data/)
TIME=$(date +%s)
varnishadm vcl.load varnish_$TIME /etc/varnish/default.vcl
varnishadm vcl.use varnish_$TIME 

# reload varnish configs
varnishreload

# All the varnish events
varnishlog

varnishadm backend.list
varnishadm vcl.list
varnishadm param.show
varnishadm storage.list

# Monitoring vmod-dynamic with varnishlog
varnishlog -g raw -q '* ~ vmod-dynamic'
```

Check the various nginx endpoints from docker host
```bash
# mock lb to varnish
curl -sv http://localhost:80/
curl -skv https://localhost:443/

# mock backend app
curl -sv http://localhost:4000/
curl -sv http://localhost:4000/podcast/feed
curl -sv http://localhost:4000/admin
curl -sv http://localhost:4000/feed

# varnish
curl -sv http://localhost:9000/

# testing full response (mock lb -> varnish -> mock backend)
curl -skv https://localhost/
curl -skv https://localhost/podcast/feed
curl -skv https://localhost/admin
curl -skv https://localhost/feed

# fake a call to pipedream.changelog.com that is actually routed to the local mock lb to varnish
curl_domain="pipedream.changelog.com"
curl_ip_address="127.0.0.1"
curl_port="443"
curl_proto="https"
curl -sk -o /dev/null -D - --resolve "${curl_domain}:${curl_port}:${curl_ip_address}" "${curl_proto}://${curl_domain}:${curl_port}"
```

## Run the test suite

Run the test suite against the local containers
```bash
# We need to pass the `--insecure` option because the nginx mock lb uses a self-signed cert
hurl --test --color --report-html tmp --insecure --variable host="https://127.0.0.1" test/*.hurl
```



## Example Results

TODO: provide examples of how this worked for me
```
$ hurl --test --color --report-html tmp --insecure --variable host="https://127.0.0.1" test/*.hurl

error: Assert failure
  --> test/admin.hurl:10:0
   |
   | GET {{host}}/admin
   | ...
10 | header "age" == "0" # NOT stored in cache
   |   actual:   string <1213>
   |   expected: string <0>
   |

error: Assert failure
  --> test/admin.hurl:11:0
   |
   | GET {{host}}/admin
   | ...
11 | header "cache-status" contains "hits=0" # double-check that it's NOT stored in cache
   |   actual:   string <Edge; ttl=-1153.423; grace=86400.000; hit; stale; hits=1; region=>
   |   expected: contains string <hits=0>
   |

error: Assert failure
  --> test/admin.hurl:12:0
   |
   | GET {{host}}/admin
   | ...
12 | header "cache-status" contains "miss" # NOT served from cache
   |   actual:   string <Edge; ttl=-1153.423; grace=86400.000; hit; stale; hits=1; region=>
   |   expected: contains string <miss>
   |

test/admin.hurl: Failure (1 request(s) in 36 ms)
test/homepage.hurl: Success (2 request(s) in 38 ms)
test/feed.hurl: Success (3 request(s) in 67049 ms)
--------------------------------------------------------------------------------
Executed files:    3
Executed requests: 6 (0.1/s)
Succeeded files:   2 (66.7%)
Failed files:      1 (33.3%)
Duration:          67052 ms
```

## Troubleshooting and Misc

Troubleshooting requests from inside the varnish container
```bash
# display dnsmasq configs
cat /etc/dnsmasq.conf

# check configs
dnsmasq --test

# look for open ports container is listening on
ss -tulpn | grep 53
netstat -tulpn

# Look at how dnsmasq expects to run with systemd
cat /lib/systemd/system/dnsmasq.service

# Start the dnsmasq process
/etc/init.d/dnsmasq systemd-exec

# query the dnsmasq configured overridden dns entries
# use system configured dns resolver
dig changelog-2024-01-12.internal
dig changelog-2024-01-12.internal aaaa
# send dns queries directly to localhost
dig changelog-2024-01-12.internal @127.0.0.1
dig changelog-2024-01-12.internal aaaa @127.0.0.1
# send dns queries directly to google dns
dig changelog-2024-01-12.internal @8.8.8.8
dig changelog-2024-01-12.internal aaaa @8.8.8.8

# send curl requests directly to nginx mock container
curl -sv -4 http://changelog-2024-01-12.fly.dev:4000/
curl -sv 'http://172.18.0.2/'
curl -sv 'http://172.18.0.2:4000/'
curl -sv -6 http://changelog-2024-01-12.fly.dev:4000/
curl -sv 'http://[fd20:b007:398e::2]/'
curl -sv 'http://[fd20:b007:398e::2]:4000/'

# check running processes
ps aux

# stop dnsmasq
/bin/kill $(cat /run/dnsmasq/dnsmasq.pid)
```

Misc
```bash
# Docker commands
docker ps
docker container list --all
docker image list
docker image prune --all

# Get info from docker about containers
docker container inspect varnish
docker container inspect nginx

echo "docker_host_ip4=$(docker container inspect varnish | jq -r '.[0].NetworkSettings.Networks.ip6net.Gateway')"
echo "docker_host_ip6=$(docker container inspect varnish | jq -r '.[0].NetworkSettings.Networks.ip6net.IPv6Gateway')"
echo "varnish_container_ip4=$(docker container inspect varnish | jq -r '.[0].NetworkSettings.Networks.ip6net.IPAddress')"
echo "varnish_container_ip6=$(docker container inspect varnish | jq -r '.[0].NetworkSettings.Networks.ip6net.GlobalIPv6Address')"

echo "docker_host_ip4=$(docker container inspect nginx | jq -r '.[0].NetworkSettings.Networks.ip6net.Gateway')"
echo "docker_host_ip6=$(docker container inspect nginx | jq -r '.[0].NetworkSettings.Networks.ip6net.IPv6Gateway')"
echo "nginx_container_ip4=$(docker container inspect nginx | jq -r '.[0].NetworkSettings.Networks.ip6net.IPAddress')"
echo "nginx_container_ip6=$(docker container inspect nginx | jq -r '.[0].NetworkSettings.Networks.ip6net.GlobalIPv6Address')"


vi /etc/resolv.conf

apt update && apt -y install dnsmasq
cat /etc/dnsmasq.conf

dnsmasq --test
ss -tulpn | grep 53
netstat -tulpn

# Start dnsmasq
/etc/init.d/dnsmasq checkconfig
/etc/init.d/dnsmasq systemd-exec
/etc/init.d/dnsmasq systemd-start-resolvconf

# Stop
/bin/kill $(cat /run/dnsmasq/dnsmasq.pid)

cat /lib/systemd/system/dnsmasq.service

dig changelog-2024-01-12.internal
dig changelog-2024-01-12.internal aaaa

dig changelog-2024-01-12.internal @127.0.0.1
dig changelog-2024-01-12.internal aaaa @127.0.0.1

```



I still need to read more about:
Hurl https://hurl.dev/
Just https://just.systems/man/en/quick-start.html
Fly https://fly.io/docs/
