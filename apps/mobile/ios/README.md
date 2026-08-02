# iOS (Flutter)

Hand-maintained: `Runner/Info.plist`, `AppDelegate.swift`, `Podfile`, storyboards, `Flutter/*.xcconfig`, `Runner.xcodeproj` (shell).

Nếu tooling Flutter báo thiếu file generate (`Generated.xcconfig`, Pods, …):

```bash
cd apps/mobile
flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=ios
flutter pub get
```

Rồi `flutter run -d ios` (Simulator, macOS) hoặc mở `ios/Runner.xcworkspace` trong Xcode.
