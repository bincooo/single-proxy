### 简单代理服务器

仿 `nginx` 代理服务，支持本地代理、ja3、 websocket

#### 编译
```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o server -trimpath
```

#### 配置
`config.yaml` 配置
```yaml
#本地代理
proxies: http://127.0.0.1:7890
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