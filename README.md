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
https://complete-mmx-kp80vkki-ai.hf.space=/v1/chat/completions,/v1/models
```