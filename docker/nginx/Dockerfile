# Nginx Dockerfile
# 用于反向代理和静态文件服务

FROM nginx:alpine

# 复制配置文件
COPY nginx.conf /etc/nginx/nginx.conf

# 创建日志目录
RUN mkdir -p /var/log/nginx

# 暴露端口
EXPOSE 80 443

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

# 启动命令
CMD ["nginx", "-g", "daemon off;"]
