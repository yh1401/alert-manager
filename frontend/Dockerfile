# Stage 1: Build the Vue.js application
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package.json and package-lock.json to leverage Docker cache
COPY package.json package-lock.json ./

# Install dependencies
RUN npm install

# Copy the rest of the application source code
COPY . .

# Build the application for production
RUN npm run build

# Stage 2: Serve the application with Nginx
FROM nginx:1.25-alpine

# Remove default nginx server configuration
RUN rm /etc/nginx/conf.d/default.conf

# Copy the custom nginx configuration file created previously
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy the built static assets from the builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose port 80 for the Nginx server
EXPOSE 80

# Start Nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]
