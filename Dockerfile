# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder

# Install git and pnpm
RUN apk add --no-cache git
RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /frontend

# Clone the repository
RUN git clone https://github.com/zyronon/douyin.git .

# Modify src/main.ts to remove mock logic
RUN sed -i '/\/\/放到最后才可以使用pinia/d' src/main.ts && \
    sed -i '/startMock()/d' src/main.ts

# Modify src/config/index.ts to point to local backend (aligned with release.yml)
RUN sed -i "s|baseUrl: 'https://dy.ttentau.top/imgs/'|baseUrl: 'http://127.0.0.1:8080/'|g" src/config/index.ts

# Modify src/mock/index.ts to remove mock adapter (aligned with release.yml)
RUN sed -i '/const mock = new MockAdapter(axiosInstance)/d' src/mock/index.ts

# Install dependencies and build
RUN pnpm install
RUN pnpm build

# Remove dist/images as per release workflow
RUN rm -rf dist/images

# Stage 2: Build Backend
FROM golang:1.23-alpine AS backend-builder

WORKDIR /backend

# Copy source code
COPY go.mod main.go ./

# Ensure dependencies are tidy (since go.sum might be missing)
RUN go mod tidy

# Build the application (aligned with .goreleaser.yaml)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o douyin main.go

# Stage 3: Final Image
FROM alpine:latest

WORKDIR /app

# Install basic certificates
RUN apk --no-cache add ca-certificates

# Copy binary from backend-builder
COPY --from=backend-builder /backend/douyin .

# Copy frontend build from frontend-builder
COPY --from=frontend-builder /frontend/dist ./dist

# Copy the specific file required by the backend at runtime
# main.go:105 reads "src/assets/data/posts6.json"
COPY --from=frontend-builder /frontend/src/assets/data/posts6.json ./src/assets/data/posts6.json

# Create media directory
RUN mkdir -p media

# Declare volume for media
VOLUME /app/media

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./douyin", "--static", "./dist", "--media", "./media"]
