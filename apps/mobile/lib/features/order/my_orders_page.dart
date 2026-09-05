import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/ui/ui.dart';
import 'order_api.dart';
import 'order_models.dart';

/// Customer — lịch sử đơn của mình + hủy PENDING.
class MyOrdersPage extends ConsumerStatefulWidget {
  const MyOrdersPage({
    super.key,
  });

  @override
  ConsumerState<MyOrdersPage> createState() => _MyOrdersPageState();
}

class _MyOrdersPageState extends ConsumerState<MyOrdersPage> {
  List<AdminOrder>? _items;
  bool _loading = true;
  String? _error;
  String? _busyId;

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
      final items = await ref.read(orderApiProvider).listMyOrders();
      if (!mounted) return;
      setState(() {
        _items = items;
        _loading = false;
      });
    } on OrderApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Không tải được lịch sử đơn.';
        _loading = false;
      });
    }
  }

  Future<void> _cancel(AdminOrder order) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Hủy đơn hàng?'),
        content: Text(
          'Đơn ${order.id} sẽ bị hủy và tồn kho được hoàn lại.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Không'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Hủy đơn'),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    setState(() => _busyId = order.id);
    try {
      await ref.read(orderApiProvider).cancelMyOrder(order.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Đã hủy đơn.')),
      );
      await _load();
    } on OrderApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.displayMessage)),
      );
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Hủy đơn thất bại.')),
      );
    } finally {
      if (mounted) setState(() => _busyId = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Đơn của tôi'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _loading ? null : _load,
          ),
        ],
      ),
      body: SafeArea(child: _body(theme)),
    );
  }

  Widget _body(ThemeData theme) {
    if (_loading && _items == null) return const AppLoading();
    if (_error != null && _items == null) {
      return AppErrorView(message: _error!, onRetry: _load);
    }

    final items = _items ?? const <AdminOrder>[];
    if (items.isEmpty) {
      return const AppEmpty(
        icon: Icons.receipt_long_outlined,
        title: 'Chưa có đơn nào',
        body: 'Đơn bạn đặt sẽ hiện ở đây.',
      );
    }

    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.xxl,
        ),
        itemCount: items.length,
        separatorBuilder: (_, __) => const VGap(AppSpacing.sm),
        itemBuilder: (context, i) => _OrderCard(
          order: items[i],
          busy: _busyId == items[i].id,
          onCancel: () => _cancel(items[i]),
        ),
      ),
    );
  }
}

class _OrderCard extends StatelessWidget {
  const _OrderCard({
    required this.order,
    required this.busy,
    required this.onCancel,
  });

  final AdminOrder order;
  final bool busy;
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final status = order.status.toUpperCase();
    final pending = status == OrderStatus.pending;
    final tone = switch (status) {
      OrderStatus.completed => AppBadgeTone.success,
      OrderStatus.cancelled => AppBadgeTone.danger,
      _ => AppBadgeTone.accent,
    };

    return AppSection(
      children: [
        Row(
          children: [
            AppBadge(orderStatusLabelVi(order.status), tone: tone),
            const Spacer(),
            MoneyText(order.total, emphasis: MoneyEmphasis.total),
          ],
        ),
        const VGap(AppSpacing.md),
        Text(
          order.addressText,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: context.text.bodyLarge,
        ),
        if (order.items.isNotEmpty) ...[
          const VGap(AppSpacing.xs),
          Text(
            order.items.map((i) => '${i.productName} × ${i.qty}').join(' · '),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: context.text.bodySmall?.copyWith(color: p.inkMuted),
          ),
        ],
        const VGap(AppSpacing.xs),
        Text(
          order.createdAt,
          style: context.text.bodySmall?.copyWith(color: p.inkFaint),
        ),
        if (pending) ...[
          const VGap(AppSpacing.md),
          Align(
            alignment: Alignment.centerRight,
            child: AppButton.danger(
              label: 'Hủy đơn',
              loading: busy,
              onPressed: busy ? null : onCancel,
            ),
          ),
        ],
      ],
    );
  }
}
