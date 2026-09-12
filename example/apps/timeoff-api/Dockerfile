FROM golang:1.25-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/timeoff-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/timeoff-api /timeoff-api
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/timeoff-api"]
