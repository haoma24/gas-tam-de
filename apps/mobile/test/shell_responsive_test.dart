import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/core/ui/ui.dart';
import 'package:gas_tam_de/features/shell/admin_shell.dart';
import 'package:gas_tam_de/features/shell/customer_shell.dart';
import 'package:go_router/go_router.dart';

/// Pumps a real StatefulShellRoute so the shells get a genuine
/// [StatefulNavigationShell], then sizes the surface to a given width.
Future<void> _pumpShell(
  WidgetTester tester, {
  required double width,
  required bool admin,
}) async {
  tester.view.physicalSize = Size(width, 900);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);

  final router = GoRouter(
    initialLocation: admin ? '/admin' : '/',
    routes: [
      StatefulShellRoute.indexedStack(
        builder: (context, state, shell) => admin
            ? AdminShell(navigationShell: shell)
            : CustomerShell(navigationShell: shell),
        branches: [
          for (final path in admin
              ? const [
                  '/admin',
                  '/admin/inventory',
                  '/admin/reports',
                  '/admin/settings'
                ]
              : const ['/', '/orders', '/profile'])
            StatefulShellBranch(
              routes: [
                GoRoute(
                  path: path,
                  builder: (context, state) => Text('body $path'),
                ),
              ],
            ),
        ],
      ),
    ],
  );
  addTearDown(router.dispose);

  await tester.pumpWidget(
    MaterialApp.router(
      theme: buildAppTheme(Brightness.light),
      routerConfig: router,
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  group('customer shell', () {
    testWidgets('keeps a bottom nav with three destinations', (tester) async {
      await _pumpShell(tester, width: 400, admin: false);

      expect(find.byType(NavigationBar), findsOneWidget);
      expect(find.text('Cửa hàng'), findsOneWidget);
      expect(find.text('Đơn hàng'), findsOneWidget);
      expect(find.text('Hồ sơ'), findsOneWidget);
    });

    testWidgets('switching tab keeps the shell mounted', (tester) async {
      await _pumpShell(tester, width: 400, admin: false);
      expect(find.text('body /'), findsOneWidget);

      await tester.tap(find.text('Hồ sơ'));
      await tester.pumpAndSettle();

      // The nav bar survives the tab change — that is the whole point of
      // moving it out of CustomerShopPage into the shell.
      expect(find.byType(NavigationBar), findsOneWidget);
      expect(find.text('body /profile'), findsOneWidget);
    });
  });

  group('admin shell', () {
    testWidgets('uses a bottom nav below the expanded breakpoint', (
      tester,
    ) async {
      await _pumpShell(tester, width: AppBreakpoints.expanded - 1, admin: true);

      expect(find.byType(NavigationBar), findsOneWidget);
      expect(find.byType(NavigationRail), findsNothing);
    });

    testWidgets('switches to a rail at the expanded breakpoint', (
      tester,
    ) async {
      await _pumpShell(tester, width: AppBreakpoints.expanded, admin: true);

      expect(find.byType(NavigationRail), findsOneWidget);
      expect(find.byType(NavigationBar), findsNothing);
    });

    testWidgets('exposes the four admin destinations', (tester) async {
      await _pumpShell(tester, width: 400, admin: true);

      for (final label in ['Đơn', 'Kho', 'Báo cáo', 'Cài đặt']) {
        expect(find.text(label), findsOneWidget, reason: 'missing $label');
      }
      // The Order Desk is the landing tab, not the dashboard.
      expect(find.text('body /admin'), findsOneWidget);
    });
  });
}
