import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_session.dart';
import '../catalog/catalog_models.dart';
import 'dashboard_api.dart';
import 'dashboard_models.dart';

/// Admin home (`/admin`) — dashboard summary widgets + desk navigation (T8.1.3).
class AdminDashboardPage extends ConsumerStatefulWidget {
  const AdminDashboardPage({
    super.key,
    required this.onBack,
    required this.onOpenOrders,
    required this.onOpenProducts,
    required this.onOpenDeliveryFee,
    required this.onOpenStore,
    required this.onOpenDeskSettings,
    required this.onOpenDebts,
    required this.onOpenInventory,
    required this.onOpenAdminPhones,
    this.onLoggedOut,
  });

  final VoidCallback onBack;
  final VoidCallback onOpenOrders;
  final VoidCallback onOpenProducts;
  final VoidCallback onOpenDeliveryFee;
  final VoidCallback onOpenStore;
  final VoidCallback onOpenDeskSettings;
  final VoidCallback onOpenDebts;
  final VoidCallback onOpenInventory;
  final VoidCallback onOpenAdminPhones;
  final VoidCallback? onLoggedOut;

  @override
  ConsumerState<AdminDashboardPage> createState() => _AdminDashboardPageState();
}

class _AdminDashboardPageState extends ConsumerState<AdminDashboardPage> {
  DashboardPeriod _period = DashboardPeriod.today;
  DashboardSummary? _summary;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final data = await ref.read(dashboardApiProvider).fetchForPeriod(_period);
      if (!mounted) return;
      setState(() {
        _summary = data;
        _loading = false;
      });
    } on DashboardApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Có lỗi xảy ra. Thử lại.';
        _loading = false;
      });
    }
  }

  Future<void> _setPeriod(DashboardPeriod period) async {
    if (period == _period) return;
    setState(() => _period = period);
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final session = ref.watch(authSessionProvider);
    final theme = Theme.of(context);
    // A phone admin has no username or display name — greet them by the number
    // they signed in with instead of the generic "admin".
    final user = session?.user;
    final label = [
      user?.displayName,
      user?.username,
      user?.phoneMasked,
    ].firstWhere(
      (v) => v != null && v.trim().isNotEmpty,
      orElse: () => 'admin',
    )!;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Quản trị cửa hàng'),
        automaticallyImplyLeading: widget.onLoggedOut == null,
        leading: widget.onLoggedOut != null
            ? null
            : IconButton(
                icon: const Icon(Icons.arrow_back),
                onPressed: widget.onBack,
              ),
        actions: [
          IconButton(
            tooltip: 'Tải lại',
            icon: const Icon(Icons.refresh),
            onPressed: _loading ? null : _load,
          ),
          if (widget.onLoggedOut != null)
            IconButton(
              tooltip: 'Đăng xuất',
              icon: const Icon(Icons.logout),
              onPressed: () async {
                await ref.read(authSessionProvider.notifier).logout();
                widget.onLoggedOut!();
              },
            ),
        ],
      ),
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: _load,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(24, 16, 24, 32),
            children: [
              Text(
                'Xin chào, $label',
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                'Dashboard kinh doanh và quản lý cửa hàng Gas Tam Đệ.',
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 20),
              Text(
                'Tổng quan',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 10),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final p in DashboardPeriod.values)
                    FilterChip(
                      label: Text(p.labelVi),
                      selected: _period == p,
                      onSelected: (selected) {
                        if (selected) _setPeriod(p);
                      },
                    ),
                ],
              ),
              const SizedBox(height: 14),
              _buildSummarySection(theme),
              const SizedBox(height: 28),
              Text(
                'Quản lý',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.receipt_long_outlined,
                title: 'Order Desk',
                subtitle: 'Đơn chờ giao — cũ nhất trước',
                onTap: widget.onOpenOrders,
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.inventory_2_outlined,
                title: 'Sản phẩm',
                subtitle: 'Thêm, sửa giá, ẩn / hiện bán',
                onTap: widget.onOpenProducts,
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.local_shipping_outlined,
                title: 'Phí giao hàng',
                subtitle: 'Bật / tắt và bậc theo khoảng cách',
                onTap: widget.onOpenDeliveryFee,
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.store_mall_directory_outlined,
                title: 'Vị trí cửa hàng',
                subtitle: 'Tọa độ gốc và bán kính giao hàng',
                onTap: widget.onOpenStore,
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.tune,
                title: 'Cấu hình Order Desk',
                subtitle: 'Màu thời gian chờ + chu kỳ thông báo giọng nói',
                onTap: widget.onOpenDeskSettings,
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.account_balance_wallet_outlined,
                title: 'Công nợ',
                subtitle: 'Khách còn nợ và tổng công nợ',
                onTap: widget.onOpenDebts,
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.move_to_inbox_outlined,
                title: 'Tồn kho',
                subtitle: 'Xem tồn, nhập / xuất / điều chỉnh',
                onTap: widget.onOpenInventory,
              ),
              const SizedBox(height: 12),
              _AdminNavTile(
                icon: Icons.admin_panel_settings_outlined,
                title: 'Số điện thoại admin',
                subtitle: 'Số nào đăng nhập OTP là vào được trang quản trị',
                onTap: widget.onOpenAdminPhones,
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildSummarySection(ThemeData theme) {
    if (_loading && _summary == null) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 32),
        child: Center(child: CircularProgressIndicator()),
      );
    }

    if (_error != null && _summary == null) {
      return Material(
        color: theme.colorScheme.errorContainer,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                _error!,
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.onErrorContainer,
                ),
              ),
              const SizedBox(height: 12),
              Align(
                alignment: Alignment.centerLeft,
                child: FilledButton(
                  onPressed: _load,
                  child: const Text('Thử lại'),
                ),
              ),
            ],
          ),
        ),
      );
    }

    final s = _summary!;
    final rangeLabel = s.from == s.to ? s.from : '${s.from} → ${s.to}';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              _error!,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.error,
              ),
            ),
          ),
        Text(
          'Kỳ: $rangeLabel (${s.timezone})',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 10),
        LayoutBuilder(
          builder: (context, constraints) {
            final wide = constraints.maxWidth >= 520;
            final tileW =
                wide ? (constraints.maxWidth - 12) / 2 : constraints.maxWidth;
            return Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                SizedBox(
                  width: tileW,
                  child: _MetricTile(
                    icon: Icons.payments_outlined,
                    label: 'Doanh thu',
                    value: formatVnd(s.revenueVnd),
                    emphasize: true,
                  ),
                ),
                SizedBox(
                  width: tileW,
                  child: _MetricTile(
                    icon: Icons.trending_up,
                    label: 'Lợi nhuận',
                    value: formatVnd(s.profitVnd),
                    emphasize: true,
                    valueColor:
                        s.profitVnd < 0 ? theme.colorScheme.error : null,
                  ),
                ),
                SizedBox(
                  width: tileW,
                  child: _MetricTile(
                    icon: Icons.local_shipping_outlined,
                    label: 'Phí giao thu',
                    value: formatVnd(s.deliveryFeeVnd),
                  ),
                ),
                SizedBox(
                  width: tileW,
                  child: _MetricTile(
                    icon: Icons.account_balance_wallet_outlined,
                    label: 'Công nợ',
                    value: formatVnd(s.debtTotal),
                    onTap: widget.onOpenDebts,
                  ),
                ),
                SizedBox(
                  width: tileW,
                  child: _MetricTile(
                    icon: Icons.check_circle_outline,
                    label: 'Đơn hoàn tất',
                    value: '${s.ordersCompleted}',
                  ),
                ),
                SizedBox(
                  width: tileW,
                  child: _MetricTile(
                    icon: Icons.receipt_outlined,
                    label: 'Đơn đặt',
                    value: '${s.ordersPlaced}',
                  ),
                ),
              ],
            );
          },
        ),
        if (s.cogsVnd > 0 || s.revenueVnd > 0) ...[
          const SizedBox(height: 8),
          Text(
            'COGS (giá vốn): ${formatVnd(s.cogsVnd)} · profit = doanh thu − COGS',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
        if (_loading)
          const Padding(
            padding: EdgeInsets.only(top: 8),
            child: LinearProgressIndicator(minHeight: 2),
          ),
      ],
    );
  }
}

class _MetricTile extends StatelessWidget {
  const _MetricTile({
    required this.icon,
    required this.label,
    required this.value,
    this.emphasize = false,
    this.valueColor,
    this.onTap,
  });

  final IconData icon;
  final String label;
  final String value;
  final bool emphasize;
  final Color? valueColor;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final bg = emphasize
        ? theme.colorScheme.primaryContainer
        : theme.colorScheme.surfaceContainerLowest;
    final fg = emphasize
        ? theme.colorScheme.onPrimaryContainer
        : theme.colorScheme.onSurface;
    final muted = emphasize
        ? theme.colorScheme.onPrimaryContainer.withOpacity(0.85)
        : theme.colorScheme.onSurfaceVariant;

    return Material(
      color: bg,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
          child: Row(
            children: [
              Icon(icon,
                  size: 26, color: emphasize ? fg : theme.colorScheme.primary),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      label,
                      style: theme.textTheme.bodyMedium?.copyWith(color: muted),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      value,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w800,
                        color: valueColor ?? fg,
                      ),
                    ),
                  ],
                ),
              ),
              if (onTap != null)
                Icon(
                  Icons.chevron_right,
                  color: muted,
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _AdminNavTile extends StatelessWidget {
  const _AdminNavTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 18),
          child: Row(
            children: [
              Icon(icon, size: 28, color: theme.colorScheme.primary),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              Icon(
                Icons.chevron_right,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
