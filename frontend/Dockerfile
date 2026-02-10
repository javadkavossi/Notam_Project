# اگر node در دسترس نیست، اول محلی اجرا کن: cd frontend && npm run build
FROM nginx:alpine
COPY dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
