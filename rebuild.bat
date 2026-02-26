@echo off
chcp 65001 >nul
echo ========================================
echo   重建 OJ 前端和后端服务
echo ========================================
echo.

echo [1/3] 停止并删除相关容器...
docker compose stop api judge-light-1 judge-light-2 judge-heavy-1 frontend
docker compose rm -f api judge-light-1 judge-light-2 judge-heavy-1 frontend
echo.

echo [2/3] 重新构建镜像（带缓存）...
docker compose build --no-cache api judge-light-1 judge-light-2 judge-heavy-1 frontend
echo.

echo [3/3] 启动所有服务...
docker compose up -d
echo.

echo ========================================
echo   重建完成！
echo ========================================
echo.
docker compose ps
