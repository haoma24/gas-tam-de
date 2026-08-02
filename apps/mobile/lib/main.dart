import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'features/auth/admin_login_page.dart';
import 'features/auth/auth_models.dart';
import 'features/auth/auth_session.dart';
import 'features/auth/otp_page.dart';
import 'features/auth/phone_page.dart';
import 'features/billing/admin_debts_page.dart';
import 'features/catalog/admin_product_form_page.dart';
import 'features/catalog/admin_products_page.dart';
import 'features/catalog/catalog_models.dart';
import 'features/dashboard/admin_dashboard_page.dart';
import 'features/home/home_page.dart';
import 'features/inventory/admin_inventory_page.dart';
import 'features/order/admin_delivery_fee_page.dart';
import 'features/order/admin_desk_settings_page.dart';
import 'features/order/admin_orders_page.dart';
import 'features/order/admin_store_page.dart';
import 'features/order/my_orders_page.dart';
import 'features/order/order_address_page.dart';
import 'features/order/order_models.dart';
import 'features/order/order_review_page.dart';
import 'features/order/order_success_page.dart';
import 'features/order/select_products_page.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final prefs = await SharedPreferences.getInstance();
  runApp(
    ProviderScope(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
      ],
      child: const GasTamDeApp(),
    ),
  );
}

final _router = GoRouter(
  routes: [
    GoRoute(
      path: '/',
      builder: (context, state) => Consumer(
        builder: (context, ref, _) {
          return HomePage(
            onStartOrder: () {
              final session = ref.read(authSessionProvider);
              if (session != null && session.isCustomer) {
                context.go('/order');
              } else {
                context.go('/auth/phone');
              }
            },
            onMyOrders: () {
              final session = ref.read(authSessionProvider);
              if (session != null && session.isCustomer) {
                context.go('/orders/history');
              } else {
                context.go('/auth/phone');
              }
            },
            onAdminLogin: () {
              final session = ref.read(authSessionProvider);
              if (session != null && session.isAdmin) {
                context.go('/admin');
              } else {
                context.go('/admin/login');
              }
            },
          );
        },
      ),
    ),
    GoRoute(
      path: '/auth/phone',
      builder: (context, state) => PhonePage(
        onBack: () => context.go('/'),
        onOtpSent: (args) => context.go('/auth/otp', extra: args),
      ),
    ),
    GoRoute(
      path: '/auth/otp',
      builder: (context, state) {
        final args = state.extra;
        if (args is! OtpNavArgs) {
          // Deep-link / refresh without extra → restart phone step.
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (context.mounted) context.go('/auth/phone');
          });
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        return OtpPage(
          args: args,
          onBack: () => context.go('/auth/phone'),
          onVerified: () => context.go('/order'),
        );
      },
    ),
    GoRoute(
      path: '/order',
      builder: (context, state) => SelectProductsPage(
        onBack: () => context.go('/'),
        onContinue: () => context.go('/order/address'),
      ),
    ),
    GoRoute(
      path: '/order/address',
      builder: (context, state) => OrderAddressPage(
        onBack: () => context.go('/order'),
        onContinue: () => context.go('/order/review'),
      ),
    ),
    GoRoute(
      path: '/order/review',
      builder: (context, state) => OrderReviewPage(
        onBack: () => context.go('/order/address'),
        onPlaced: (PlacedOrder order) =>
            context.go('/order/success', extra: order),
      ),
    ),
    GoRoute(
      path: '/order/success',
      builder: (context, state) {
        final order = state.extra;
        if (order is! PlacedOrder) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (context.mounted) context.go('/');
          });
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        return OrderSuccessPage(
          order: order,
          onDone: () => context.go('/'),
        );
      },
    ),
    GoRoute(
      path: '/orders/history',
      builder: (context, state) => MyOrdersPage(
        onBack: () => context.go('/'),
      ),
    ),
    GoRoute(
      path: '/admin/login',
      builder: (context, state) => Consumer(
        builder: (context, ref, _) {
          final session = ref.watch(authSessionProvider);
          if (session != null && session.isAdmin) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (context.mounted) context.go('/admin');
            });
            return const Scaffold(
              body: Center(child: CircularProgressIndicator()),
            );
          }
          return AdminLoginPage(
            onBack: () => context.go('/'),
            onLoggedIn: () => context.go('/admin'),
          );
        },
      ),
    ),
    GoRoute(
      path: '/admin',
      builder: (context, state) => AdminDashboardPage(
        onBack: () => context.go('/'),
        onOpenOrders: () => context.go('/admin/orders'),
        onOpenProducts: () => context.go('/admin/products'),
        onOpenDeliveryFee: () => context.go('/admin/delivery-fee'),
        onOpenStore: () => context.go('/admin/store'),
        onOpenDeskSettings: () => context.go('/admin/desk-settings'),
        onOpenDebts: () => context.go('/admin/debts'),
        onOpenInventory: () => context.go('/admin/inventory'),
        onLoggedOut: () => context.go('/'),
      ),
    ),
    GoRoute(
      path: '/admin/orders',
      builder: (context, state) => AdminOrdersPage(
        onBack: () => context.go('/admin'),
        onOpenOrder: (AdminOrder o) =>
            context.go('/admin/orders/detail', extra: o),
      ),
    ),
    GoRoute(
      path: '/admin/orders/detail',
      builder: (context, state) {
        final order = state.extra;
        if (order is! AdminOrder) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (context.mounted) context.go('/admin/orders');
          });
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        return AdminOrderDetailPage(
          order: order,
          onBack: () => context.go('/admin/orders'),
          onCompleted: () => context.go('/admin/orders'),
        );
      },
    ),
    GoRoute(
      path: '/admin/products',
      builder: (context, state) => AdminProductsPage(
        onBack: () => context.go('/admin'),
        onCreate: () => context.go('/admin/products/new'),
        onEdit: (Product p) => context.go('/admin/products/${p.id}'),
      ),
    ),
    GoRoute(
      path: '/admin/products/new',
      builder: (context, state) => AdminProductFormPage(
        onBack: () => context.go('/admin/products'),
        onDone: () => context.go('/admin/products'),
      ),
    ),
    GoRoute(
      path: '/admin/products/:id',
      builder: (context, state) {
        final id = state.pathParameters['id'] ?? '';
        return AdminProductFormPage(
          productId: id,
          onBack: () => context.go('/admin/products'),
          onDone: () => context.go('/admin/products'),
        );
      },
    ),
    GoRoute(
      path: '/admin/delivery-fee',
      builder: (context, state) => AdminDeliveryFeePage(
        onBack: () => context.go('/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/store',
      builder: (context, state) => AdminStorePage(
        onBack: () => context.go('/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/desk-settings',
      builder: (context, state) => AdminDeskSettingsPage(
        onBack: () => context.go('/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/debts',
      builder: (context, state) => AdminDebtsPage(
        onBack: () => context.go('/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/inventory',
      builder: (context, state) => AdminInventoryPage(
        onBack: () => context.go('/admin'),
      ),
    ),
  ],
);

/// App shell — theme + router; waits for session bootstrap before routes.
class GasTamDeApp extends ConsumerWidget {
  const GasTamDeApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    const seed = Color(0xFFB45309); // amber/gas cylinder tone — not purple default
    final boot = ref.watch(authBootstrapProvider);

    return MaterialApp.router(
      title: 'Gas Tam Đệ',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: seed,
          brightness: Brightness.light,
        ),
        useMaterial3: true,
      ),
      routerConfig: _router,
      builder: (context, child) {
        if (boot.isLoading) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        return child ?? const SizedBox.shrink();
      },
    );
  }
}
