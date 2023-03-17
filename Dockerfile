FROM golang:1.18-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o /godocker
EXPOSE 8000
CMD ["/godocker"]