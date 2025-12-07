@echo off
REM Payment Gateway Setup Script for Windows

echo 🚀 Setting up Payment Gateway...

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Go is not installed. Please install Go 1.23 or higher.
    exit /b 1
)

echo ✅ Go is installed
go version

REM Create .env file if it doesn't exist
if not exist .env (
    echo 📝 Creating .env file from .env.example...
    copy .env.example .env
    echo ⚠️  Please update .env with your configuration
) else (
    echo ✅ .env file already exists
)

REM Download dependencies
echo 📦 Downloading dependencies...
go mod download
go mod tidy

REM Build the application
echo 🔨 Building application...
go build -o payment-gateway.exe main.go

echo.
echo ✅ Setup complete!
echo.
echo To run the application:
echo   payment-gateway.exe
echo.
echo Or:
echo   go run main.go
echo.

pause

