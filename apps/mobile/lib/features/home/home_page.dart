import 'package:flutter/material.dart';

/// Guest landing — brand Gas Tam Đệ + CTA đặt gas (OTP). No admin entry CTA.
class HomePage extends StatelessWidget {
  const HomePage({
    super.key,
    required this.onStartOrder,
    this.onMyOrders,
  });

  final VoidCallback onStartOrder;
  final VoidCallback? onMyOrders;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    const onHero = Colors.white;

    return Scaffold(
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Expanded(
            flex: 5,
            child: Container(
              width: double.infinity,
              decoration: const BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [
                    Color(0xFF1C1917),
                    Color(0xFF44403C),
                    Color(0xFF9A3412),
                  ],
                  stops: [0.0, 0.55, 1.0],
                ),
              ),
              child: SafeArea(
                bottom: false,
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(28, 40, 28, 32),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Spacer(),
                      Text(
                        'Gas Tam Đệ',
                        style: theme.textTheme.displayMedium?.copyWith(
                          color: onHero,
                          fontWeight: FontWeight.w800,
                          letterSpacing: -1.4,
                          height: 1.02,
                        ),
                      ),
                      const SizedBox(height: 14),
                      Text(
                        'Giao gas tận nơi — nhanh, rõ phí, đúng địa chỉ.',
                        style: theme.textTheme.titleMedium?.copyWith(
                          color: onHero.withValues(alpha: 0.88),
                          height: 1.35,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      const SizedBox(height: 8),
                    ],
                  ),
                ),
              ),
            ),
          ),
          Expanded(
            flex: 4,
            child: SafeArea(
              top: false,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'Đặt bình gas chỉ với số điện thoại.',
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
                        child: const Text('Đã có tài khoản? Xem đơn của tôi'),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
