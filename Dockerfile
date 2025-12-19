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

# Install dependencies and build
RUN pnpm install
RUN pnpm build

# Stage 2: Build Backend
FROM golang:1.23-alpine AS backend-builder

WORKDIR /backend

# Copy source code
COPY go.mod main.go ./

# Ensure dependencies are tidy (since go.sum might be missing)
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o douyin main.go

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
