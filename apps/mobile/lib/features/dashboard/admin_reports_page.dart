import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/phone_link.dart';
import '../../core/ui/ui.dart';
import '../billing/billing_api.dart';
import '../billing/billing_models.dart';
import '../order/customer_stats_models.dart';
import '../order/order_api.dart';
import 'dashboard_api.dart';
import 'dashboard_models.dart';

/// Số khách hiện sẵn trong mục «Khách hàng» trước khi bấm «Xem tất cả».
const int kCustomerPreviewCount = 10;

/// Admin «Báo cáo» tab — period metrics, per-customer activity, and the
/// outstanding-debt list.
///
/// Merges what used to be two destinations (the dashboard summary at `/admin`
/// and `/admin/debts`): debt is a number the owner reads alongside revenue, not
/// a separate errand. The customer breakdown answers «khách nào đã đặt bao
/// nhiêu đơn», which the debt list alone could not.
class AdminReportsPage extends ConsumerStatefulWidget {
  const AdminReportsPage({super.key});

  @override
  ConsumerState<AdminReportsPage> createState() => _AdminReportsPageState();
}

class _AdminReportsPageState extends ConsumerState<AdminReportsPage> {
  DashboardPeriod _period = DashboardPeriod.today;
  DashboardSummary? _summary;
  DebtsList? _debts;
  CustomerStatsList? _customers;
  bool _showAllCustomers = false;
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
    final range = rangeForPeriod(_period);
    try {
      // Debt is an aggregate independent of the period, so all three land
      // together rather than making the owner wait on three separate spinners.
      final results = await Future.wait([
        ref.read(dashboardApiProvider).fetchForPeriod(_period),
        ref.read(billingApiProvider).listDebts(),
        ref
            .read(orderApiProvider)
            .listCustomerStats(from: range.from, to: range.to),
      ]);
      if (!mounted) return;
      setState(() {
        _summary = results[0] as DashboardSummary;
        _debts = results[1] as DebtsList;
        _customers = results[2] as CustomerStatsList;
        _loading = false;
      });
    } on DashboardApiException catch (e) {
      _fail(e.displayMessage);
    } on BillingApiException catch (e) {
      _fail(e.displayMessage);
    } on OrderApiException catch (e) {
      _fail(e.displayMessage);
    } catch (_) {
      _fail('Có lỗi xảy ra. Thử lại.');
    }
  }

  void _fail(String message) {
    if (!mounted) return;
    setState(() {
      _error = message;
      _loading = false;
    });
  }

  Future<void> _setPeriod(DashboardPeriod period) async {
    if (period == _period) return;
    setState(() {
      _period = period;
      _showAllCustomers = false;
    });
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    return AppScaffold(
      title: 'Báo cáo',
      showBack: false,
      padBody: false,
      actions: [
        IconButton(
          tooltip: 'Tải lại',
          icon: const Icon(Icons.refresh),
          onPressed: _loading ? null : _load,
        ),
      ],
      body: _body(),
    );
  }

  Widget _body() {
    if (_loading && _summary == null) return const AppLoading();
    if (_error != null && _summary == null) {
      return AppErrorView(message: _error!, onRetry: _load);
    }

    final s = _summary!;
    final debts = _debts;

    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.xxl,
        ),
        children: [
          SegmentedButton<DashboardPeriod>(
            segments: [
              for (final p in DashboardPeriod.values)
                ButtonSegment(value: p, label: Text(p.labelVi)),
            ],
            selected: {_period},
            showSelectedIcon: false,
            onSelectionChanged: (set) => _setPeriod(set.first),
          ),
          const VGap(AppSpacing.md),
          Text(
            'Kỳ: ${s.from == s.to ? s.from : '${s.from} → ${s.to}'} '
            '(${s.timezone})',
            style: context.text.bodySmall?.copyWith(
              color: context.palette.inkMuted,
            ),
          ),
          if (_loading) ...[
            const VGap(AppSpacing.sm),
            const LinearProgressIndicator(minHeight: 2),
          ],
          const VGap(AppSpacing.lg),
          _MetricGrid(summary: s),
          if (s.revenueVnd > 0 && s.cogsVnd <= 0)
            // Without a purchase price there is no COGS, and profit silently
            // equals revenue — exactly the number the owner reported as wrong.
            // Say so instead of showing a flattering total.
            const Padding(
              padding: EdgeInsets.only(top: AppSpacing.md),
              child: _CogsWarning(),
            )
          else if (s.cogsVnd > 0) ...[
            const VGap(AppSpacing.md),
            Text(
              'Giá vốn (COGS): ${formatVnd(s.cogsVnd)} · '
              'lợi nhuận = doanh thu − giá vốn',
              style: context.text.bodySmall?.copyWith(
                color: context.palette.inkMuted,
              ),
            ),
          ],
          const VGap(AppSpacing.xl),
          _CustomerSection(
            stats: _customers,
            showAll: _showAllCustomers,
            onToggleShowAll: () =>
                setState(() => _showAllCustomers = !_showAllCustomers),
          ),
          const VGap(AppSpacing.xl),
          AppSectionTitle(
            'Công nợ',
            trailing: debts == null
                ? null
                : Text(
                    '${debts.count} khách',
                    style: context.text.bodySmall?.copyWith(
                      color: context.palette.inkMuted,
                    ),
                  ),
          ),
          if (debts == null)
            const AppLoading()
          else if (debts.items.isEmpty)
            const AppSection(
              children: [
                AppEmpty(
                  icon: Icons.check_circle_outline,
                  title: 'Không có công nợ',
                  body: 'Khách đã thanh toán đủ sẽ không hiện ở đây.',
                ),
              ],
            )
          else
            AppSection(
              padding: const EdgeInsets.symmetric(
                horizontal: AppSpacing.lg,
                vertical: AppSpacing.xs,
              ),
              children: [
                MoneyRow(
                  label: 'Tổng công nợ',
                  amount: debts.totalBalance,
                  emphasis: MoneyEmphasis.total,
                  valueColor: context.palette.danger,
                ),
                const Divider(),
                for (final item in debts.items) _DebtRow(item: item),
              ],
            ),
        ],
      ),
    );
  }
}

/// Warning shown when the period has revenue but no cost of goods.
class _CogsWarning extends StatelessWidget {
  const _CogsWarning();

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Container(
      padding: const EdgeInsets.all(AppSpacing.lg),
      decoration: BoxDecoration(
        color: p.warning.withValues(alpha: 0.08),
        borderRadius: AppRadius.md,
        border: Border.all(color: p.warning.withValues(alpha: 0.30)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.info_outline, size: 18, color: p.warning),
          const HGap(AppSpacing.sm),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Lợi nhuận đang bằng doanh thu',
                  style: context.text.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const VGap(AppSpacing.xs),
                Text(
                  'Kỳ này chưa có giá nhập nên giá vốn = 0. Vào tab «Kho», '
                  'nhập giá nhập cho từng sản phẩm — các đơn sau đó sẽ tính '
                  'lợi nhuận = giá bán − giá nhập.',
                  style: context.text.bodySmall?.copyWith(color: p.inkMuted),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// «Khách hàng» — how many orders each customer placed in the period.
class _CustomerSection extends StatelessWidget {
  const _CustomerSection({
    required this.stats,
    required this.showAll,
    required this.onToggleShowAll,
  });

  final CustomerStatsList? stats;
  final bool showAll;
  final VoidCallback onToggleShowAll;

  @override
  Widget build(BuildContext context) {
    final data = stats;
    if (data == null) {
      return const Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [AppSectionTitle('Khách hàng'), AppLoading()],
      );
    }

    if (data.isEmpty) {
      return const Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppSectionTitle('Khách hàng'),
          AppSection(
            children: [
              AppEmpty(
                icon: Icons.people_outline,
                title: 'Chưa có khách đặt đơn',
                body: 'Kỳ này chưa có đơn nào.',
              ),
            ],
          ),
        ],
      );
    }

    final visible = showAll
        ? data.customers
        : data.customers.take(kCustomerPreviewCount).toList();
    final hidden = data.customers.length - visible.length;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AppSectionTitle(
          'Khách hàng',
          trailing: Text(
            '${data.total} khách',
            style: context.text.bodySmall?.copyWith(
              color: context.palette.inkMuted,
            ),
          ),
        ),
        AppSection(
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.lg,
            vertical: AppSpacing.xs,
          ),
          children: [
            for (var i = 0; i < visible.length; i++) ...[
              if (i > 0) const Divider(height: 1),
              _CustomerRow(stat: visible[i]),
            ],
            if (hidden > 0 || showAll) ...[
              const Divider(height: 1),
              Align(
                alignment: Alignment.center,
                child: TextButton(
                  onPressed: onToggleShowAll,
                  child: Text(showAll ? 'Thu gọn' : 'Xem tất cả ($hidden nữa)'),
                ),
              ),
            ],
          ],
        ),
      ],
    );
  }
}

class _CustomerRow extends StatelessWidget {
  const _CustomerRow({required this.stat});

  final CustomerStat stat;

  Future<void> _call(BuildContext context) async {
    final result = await dialPhone(stat.customerPhone);
    if (!context.mounted || result.isOk) return;
    ScaffoldMessenger.maybeOf(context)
      ?..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Text(result.errorMessage!),
          behavior: SnackBarBehavior.floating,
        ),
      );
  }

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final callable = stat.customerPhone.isNotEmpty;

    // «7 đơn» is the headline; the completed/cancelled split explains it.
    final detail = StringBuffer('${stat.ordersCompleted} hoàn tất');
    if (stat.ordersPending > 0) detail.write(' · ${stat.ordersPending} chờ');
    if (stat.ordersCancelled > 0) detail.write(' · ${stat.ordersCancelled} hủy');

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.md),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  stat.displayName,
                  style: context.text.bodyLarge?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const VGap(AppSpacing.xs),
                InkWell(
                  onTap: callable ? () => _call(context) : null,
                  borderRadius: AppRadius.sm,
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        stat.displayPhone,
                        style: context.text.bodySmall?.copyWith(
                          color: callable
                              ? Theme.of(context).colorScheme.primary
                              : p.inkMuted,
                        ),
                      ),
                      if (callable) ...[
                        const HGap(AppSpacing.xs),
                        Icon(
                          Icons.call,
                          size: 13,
                          color: Theme.of(context).colorScheme.primary,
                        ),
                      ],
                    ],
                  ),
                ),
                const VGap(AppSpacing.xs),
                Text(
                  detail.toString(),
                  style: context.text.bodySmall?.copyWith(color: p.inkMuted),
                ),
              ],
            ),
          ),
          const HGap(AppSpacing.md),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                '${stat.ordersTotal} đơn',
                style: context.text.bodyLarge?.copyWith(
                  fontWeight: FontWeight.w600,
                  fontFeatures: kTabularFigures,
                ),
              ),
              const VGap(AppSpacing.xs),
              Text(
                formatVnd(stat.spentVnd),
                style: context.text.bodySmall?.copyWith(
                  color: p.inkMuted,
                  fontFeatures: kTabularFigures,
                ),
              ),
              if (stat.debtVnd > 0) ...[
                const VGap(AppSpacing.xs),
                Text(
                  'Nợ ${formatVnd(stat.debtVnd)}',
                  style: context.text.bodySmall?.copyWith(
                    color: p.danger,
                    fontFeatures: kTabularFigures,
                  ),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }
}

class _MetricGrid extends StatelessWidget {
  const _MetricGrid({required this.summary});

  final DashboardSummary summary;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final tiles = <_Metric>[
      _Metric('Doanh thu', formatVnd(summary.revenueVnd), primary: true),
      _Metric(
        'Lợi nhuận',
        formatVnd(summary.profitVnd),
        primary: true,
        color: summary.profitVnd < 0 ? p.danger : null,
      ),
      _Metric('Phí giao thu', formatVnd(summary.deliveryFeeVnd)),
      _Metric(
        'Công nợ',
        formatVnd(summary.debtTotal),
        color: summary.debtTotal > 0 ? p.danger : null,
      ),
      _Metric('Đơn hoàn tất', '${summary.ordersCompleted}'),
      _Metric('Đơn đặt', '${summary.ordersPlaced}'),
    ];

    return LayoutBuilder(
      builder: (context, constraints) {
        final columns = constraints.maxWidth >= 640
            ? 3
            : constraints.maxWidth >= 380
                ? 2
                : 1;
        final width =
            (constraints.maxWidth - AppSpacing.md * (columns - 1)) / columns;
        return Wrap(
          spacing: AppSpacing.md,
          runSpacing: AppSpacing.md,
          children: [
            for (final t in tiles)
              SizedBox(width: width, child: _MetricTile(metric: t)),
          ],
        );
      },
    );
  }
}

class _Metric {
  const _Metric(this.label, this.value, {this.primary = false, this.color});

  final String label;
  final String value;
  final bool primary;
  final Color? color;
}

class _MetricTile extends StatelessWidget {
  const _MetricTile({required this.metric});

  final _Metric metric;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Container(
      padding: const EdgeInsets.all(AppSpacing.lg),
      decoration: BoxDecoration(
        color: p.surface,
        borderRadius: AppRadius.md,
        border: Border.all(color: p.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            metric.label,
            style: context.text.bodySmall?.copyWith(color: p.inkMuted),
          ),
          const VGap(AppSpacing.xs),
          Text(
            metric.value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: (metric.primary
                    ? context.text.titleLarge
                    : context.text.titleMedium)
                ?.copyWith(
              color: metric.color ?? p.ink,
              fontFeatures: kTabularFigures,
            ),
          ),
        ],
      ),
    );
  }
}

class _DebtRow extends StatelessWidget {
  const _DebtRow({required this.item});

  final DebtItem item;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.md),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  item.phoneMasked.isEmpty
                      ? item.customerKey
                      : item.phoneMasked,
                  style: context.text.bodyLarge?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                ),
                if (item.updatedAt.isNotEmpty)
                  Text(
                    item.updatedAt,
                    style: context.text.bodySmall?.copyWith(color: p.inkMuted),
                  ),
              ],
            ),
          ),
          const HGap(AppSpacing.md),
          MoneyText(item.balance, color: p.danger),
        ],
      ),
    );
  }
}
