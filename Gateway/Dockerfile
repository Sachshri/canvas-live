FROM nginx:latest

# Remove the default Nginx configuration
RUN rm /etc/nginx/conf.d/default.conf

# Copy your custom Nginx configuration file into the container

COPY nginx.conf /etc/nginx/nginx.conf

# Create a directory for SSL certificates if you're using HTTPS

RUN mkdir -p /etc/nginx/ssl

# If you have SSL certificates, copy them into the container
# COPY ssl/api.example.com.crt /etc/nginx/ssl/
# COPY ssl/api.example.com.key /etc/nginx/ssl/

# Expose the HTTP and HTTPS ports
EXPOSE 80
EXPOSE 443
