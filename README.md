### 简单代理服务器

仿 `nginx` 代理服务，支持本地代理、ja3、 websocket

默认配置文件已提供了claude镜像代理、ChatGPT API代理、NewBing API代理、gptscopilot代理

#### 编译
```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o server -trimpath
```

#### 配置
`config.yaml` 配置
```yaml
#本地代理
proxies: http://127.0.0.1:7890
#本地代理池请求接口
proxies-pool: http://xxxx.com/get
# 服务端口
port: 8080
# 目标代理地址
mappers:
  - addr: https://xxx.com
    #是否开启ja3
    ja3: true
    routes:
      # 转发路径
      - path: /gpts/proxies/v1/chat/(completions)
        # 重写路径 （支持正则写法）
        rewrite: /api/chat/$1
        # 本地代理 auto为代理池模式，或填写代理地址http://xxx.com
        proxies: auto
        # 前置 request、response 设置器
        action:
          - '{{$var := rGet "Authorization"}}
            {{if contains $var "Bearer "}}
            {{$var = split $var " "}}
            {{$var = index $var 1}}
            {{end}}
            {{$var = append "__sid__=" $var}}
            {{rSet "cookie" $var}}
            {{rSet "origin" "https://gptscopilot.ai"}}
            {{rSet "referer" "https://gptscopilot.ai/gpts"}}
            {{rDel "Authorization"}}'
```
#### vercel
一键部署，点这里 => [![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https://github.com/bincooo/single-proxy&repository-name=single-proxy)

建议Fork到自己的github上，修改自己的`config.yaml`文件。并在vercel上的`Environment Variables`配置`CONFIG`

`hugggingface.co`、`vercel` ip封锁，`claude.ai` 基本无法代理

#### 本地代理池

[monkey-soft/Scrapy_IPProxyPool](https://github.com/monkey-soft/Scrapy_IPProxyPool.git)

[jhao104/proxy_pool](https://github.com/jhao104/proxy_pool.git)
