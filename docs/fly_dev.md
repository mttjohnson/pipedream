# More crazy experiments testing this with a personal Fly.io account

TODO:

building containers
```bash
fly_username="xxxxx_fly_username_xxxxx" # not sure how to programatically fetch this
random6="$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 6)"

# "registry.fly.io/xxxxx_app_name_xxxxx:xxxxx_image_name_xxxxx"
# Seems like the app name may need to be unique, or maybe it's supposed to be the
# account username ¯\_(ツ)_/¯

VARNISH_APP_NAME="varnish"
VARNISH_IMAGE="registry.fly.io/${fly_username}-${VARNISH_APP_NAME}:$(date +'%F.%H-%M-%S')"
rg FROM Dockerfile | awk '{ print $2 }'
docker buildx build -f varnish.Dockerfile . --tag "${VARNISH_IMAGE}"

NGINX_APP_NAME="nginx"
NGINX_IMAGE="registry.fly.io/${fly_username}-${NGINX_APP_NAME}:$(date +'%F.%H-%M-%S')"
rg FROM nginx.Dockerfile | awk '{ print $2 }'
docker buildx build -f nginx.Dockerfile . --tag "${NGINX_IMAGE}"

echo "
fly_username=${fly_username}
random6=${random6}
VARNISH_APP_NAME=${VARNISH_APP_NAME}
VARNISH_IMAGE=${VARNISH_IMAGE}
NGINX_APP_NAME=${NGINX_APP_NAME}
NGINX_IMAGE=${NGINX_IMAGE}
"
```

Misc
```bash
fly apps list

flyctl auth docker && docker push ${VARNISH_IMAGE} && flyctl deploy --ha=false --image ${VARNISH_IMAGE}
flyctl auth docker && docker push ${NGINX_IMAGE} && flyctl deploy --ha=false --image ${NGINX_IMAGE}
flyctl machines list

flyctl deploy -i registry.fly.io/your-app:something

flyctl scale count 0
./run http-detailed https://pipedream.changelog.com/
./run http-measure https://pipedream.changelog.com/
./run http-profile https://pipedream.changelog.com/
```
