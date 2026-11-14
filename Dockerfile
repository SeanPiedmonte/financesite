# Stage 1: build the UI
FROM node:22 AS ui
WORKDIR /app/ui
COPY ui/package*.json ./
RUN npm ci
COPY ui ./
RUN npm run build

# Stage 2: build the Go server
FROM golang:1.25 AS server
WORKDIR /app
COPY go.mod ./
COPY cmd ./cmd
COPY api ./api
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /financesite ./cmd/financesite

# Stage 3: runtime
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=server /financesite ./financesite
COPY --from=ui /app/ui/dist ./ui/dist
# Listen on all interfaces inside the container; publish to localhost only:
#   docker run -p 127.0.0.1:8080:8080 -v financesite-data:/app/data/local financesite
ENV FINANCESITE_ADDR=0.0.0.0:8080
EXPOSE 8080
CMD ["./financesite"]
