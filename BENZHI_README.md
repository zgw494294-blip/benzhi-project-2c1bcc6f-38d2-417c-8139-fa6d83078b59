# BENZHI_README

基于 Go 实现的典藏扫描品发布核准台 Web 项目，一款后端服务，档案管理员为扫描档案建立质检证据、整改问题并经复核后发布。

## 项目说明
- 项目：benzhi-project-2c1bcc6f-38d2-417c-8139-fa6d83078b59
- 项目用途：档案管理员为扫描档案建立质检证据、整改问题并经复核后发布。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run . -addr=127.0.0.1:19081 -selfcheck
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-2c1bcc6f-38d2-417c-8139-fa6d83078b59-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-2c1bcc6f-38d2-417c-8139-fa6d83078b59-arm64 linux/arm64
docker run -it benzhi-project-2c1bcc6f-38d2-417c-8139-fa6d83078b59-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run . -addr=127.0.0.1:19081 -selfcheck`
