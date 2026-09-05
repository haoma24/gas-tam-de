import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';

/// Persistent customer chrome.
///
/// The bottom bar used to live inside `CustomerShopPage`, so it vanished on
/// every other screen. It belongs to the shell now; the order funnel and the
/// product detail push above it on the root navigator and stay full-screen.
class CustomerShell extends StatelessWidget {
  const CustomerShell({super.key, required this.navigationShell});

  final StatefulNavigationShell navigationShell;

  static const _destinations = <NavigationDestination>[
    NavigationDestination(
      icon: Icon(Icons.storefront_outlined),
      selectedIcon: Icon(Icons.storefront_rounded),
      label: 'Cửa hàng',
    ),
    NavigationDestination(
      icon: Icon(Icons.receipt_long_outlined),
      selectedIcon: Icon(Icons.receipt_long_rounded),
      label: 'Đơn hàng',
    ),
    NavigationDestination(
      icon: Icon(Icons.person_outline_rounded),
      selectedIcon: Icon(Icons.person_rounded),
      label: 'Hồ sơ',
    ),
  ];

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Scaffold(
      backgroundColor: p.bg,
      body: navigationShell,
      bottomNavigationBar: DecoratedBox(
        decoration: BoxDecoration(
          border: Border(top: BorderSide(color: p.border)),
        ),
        child: NavigationBar(
          selectedIndex: navigationShell.currentIndex,
          destinations: _destinations,
          onDestinationSelected: (i) => navigationShell.goBranch(
            i,
            // Tapping the active tab pops back to that branch's root.
            initialLocation: i == navigationShell.currentIndex,
          ),
        ),
      ),
    );
  }
}
