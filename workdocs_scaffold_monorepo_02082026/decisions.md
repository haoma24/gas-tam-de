# Assumptions & decisions

## HTTP: Chi (not Fiber)

Architecture cho phép “Chi hoặc Fiber”. Chọn Chi vì nhẹ, middleware stack rõ, phù hợp gateway/JWT/rate-limit sau này.

## Go module layout

Một `go.mod` tại root + `pkg/` shared thay vì multi-module `go.work`. Đủ cho MVP 1–2 người; có thể tách module sau nếu CI chậm.

## Flutter platforms

Không commit empty `android/`/`ios/`/`web/` giả; user chạy `flutter create . --platforms=...` khi có SDK để tránh scaffold sai version.
