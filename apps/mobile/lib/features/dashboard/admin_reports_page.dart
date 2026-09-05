import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/ui/ui.dart';
import '../billing/billing_api.dart';
import '../billing/billing_models.dart';
import 'dashboard_api.dart';
import 'dashboard_models.dart';

/// Admin «Báo cáo» tab — period metrics plus the outstanding-debt list.
///
/// Merges what used to be two destinations (the dashboard summary at `/admin`
/// and `/admin/debts`): debt is a number the owner reads alongside revenue, not
/// a separate errand.
class AdminReportsPage extends ConsumerStatefulWidget {
  const AdminReportsPage({super.key});

  @override
  ConsumerState<AdminReportsPage> createState() => _AdminReportsPageState();
}

class _AdminReportsPageState extends ConsumerState<AdminReportsPage> {
  DashboardPeriod _period = DashboardPeriod.today;
  DashboardSummary? _summary;
  DebtsList? _debts;
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
      // Debt is an aggregate independent of the period, so both land together.
      final results = await Future.wait([
        ref.read(dashboardApiProvider).fetchForPeriod(_period),
        ref.read(billingApiProvider).listDebts(),
      ]);
      if (!mounted) return;
      setState(() {
        _summary = results[0] as DashboardSummary;
        _debts = results[1] as DebtsList;
        _loading = false;
      });
    } on DashboardApiException catch (e) {
      _fail(e.displayMessage);
    } on BillingApiException catch (e) {
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
    setState(() => _period = period);
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
          if (s.cogsVnd > 0 || s.revenueVnd > 0) ...[
            const VGap(AppSpacing.md),
            Text(
              'COGS (giá vốn): ${formatVnd(s.cogsVnd)} · '
              'lợi nhuận = doanh thu − COGS',
              style: context.text.bodySmall?.copyWith(
                color: context.palette.inkMuted,
              ),
            ),
          ],
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
