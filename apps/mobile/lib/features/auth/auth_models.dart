import 'package:dio/dio.dart';

/// Args passed from phone screen → OTP screen (via go_router `extra`).
class OtpNavArgs {
  const OtpNavArgs({
    required this.phone,
    required this.phoneMasked,
    required this.resendAfterSec,
    required this.expiresInSec,
    this.devCode,
    this.requestOtpOnMount = false,
  });

  final String phone;
  final String phoneMasked;
  final int resendAfterSec;
  final int expiresInSec;
  final String? devCode;

  /// When true, [OtpPage] / auth flow sends OTP after open (phone screen navigates
  /// here synchronously on button tap so mobile web keeps the user-gesture chain).
  final bool requestOtpOnMount;
}

class OtpRequestResult {
  const OtpRequestResult({
    required this.phoneMasked,
    required this.expiresInSec,
    required this.resendAfterSec,
    this.devCode,
  });

  final String phoneMasked;
  final int expiresInSec;
  final int resendAfterSec;
  final String? devCode;

  factory OtpRequestResult.fromJson(Map<String, dynamic> json) {
    return OtpRequestResult(
      phoneMasked: json['phone_masked'] as String? ?? '',
      expiresInSec: (json['expires_in_sec'] as num?)?.toInt() ?? 300,
      resendAfterSec: (json['resend_after_sec'] as num?)?.toInt() ?? 60,
      devCode: json['dev_code'] as String?,
    );
  }
}

class AuthUser {
  const AuthUser({
    required this.id,
    required this.role,
    this.phoneMasked = '',
    this.username,
    this.displayName,
  });

  final String id;
  final String role;
  final String phoneMasked;
  final String? username;
  final String? displayName;

  factory AuthUser.fromJson(Map<String, dynamic> json) {
    return AuthUser(
      id: json['id'] as String? ?? '',
      role: json['role'] as String? ?? 'customer',
      phoneMasked: json['phone_masked'] as String? ?? '',
      username: json['username'] as String?,
      displayName: json['display_name'] as String?,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'role': role,
        'phone_masked': phoneMasked,
        if (username != null) 'username': username,
        if (displayName != null) 'display_name': displayName,
      };
}

/// Shared token payload from OTP verify or admin login.
class AuthTokenResult {
  const AuthTokenResult({
    required this.accessToken,
    required this.refreshToken,
    required this.tokenType,
    required this.expiresIn,
    required this.user,
  });

  final String accessToken;
  final String refreshToken;
  final String tokenType;
  final int expiresIn;
  final AuthUser user;

  factory AuthTokenResult.fromJson(Map<String, dynamic> json) {
    final userJson = json['user'];
    return AuthTokenResult(
      accessToken: json['access_token'] as String? ?? '',
      refreshToken: json['refresh_token'] as String? ?? '',
      tokenType: json['token_type'] as String? ?? 'Bearer',
      expiresIn: (json['expires_in'] as num?)?.toInt() ?? 900,
      user: userJson is Map<String, dynamic>
          ? AuthUser.fromJson(userJson)
          : const AuthUser(id: '', role: 'customer'),
    );
  }
}

/// Alias kept for OTP call sites (T1.1.4).
typedef OtpVerifyResult = AuthTokenResult;

/// Alias for admin login response (T1.2.3).
typedef AdminLoginResult = AuthTokenResult;

/// API error from auth-service `{"error":{"code","message",...}}`.
class AuthApiException implements Exception {
  AuthApiException({
    required this.code,
    required this.message,
    this.statusCode,
    this.retryAfterSec,
    this.attemptsRemaining,
  });

  final String code;
  final String message;
  final int? statusCode;
  final int? retryAfterSec;
  final int? attemptsRemaining;

  @override
  String toString() => 'AuthApiException($code: $message)';

  /// User-facing Vietnamese copy for common OTP codes.
  String get displayMessage {
    switch (code) {
      case 'INVALID_PHONE':
        return 'Số điện thoại không hợp lệ.';
      case 'RATE_LIMITED':
        return retryAfterSec != null
            ? 'Gửi quá nhiều lần. Thử lại sau $retryAfterSec giây.'
            : 'Gửi quá nhiều lần. Vui lòng thử lại sau.';
      case 'SMS_FAILED':
        return 'Không gửi được SMS. Thử lại sau.';
      case 'INVALID_CODE':
        return 'Mã OTP phải gồm 6 chữ số.';
      case 'OTP_NOT_FOUND':
        return 'Chưa có mã OTP. Hãy gửi lại mã.';
      case 'OTP_CONSUMED':
        return 'Mã OTP đã dùng. Hãy gửi mã mới.';
      case 'OTP_EXPIRED':
        return 'Mã OTP hết hạn. Hãy gửi mã mới.';
      case 'OTP_LOCKED':
        return 'Nhập sai quá nhiều lần. Hãy gửi mã mới.';
      case 'OTP_INVALID':
        return attemptsRemaining != null
            ? 'Mã OTP sai. Còn $attemptsRemaining lần thử.'
            : 'Mã OTP không đúng.';
      case 'INVALID_CREDENTIALS':
        return 'Tên đăng nhập hoặc mật khẩu không đúng.';
      case 'INVALID_TOKEN':
      case 'UNAUTHORIZED':
        return 'Phiên đăng nhập hết hạn. Vui lòng đăng nhập lại.';
      case 'FORBIDDEN':
        return 'Không có quyền thực hiện thao tác này.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ. Kiểm tra API đang chạy.';
      case 'api_unavailable':
        return 'API gateway chưa sẵn sàng. Khởi động lại stack (make compose-up hoặc bật service api-gateway trên VPS).';
      case 'BAD_GATEWAY':
        return 'Dịch vụ xác thực chưa sẵn sàng. Chạy auth-service (make compose-up hoặc make web-up).';
      default:
        return message.isNotEmpty ? message : 'Có lỗi xảy ra. Thử lại.';
    }
  }
}

/// Maps a Dio failure onto the `{"error":{"code","message"}}` envelope every Go
/// service returns, so all auth-backed clients surface the same Vietnamese copy.
AuthApiException mapDioToAuthException(DioException e) {
  if (e.type == DioExceptionType.connectionError ||
      e.type == DioExceptionType.connectionTimeout ||
      e.type == DioExceptionType.receiveTimeout ||
      e.type == DioExceptionType.sendTimeout) {
    return AuthApiException(
      code: 'NETWORK',
      message: e.message ?? 'network error',
    );
  }

  final data = e.response?.data;
  if (data is Map<String, dynamic>) {
    final err = data['error'];
    if (err is Map<String, dynamic>) {
      return AuthApiException(
        code: err['code'] as String? ?? 'UNKNOWN',
        message: err['message'] as String? ?? 'request failed',
        statusCode: e.response?.statusCode,
        retryAfterSec: (err['retry_after_sec'] as num?)?.toInt(),
        attemptsRemaining: (err['attempts_remaining'] as num?)?.toInt(),
      );
    }
  }

  return AuthApiException(
    code: 'HTTP',
    message: e.message ?? 'request failed',
    statusCode: e.response?.statusCode,
  );
}
