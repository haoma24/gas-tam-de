import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';
import 'order_models.dart';

/// Confirmation after a successful place-order.
class OrderSuccessPage extends StatelessWidget {
  const OrderSuccessPage({super.key, required this.order});

  final PlacedOrder order;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final shortId = order.id.length > 8
        ? order.id.substring(0, 8).toUpperCase()
        : order.id.toUpperCase();

    return AppScaffold(
      showBack: false,
      padBody: false,
      body: ListView(
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.lg,
          AppSpacing.xxl,
          AppSpacing.lg,
          AppSpacing.xxl,
        ),
        children: [
          Icon(Icons.check_circle_rounded, size: 40, color: p.success),
          const VGap(AppSpacing.lg),
          Text('Đã nhận đơn', style: context.text.headlineSmall),
          const VGap(AppSpacing.sm),
          Text(
            'Cửa hàng sẽ liên hệ để giao gas. Giữ máy để nhận cuộc gọi.',
            style: context.text.bodyLarge?.copyWith(color: p.inkMuted),
          ),
          const VGap(AppSpacing.xl),
          AppSection(
            title: 'Đơn #$shortId',
            children: [
              AppDataRow(
                label: 'Trạng thái',
                value: '',
                valueWidget: AppBadge(
                  orderStatusLabelVi(order.status),
                  tone: AppBadgeTone.neutral,
                ),
              ),
              AppDataRow(label: 'Người nhận', value: order.customerName),
              if (order.phoneMasked.isNotEmpty)
                AppDataRow(label: 'Số điện thoại', value: order.phoneMasked),
              AppDataRow(
                label: 'Địa chỉ',
                value: order.addressText,
                stacked: true,
              ),
            ],
          ),
          const VGap(AppSpacing.lg),
          AppSection(
            title: 'Chi tiết',
            children: [
              for (final item in order.items)
                MoneyRow(
                  label: '${item.productName} × ${item.qty}',
                  amount: item.lineTotal,
                ),
              const Divider(),
              MoneyRow(label: 'Tạm tính', amount: order.subtotal),
              MoneyRow(label: 'Phí giao hàng', amount: order.deliveryFee),
              MoneyRow(
                label: 'Tổng cộng',
                amount: order.total,
                emphasis: MoneyEmphasis.total,
              ),
            ],
          ),
        ],
      ),
      bottomBar: AppButton.primary(
        label: 'Về trang chủ',
        expand: true,
        onPressed: () => context.go('/'),
      ),
    );
  }
}
