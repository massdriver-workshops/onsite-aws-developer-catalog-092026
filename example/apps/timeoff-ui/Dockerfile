FROM nginxinc/nginx-unprivileged:1.27-alpine
ARG VERSION=dev
ENV UI_VERSION=${VERSION}
COPY app/ /app/
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY nginx/40-render-site.sh /docker-entrypoint.d/40-render-site.sh
EXPOSE 8080
