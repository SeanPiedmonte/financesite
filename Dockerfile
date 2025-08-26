# Stage 1: Build UI
FROM node:16 AS build

WORKDIR /app


COPY ui/package*.json ./ui/
RUN cd ui && npm install
COPY ui /app/ui
RUN cd ui && npm run build

# Stage 2: Build Go server
FROM golang:1.20

WORKDIR /app
COPY main.go .
COPY --from=build /app/ui/dist /app/ui/dist
RUN go build -o server main.go

EXPOSE 8080
CMD ["./server"]
