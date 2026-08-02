import 'package:flutter/material.dart';

/// Multi-platform CTA shell (T9.2.4): brand Gas Tam Đệ + customer vs admin entry.
/// Same widget tree on Web, Android, and iOS.
class HomePage extends StatelessWidget {
  const HomePage({
    super.key,
    required this.onStartOrder,
    this.onAdminLogin,
    this.onMyOrders,
  });

  final VoidCallback onStartOrder;
  final VoidCallback? onAdminLogin;
  final VoidCallback? onMyOrders;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Spacer(),
              Text(
                'Gas Tam Đệ',
                textAlign: TextAlign.center,
                style: theme.textTheme.displaySmall?.copyWith(
                  fontWeight: FontWeight.w800,
                  letterSpacing: -0.5,
                ),
              ),
              const SizedBox(height: 12),
              Text(
                'Giao gas tận nơi — nhanh, rõ phí, đúng địa chỉ.',
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const Spacer(),
              FilledButton(
                onPressed: onStartOrder,
                style: FilledButton.styleFrom(
                  minimumSize: const Size.fromHeight(56),
                  textStyle: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                child: const Text('Đặt giao gas'),
              ),
              if (onMyOrders != null) ...[
                const SizedBox(height: 12),
                OutlinedButton(
                  onPressed: onMyOrders,
                  style: OutlinedButton.styleFrom(
                    minimumSize: const Size.fromHeight(48),
                  ),
                  child: const Text('Đơn của tôi'),
                ),
              ],
              const SizedBox(height: 12),
              TextButton(
                onPressed: onAdminLogin,
                child: const Text('Dành cho cửa hàng'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
