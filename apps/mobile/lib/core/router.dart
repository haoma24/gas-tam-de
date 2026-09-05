import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/admin_admin_accounts_page.dart';
import '../features/auth/admin_admin_phones_page.dart';
import '../features/auth/admin_login_page.dart';
import '../features/auth/auth_session.dart';
import '../features/auth/customer_profile_page.dart';
import '../features/auth/google_login_page.dart';
import '../features/catalog/admin_product_form_page.dart';
import '../features/catalog/admin_products_page.dart';
import '../features/catalog/catalog_models.dart';
import '../features/catalog/product_detail_page.dart';
import '../features/dashboard/admin_reports_page.dart';
import '../features/dashboard/admin_settings_page.dart';
import '../features/home/customer_shop_page.dart';
import '../features/home/welcome_page.dart';
import '../features/inventory/admin_inventory_page.dart';
import '../features/order/admin_delivery_fee_page.dart';
import '../features/order/admin_desk_settings_page.dart';
import '../features/order/admin_orders_page.dart';
import '../features/order/admin_store_page.dart';
import '../features/order/my_orders_page.dart';
import '../features/order/order_address_page.dart';
import '../features/order/order_models.dart';
import '../features/order/order_page.dart';
import '../features/order/order_success_page.dart';
import '../features/shell/admin_shell.dart';
import '../features/shell/customer_shell.dart';

final _rootKey = GlobalKey<NavigatorState>(debugLabel: 'root');
final _customerShellKey = GlobalKey<NavigatorState>(debugLabel: 'customer');

/// Routes that an unauthenticated visitor is allowed to see.
bool _isPublic(String location) =>
    location == '/welcome' ||
    location.startsWith('/auth') ||
    location == '/admin/login';

final routerProvider = Provider<GoRouter>((ref) {
  final refresh = _SessionRefresh(ref);
  ref.onDispose(refresh.dispose);

  return GoRouter(
    navigatorKey: _rootKey,
    initialLocation: '/',
    refreshListenable: refresh,

    // One guard for the whole app. This used to be four copies of
    // `Consumer + addPostFrameCallback + spinner` inside route builders.
    redirect: (context, state) {
      final session = ref.read(authSessionProvider);
      final loc = state.matchedLocation;
      final isAdmin = session?.isAdmin ?? false;

      if (loc == '/admin/login') return isAdmin ? '/admin' : null;
      if (loc.startsWith('/admin')) return isAdmin ? null : '/admin/login';

      // Everything below is the customer surface.
      if (isAdmin) return '/admin';
      if (session == null) return _isPublic(loc) ? null : '/welcome';
      return _isPublic(loc) ? '/' : null;
    },

    routes: [
      GoRoute(
        path: '/welcome',
        builder: (context, state) => const WelcomePage(),
      ),
      GoRoute(
        path: '/auth/login',
        builder: (context, state) => const GoogleLoginPage(),
      ),
      // Legacy OTP entry points — the app signs in with Google now.
      GoRoute(path: '/auth/phone', redirect: (_, __) => '/auth/login'),
      GoRoute(path: '/auth/otp', redirect: (_, __) => '/auth/login'),
      GoRoute(
        path: '/admin/login',
        builder: (context, state) => const AdminLoginPage(),
      ),

      // ── Customer full-screen routes (above the shell, no bottom nav) ──
      GoRoute(
        path: '/products/:id',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => ProductDetailRoutePage(
          productId: state.pathParameters['id'] ?? '',
          initialProduct:
              state.extra is Product ? state.extra as Product : null,
        ),
      ),
      GoRoute(
        path: '/order',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => const OrderPage(),
        routes: [
          GoRoute(
            path: 'address',
            parentNavigatorKey: _rootKey,
            builder: (context, state) => const OrderAddressPage(),
          ),
          GoRoute(
            path: 'success',
            parentNavigatorKey: _rootKey,
            builder: (context, state) {
              final order = state.extra;
              if (order is! PlacedOrder) return const _Bounce(to: '/');
              return OrderSuccessPage(order: order);
            },
          ),
        ],
      ),

      // ── Customer shell: persistent bottom nav across three tabs ──
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) =>
            CustomerShell(navigationShell: navigationShell),
        branches: [
          StatefulShellBranch(
            navigatorKey: _customerShellKey,
            routes: [
              GoRoute(
                path: '/',
                builder: (context, state) => const CustomerShopPage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/orders',
                builder: (context, state) => const MyOrdersPage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/profile',
                builder: (context, state) => const CustomerProfilePage(),
              ),
            ],
          ),
        ],
      ),

      // ── Admin full-screen routes (pushed from the Settings tab) ──
      GoRoute(
        path: '/admin/orders/detail',
        parentNavigatorKey: _rootKey,
        builder: (context, state) {
          final order = state.extra;
          if (order is! AdminOrder) return const _Bounce(to: '/admin');
          return AdminOrderDetailPage(
            order: order,
            onCompleted: () => popOrGoRoot(context, '/admin'),
          );
        },
      ),
      GoRoute(
        path: '/admin/products',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => const AdminProductsPage(),
        routes: [
          GoRoute(
            path: 'new',
            parentNavigatorKey: _rootKey,
            builder: (context, state) => const AdminProductFormPage(),
          ),
          GoRoute(
            path: ':id',
            parentNavigatorKey: _rootKey,
            builder: (context, state) => AdminProductFormPage(
              productId: state.pathParameters['id'] ?? '',
            ),
          ),
        ],
      ),
      GoRoute(
        path: '/admin/delivery-fee',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => const AdminDeliveryFeePage(),
      ),
      GoRoute(
        path: '/admin/store',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => const AdminStorePage(),
      ),
      GoRoute(
        path: '/admin/desk-settings',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => const AdminDeskSettingsPage(),
      ),
      GoRoute(
        path: '/admin/admin-phones',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => const AdminPhonesPage(),
      ),
      GoRoute(
        path: '/admin/admin-accounts',
        parentNavigatorKey: _rootKey,
        builder: (context, state) => const AdminAccountsPage(),
      ),

      // ── Admin shell: four tabs, bottom bar or rail by width ──
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) =>
            AdminShell(navigationShell: navigationShell),
        branches: [
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/admin',
                builder: (context, state) => const AdminOrdersPage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/admin/inventory',
                builder: (context, state) => const AdminInventoryPage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/admin/reports',
                builder: (context, state) => const AdminReportsPage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/admin/settings',
                builder: (context, state) => const AdminSettingsPage(),
              ),
            ],
          ),
        ],
      ),
    ],
  );
});

/// Pops on the root navigator (used by full-screen routes pushed above a
/// shell), falling back to [fallback] on a cold deep link.
void popOrGoRoot(BuildContext context, String fallback) {
  final router = GoRouter.of(context);
  if (router.canPop()) {
    router.pop();
  } else {
    router.go(fallback);
  }
}

/// Rebuilds the router's redirect whenever the session changes.
class _SessionRefresh extends ChangeNotifier {
  _SessionRefresh(Ref ref) {
    _sub = ref.listen<AuthSession?>(
      authSessionProvider,
      (_, __) => notifyListeners(),
    );
  }

  late final ProviderSubscription<AuthSession?> _sub;

  @override
  void dispose() {
    _sub.close();
    super.dispose();
  }
}

/// Placeholder shown for one frame when a route was entered without the
/// `extra` payload it needs (a hard reload or a hand-typed URL).
class _Bounce extends StatefulWidget {
  const _Bounce({required this.to});

  final String to;

  @override
  State<_Bounce> createState() => _BounceState();
}

class _BounceState extends State<_Bounce> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) GoRouter.of(context).go(widget.to);
    });
  }

  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Center(child: CircularProgressIndicator()));
}
