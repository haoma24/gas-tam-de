import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';
import '../auth/auth_session.dart';

/// Admin «Cài đặt» tab — the shop configuration that used to be six of the
/// nine tiles on the old dashboard.
class AdminSettingsPage extends ConsumerWidget {
  const AdminSettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final session = ref.watch(authSessionProvider);
    final user = session?.user;
    final who = [
      user?.displayName,
      user?.username,
      user?.phoneMasked,
    ].firstWhere((v) => v != null && v.trim().isNotEmpty,
        orElse: () => 'admin');

    return AppScaffold(
      title: 'Cài đặt',
      showBack: false,
      padBody: false,
      body: ListView(
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.xxl,
        ),
        children: [
          AppSection(
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Đang đăng nhập',
                          style: context.text.bodySmall?.copyWith(
                            color: context.palette.inkMuted,
                          ),
                        ),
                        const VGap(2),
                        Text(who!, style: context.text.titleSmall),
                      ],
                    ),
                  ),
                  AppButton.secondary(
                    label: 'Đăng xuất',
                    onPressed: () async {
                      await ref.read(authSessionProvider.notifier).logout();
                      if (context.mounted) context.go('/welcome');
                    },
                  ),
                ],
              ),
            ],
          ),
          const VGap(AppSpacing.xl),
          const AppThemeModeSection(),
          const VGap(AppSpacing.xl),
          const AppSectionTitle('Cửa hàng'),
          AppNavTile(
            icon: Icons.inventory_2_outlined,
            title: 'Sản phẩm',
            subtitle: 'Thêm, sửa giá, ẩn / hiện bán',
            onTap: () => context.push('/admin/products'),
          ),
          const VGap(AppSpacing.sm),
          AppNavTile(
            icon: Icons.local_shipping_outlined,
            title: 'Phí giao hàng',
            subtitle: 'Bật / tắt và bậc theo khoảng cách',
            onTap: () => context.push('/admin/delivery-fee'),
          ),
          const VGap(AppSpacing.sm),
          AppNavTile(
            icon: Icons.store_mall_directory_outlined,
            title: 'Vị trí cửa hàng',
            subtitle: 'Tọa độ gốc và bán kính giao hàng',
            onTap: () => context.push('/admin/store'),
          ),
          const VGap(AppSpacing.sm),
          AppNavTile(
            icon: Icons.tune_rounded,
            title: 'Cấu hình Order Desk',
            subtitle: 'Màu thời gian chờ + chu kỳ thông báo giọng nói',
            onTap: () => context.push('/admin/desk-settings'),
          ),
          const VGap(AppSpacing.xl),
          const AppSectionTitle('Quản trị'),
          AppNavTile(
            icon: Icons.manage_accounts_outlined,
            title: 'Tài khoản quản lý',
            subtitle: 'Tạo tài khoản, đổi tên đăng nhập và mật khẩu',
            onTap: () => context.push('/admin/admin-accounts'),
          ),
          const VGap(AppSpacing.sm),
          AppNavTile(
            icon: Icons.admin_panel_settings_outlined,
            title: 'Số điện thoại admin',
            subtitle: 'Số nào đăng nhập là vào được trang quản trị',
            onTap: () => context.push('/admin/admin-phones'),
          ),
        ],
      ),
    );
  }
}
