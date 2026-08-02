# Android (Flutter)

Hand-maintained: `AndroidManifest.xml` (location + maps queries), Gradle Kotlin DSL entries, themes, mipmap icons, `MainActivity`.

**Gradle wrapper** (`gradlew`) and `local.properties` come from Flutter SDK / local machine. Bootstrap:

```bash
cd apps/mobile
flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=android
```

Emulator: `flutter run -d android` (host API → `10.0.2.2`).
