/// API base URL for local / emulator.
///
/// Default targets the gateway (`:8080`) which reverse-proxies to all services
/// (T9.1.1). Prefer the gateway for Flutter Web + mobile — do not call
/// upstream ports directly unless debugging a single service.
///
/// ```
/// --dart-define=API_BASE_URL=http://127.0.0.1:8080
/// ```
///
/// Android emulator → host machine gateway: `http://10.0.2.2:8080`.
/// iOS simulator / Chrome / desktop → `http://127.0.0.1:8080`.
class ApiConfig {
  static const String gatewayBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080',
  );
}
