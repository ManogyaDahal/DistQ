FROM node:20-alpine AS builder

WORKDIR /app

# Install pnpm
RUN npm install -g pnpm

# Copy package files and install dependencies
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install

# Copy all source files and build
COPY . .
RUN pnpm run build

# Final stage: Nginx
FROM nginx:alpine

# Copy built frontend assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy custom nginx configuration for routing and API proxy
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
