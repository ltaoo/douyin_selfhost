# Douyin Server

这是一个为 [zyronon/douyin](https://github.com/zyronon/douyin) 前端项目设计的后端服务。它允许你在本地托管该应用并播放你自己的视频文件。

## 功能特性

- **本地视频托管**：扫描本地目录中的视频文件（`.mp4`, `.webm`, `.ogg`）并通过 API 提供服务。
- **前端托管**：将 Vue3 前端应用作为单页应用（SPA）进行托管。
- **API 模拟**：实现了必要的 API 接口以支持前端功能（如用户面板、推荐视频等）。
- **Docker 支持**：使用 Docker 轻松部署，自动构建前端并设置后端环境。

## 前置条件

- **视频文件**：包含你想要展示的视频文件的目录。
- **Docker**：推荐使用 Docker 进行最简单的设置。
- **Go 1.23+**：（如果不使用 Docker 并在本地运行后端）。
- **Node.js & pnpm**：（如果需要手动构建前端）。

## Docker 快速开始

提供的 `Dockerfile` 会处理前端和后端的构建。

1. **构建镜像**

   ```bash
   docker build -t douyin-server .
   ```

2. **运行容器**

   将 `/path/to/your/videos` 替换为你本地视频目录的绝对路径。

   ```bash
   docker run -d \
     -p 8080:8080 \
     -v /path/to/your/videos:/app/media \
     douyin-server
   ```

3. **访问应用**

   在浏览器中打开 [http://localhost:8080](http://localhost:8080)。

## 本地运行（手动设置）

如果你不想使用 Docker，可以按照以下步骤操作：

### 1. 构建前端

克隆前端仓库并进行构建：

```bash
git clone https://github.com/zyronon/douyin.git
cd douyin
pnpm install
pnpm build
```

将生成的 `dist` 文件夹复制到本 `douyin_server` 项目的根目录下。

### 2. 准备数据文件

后端需要一些特定的数据文件。

- 确保 `dist/data` 存在（这应该是前端构建的一部分）。
- 在 `douyin_server` 根目录下创建目录结构 `src/assets/data/`。
- 将前端源码中的 `posts6.json`（路径 `src/assets/data/posts6.json`）复制到 `douyin_server/src/assets/data/posts6.json`。

### 3. 运行服务器

```bash
# 默认假设视频文件在当前目录下的 'media' 文件夹中
go run main.go --static ./dist --media /path/to/your/videos
```

### 命令行参数

- `--static`：静态文件目录路径（默认："dist"）。
- `--index`：索引文件路径（默认："index.html"）。
- `--media`：包含视频的媒体目录路径（默认："media"）。

## API 接口

服务器实现了以下接口以支持前端：

- `/video/recommended`：返回视频列表（本地视频 + 模拟数据）。
- `/media/*`：提供实际的视频文件流。
- `/user/*`：用户相关接口（面板、收藏、朋友等）。
- `/post/recommended`：推荐帖子。
- `/shop/recommended`：推荐商品。
- `/music`：音乐列表。

## 许可证

MIT
