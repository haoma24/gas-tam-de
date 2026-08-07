import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'features/auth/admin_admin_phones_page.dart';
import 'features/auth/admin_login_page.dart';
import 'features/auth/auth_models.dart';
import 'features/auth/auth_session.dart';
import 'features/auth/customer_auth_flow_page.dart';
import 'features/auth/customer_profile_page.dart';
import 'features/billing/admin_debts_page.dart';
import 'features/catalog/admin_product_form_page.dart';
import 'features/catalog/admin_products_page.dart';
import 'features/catalog/catalog_models.dart';
import 'features/dashboard/admin_dashboard_page.dart';
import 'features/home/customer_shop_page.dart';
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

void _popOrGo(BuildContext context, String fallbackLocation) {
  if (GoRouter.of(context).canPop()) {
    context.pop();
  } else {
    context.go(fallbackLocation);
  }
}

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
          final session = ref.watch(authSessionProvider);

          if (session != null && session.isAdmin) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (context.mounted) context.go('/admin');
            });
            return const Scaffold(
              body: Center(child: CircularProgressIndicator()),
            );
          }

          if (session != null && session.isCustomer) {
            return CustomerShopPage(
              onStartOrder: () => context.push('/order'),
              onProfile: () => context.push('/profile'),
            );
          }

          return HomePage(
            onLogin: () => context.push('/auth/phone'),
          );
        },
      ),
    ),
    GoRoute(
      path: '/auth/phone',
      builder: (context, state) => Consumer(
        builder: (context, ref, _) {
          return CustomerAuthFlowPage(
            onBack: () => _popOrGo(context, '/'),
            onVerified: () {
              final session = ref.read(authSessionProvider);
              if (session != null && session.isAdmin) {
                context.go('/admin');
              } else {
                context.go('/');
              }
            },
          );
        },
      ),
    ),
    GoRoute(
      path: '/auth/otp',
      builder: (context, state) {
        final args = state.extra;
        if (args is! OtpNavArgs) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (context.mounted) context.go('/auth/phone');
          });
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        return Consumer(
          builder: (context, ref, _) {
            return CustomerAuthFlowPage(
              initialOtpArgs: args,
              onBack: () => _popOrGo(context, '/auth/phone'),
              onVerified: () {
                final session = ref.read(authSessionProvider);
                if (session != null && session.isAdmin) {
                  context.go('/admin');
                } else {
                  context.go('/');
                }
              },
            );
          },
        );
      },
    ),
    GoRoute(
      path: '/profile',
      builder: (context, state) => Consumer(
        builder: (context, ref, _) {
          final session = ref.watch(authSessionProvider);
          if (session == null || !session.isCustomer) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (context.mounted) context.go('/auth/phone');
            });
            return const Scaffold(
              body: Center(child: CircularProgressIndicator()),
            );
          }
          return CustomerProfilePage(
            onBack: () => _popOrGo(context, '/'),
            onMyOrders: () => context.push('/orders/history'),
            onLoggedOut: () => context.go('/'),
          );
        },
      ),
    ),
    GoRoute(
      path: '/order',
      builder: (context, state) => SelectProductsPage(
        onBack: () => _popOrGo(context, '/'),
        onContinue: () => context.push('/order/address'),
      ),
    ),
    GoRoute(
      path: '/order/address',
      builder: (context, state) => OrderAddressPage(
        onBack: () => _popOrGo(context, '/order'),
        onContinue: () => context.push('/order/review'),
      ),
    ),
    GoRoute(
      path: '/order/review',
      builder: (context, state) => OrderReviewPage(
        onBack: () => _popOrGo(context, '/order/address'),
        onPlaced: (PlacedOrder order) =>
            context.push('/order/success', extra: order),
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
        // Order history lives under profile after login.
        onBack: () => _popOrGo(context, '/profile'),
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
            onBack: () => _popOrGo(context, '/'),
            onLoggedIn: () => context.go('/admin'),
          );
        },
      ),
    ),
    GoRoute(
      path: '/admin',
      builder: (context, state) => Consumer(
        builder: (context, ref, _) {
          final session = ref.watch(authSessionProvider);
          if (session == null || !session.isAdmin) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (context.mounted) context.go('/admin/login');
            });
            return const Scaffold(
              body: Center(child: CircularProgressIndicator()),
            );
          }
          return AdminDashboardPage(
            onBack: () => _popOrGo(context, '/'),
            onOpenOrders: () => context.push('/admin/orders'),
            onOpenProducts: () => context.push('/admin/products'),
            onOpenDeliveryFee: () => context.push('/admin/delivery-fee'),
            onOpenStore: () => context.push('/admin/store'),
            onOpenDeskSettings: () => context.push('/admin/desk-settings'),
            onOpenDebts: () => context.push('/admin/debts'),
            onOpenInventory: () => context.push('/admin/inventory'),
            onOpenAdminPhones: () => context.push('/admin/admin-phones'),
            onLoggedOut: () => context.go('/'),
          );
        },
      ),
    ),
    GoRoute(
      path: '/admin/orders',
      builder: (context, state) => AdminOrdersPage(
        onBack: () => _popOrGo(context, '/admin'),
        onOpenOrder: (AdminOrder o) =>
            context.push('/admin/orders/detail', extra: o),
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
          onBack: () => _popOrGo(context, '/admin/orders'),
          onCompleted: () => _popOrGo(context, '/admin/orders'),
        );
      },
    ),
    GoRoute(
      path: '/admin/products',
      builder: (context, state) => AdminProductsPage(
        onBack: () => _popOrGo(context, '/admin'),
        onCreate: () => context.push('/admin/products/new'),
        onEdit: (Product p) => context.push('/admin/products/${p.id}'),
      ),
    ),
    GoRoute(
      path: '/admin/products/new',
      builder: (context, state) => AdminProductFormPage(
        onBack: () => _popOrGo(context, '/admin/products'),
        onDone: () => _popOrGo(context, '/admin/products'),
      ),
    ),
    GoRoute(
      path: '/admin/products/:id',
      builder: (context, state) {
        final id = state.pathParameters['id'] ?? '';
        return AdminProductFormPage(
          productId: id,
          onBack: () => _popOrGo(context, '/admin/products'),
          onDone: () => _popOrGo(context, '/admin/products'),
        );
      },
    ),
    GoRoute(
      path: '/admin/delivery-fee',
      builder: (context, state) => AdminDeliveryFeePage(
        onBack: () => _popOrGo(context, '/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/store',
      builder: (context, state) => AdminStorePage(
        onBack: () => _popOrGo(context, '/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/desk-settings',
      builder: (context, state) => AdminDeskSettingsPage(
        onBack: () => _popOrGo(context, '/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/debts',
      builder: (context, state) => AdminDebtsPage(
        onBack: () => _popOrGo(context, '/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/inventory',
      builder: (context, state) => AdminInventoryPage(
        onBack: () => _popOrGo(context, '/admin'),
      ),
    ),
    GoRoute(
      path: '/admin/admin-phones',
      builder: (context, state) => AdminPhonesPage(
        onBack: () => _popOrGo(context, '/admin'),
      ),
    ),
  ],
);

TextTheme _brandTextTheme(Brightness brightness) {
  final base = GoogleFonts.beVietnamProTextTheme(
    brightness == Brightness.light
        ? ThemeData.light().textTheme
        : ThemeData.dark().textTheme,
  );
  return base.copyWith(
    displayMedium: base.displayMedium?.copyWith(fontWeight: FontWeight.w800),
    displaySmall: base.displaySmall?.copyWith(fontWeight: FontWeight.w800),
    headlineSmall: base.headlineSmall?.copyWith(fontWeight: FontWeight.w700),
  );
}

/// App shell — theme + router; waits for session bootstrap before routes.
class GasTamDeApp extends ConsumerWidget {
  const GasTamDeApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    const seed =
        Color(0xFFB45309); // amber/gas cylinder tone — not purple default
    final boot = ref.watch(authBootstrapProvider);

    return MaterialApp.router(
      title: 'Gas Tam Đệ',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: seed,
          brightness: Brightness.light,
        ),
        textTheme: _brandTextTheme(Brightness.light),
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
