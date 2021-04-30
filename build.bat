@echo off
go build -ldflags "-s -w"
npm i && npm run build
