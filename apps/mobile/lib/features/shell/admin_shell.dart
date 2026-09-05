import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';

/// Persistent admin chrome — four tabs instead of the nine dashboard tiles.
///
/// Responsive: a bottom NavigationBar on phones, a left NavigationRail from
/// [AppBreakpoints.expanded] up, where the Order Desk also shows its detail
/// pane beside the list.
class AdminShell extends StatelessWidget {
  const AdminShell({super.key, required this.navigationShell});

  final StatefulNavigationShell navigationShell;

  static const _items = <({IconData icon, IconData selected, String label})>[
    (
      icon: Icons.receipt_long_outlined,
      selected: Icons.receipt_long_rounded,
      label: 'Đơn',
    ),
    (
      icon: Icons.inventory_2_outlined,
      selected: Icons.inventory_2_rounded,
      label: 'Kho',
    ),
    (
      icon: Icons.bar_chart_outlined,
      selected: Icons.bar_chart_rounded,
      label: 'Báo cáo',
    ),
    (
      icon: Icons.settings_outlined,
      selected: Icons.settings_rounded,
      label: 'Cài đặt',
    ),
  ];

  void _go(int index) => navigationShell.goBranch(
        index,
        initialLocation: index == navigationShell.currentIndex,
      );

  @override
  Widget build(BuildContext context) {
    final p = context.palette;

    if (context.isExpanded) {
      return Scaffold(
        backgroundColor: p.bg,
        body: Row(
          children: [
            NavigationRail(
              selectedIndex: navigationShell.currentIndex,
              onDestinationSelected: _go,
              labelType: NavigationRailLabelType.all,
              destinations: [
                for (final item in _items)
                  NavigationRailDestination(
                    icon: Icon(item.icon),
                    selectedIcon: Icon(item.selected),
                    label: Text(item.label),
                  ),
              ],
            ),
            VerticalDivider(width: 1, color: p.border),
            Expanded(child: navigationShell),
          ],
        ),
      );
    }

    return Scaffold(
      backgroundColor: p.bg,
      body: navigationShell,
      bottomNavigationBar: DecoratedBox(
        decoration: BoxDecoration(
          border: Border(top: BorderSide(color: p.border)),
        ),
        child: NavigationBar(
          selectedIndex: navigationShell.currentIndex,
          onDestinationSelected: _go,
          destinations: [
            for (final item in _items)
              NavigationDestination(
                icon: Icon(item.icon),
                selectedIcon: Icon(item.selected),
                label: item.label,
              ),
          ],
        ),
      ),
    );
  }
}
