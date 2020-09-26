### 实现连续90天微信支付的网页服务
1. 配置文件是application.ini 将application.ini.example改名即可
2. index.go中的MP_verify_6mlClfsaZU0IuAW9.txt是微信的域名校验文件。按照自己的情况设置
3. 附上NGINX反向代理的配置
```
upstream your.domain {
    # server 要代理到的服务器节点，weight是轮询的权重
    server 127.0.0.1:8090 weight=1;
}
server{
        listen 80;
        server_name your.domain;
        location /letusgo {
                proxy_set_header X-Forward-For $remote_addr;
                proxy_set_header X-real-ip $remote_addr;
                proxy_pass your.domain;
        }
        // 校验文件映射
        location /MP_verify_6mlClfsaZU0IuAW9.txt {
                proxy_pass your.domain;
        }
}
```