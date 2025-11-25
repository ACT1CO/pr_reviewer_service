@echo off
echo Building and starting services...
docker compose up --build -d

echo Running E2E tests...
cd ..\..
go test -v ./tests/e2e

echo Stopping services...
docker compose down -v

pause