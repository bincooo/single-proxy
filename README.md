### 简单代理服务器

#### 编译
```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o server -trimpath
```

#### 配置
`.env` `PROXY`本地代理，`PORT`服务端口
```
PROXY=http://127.0.0.1:7890
PORT=8444
```
`config.ini` 左值转发地址，右值托管路径，需要写在一行，可每行配置一个转发地址
```
https://api.openai.com=/v1/chat/completions,/v1/models
```

#### vercel
一键部署，点这里 => [![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https://github.com/bincooo/single-proxy&repository-name=single-proxy)

建议Fork到自己的github上，修改自己的`config.ini`文件。并在vercel上的`Environment Variables`配置`CONFIG`

`hugggingface.co`、`vercel` ip封锁，`claude.ai` 基本无法代理