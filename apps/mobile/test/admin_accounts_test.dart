import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/auth/admin_accounts_api.dart';
import 'package:gas_tam_de/features/auth/admin_admin_accounts_page.dart';
import 'package:gas_tam_de/features/auth/auth_models.dart';

Dio accountsFakeDio(
  Map<String, Response<dynamic> Function(RequestOptions)> routes,
) {
  final dio = Dio();
  dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
    final route = routes['${options.method} ${options.path}'];
    if (route == null) {
      handler.reject(
        DioException(
          requestOptions: options,
          response: Response<void>(requestOptions: options, statusCode: 404),
        ),
        true,
      );
      return;
    }
    handler.resolve(route(options));
  }));
  return dio;
}

Response<Map<String, dynamic>> accountsOK(
  RequestOptions options,
  Map<String, dynamic> data, {
  int statusCode = 200,
}) {
  return Response<Map<String, dynamic>>(
    requestOptions: options,
    statusCode: statusCode,
    data: data,
  );
}

Widget accountsHost(Dio dio) {
  return ProviderScope(
    overrides: [
      adminAccountsApiProvider.overrideWithValue(AdminAccountsApi(dio)),
    ],
    child: MaterialApp(home: AdminAccountsPage(onBack: () {})),
  );
}

void main() {
  test('AdminAccount parses list response without credential data', () {
    final account = AdminAccount.fromJson(const {
      'id': 'a1',
      'username': 'manager',
      'display_name': ' Quản lý ',
      'created_at': '2026-09-05T00:00:00Z',
      'is_self': true,
    });
    expect(account.username, 'manager');
    expect(account.displayName, 'Quản lý');
    expect(account.isSelf, isTrue);
  });

  test('account API errors have Vietnamese copy', () {
    expect(
      AuthApiException(code: 'USERNAME_EXISTS', message: '').displayMessage,
      'Tên đăng nhập này đã được sử dụng.',
    );
    expect(
      AuthApiException(code: 'CURRENT_PASSWORD_INVALID', message: '')
          .displayMessage,
      'Mật khẩu hiện tại không đúng.',
    );
  });

  testWidgets('page lists and creates a manager account', (tester) async {
    var listCalls = 0;
    Map<dynamic, dynamic>? posted;
    final dio = accountsFakeDio({
      'GET /v1/admin/admin-accounts': (options) {
        listCalls++;
        return accountsOK(options, {
          'admin_accounts': [
            if (listCalls > 1)
              {
                'id': 'a2',
                'username': 'manager02',
                'display_name': 'Ca tối',
              },
          ],
        });
      },
      'POST /v1/admin/admin-accounts': (options) {
        posted = options.data as Map;
        return accountsOK(
          options,
          const {
            'id': 'a2',
            'username': 'manager02',
            'display_name': 'Ca tối',
          },
          statusCode: 201,
        );
      },
    });

    await tester.pumpWidget(accountsHost(dio));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.widgetWithText(TextField, 'Tên đăng nhập'),
      'manager02',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Tên hiển thị (tùy chọn)'),
      'Ca tối',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Mật khẩu ban đầu'),
      'strong-password',
    );
    await tester.tap(find.text('Tạo tài khoản quản lý'));
    await tester.pumpAndSettle();

    expect(posted?['username'], 'manager02');
    expect(posted?['password'], 'strong-password');
    expect(listCalls, 2);
    expect(find.text('Ca tối'), findsOneWidget);
    expect(find.text('manager02'), findsOneWidget);
  });
}
