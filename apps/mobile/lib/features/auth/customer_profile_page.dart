import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';
import 'auth_models.dart';
import 'auth_session.dart';
import 'google_auth.dart';
import 'me_api.dart';
import 'phone_utils.dart';

/// Customer personal profile — rich header + form + actions.
class CustomerProfilePage extends ConsumerStatefulWidget {
  const CustomerProfilePage({super.key});

  @override
  ConsumerState<CustomerProfilePage> createState() =>
      _CustomerProfilePageState();
}

class _CustomerProfilePageState extends ConsumerState<CustomerProfilePage> {
  final _nameController = TextEditingController();
  final _phoneController = TextEditingController();
  bool _saving = false;
  bool _loggingOut = false;
  bool _editingName = false;
  bool _editingPhone = false;
  String? _error;
  String? _savedHint;
  CustomerProfile? _profile;

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  @override
  void dispose() {
    _nameController.dispose();
    _phoneController.dispose();
    super.dispose();
  }

  Future<void> _bootstrap() async {
    final cached = await ref.read(customerProfileProvider.future);
    if (!mounted) return;
    if (cached != null) {
      setState(() {
        _profile = cached;
        _nameController.text = cached.fullName ?? '';
        _editingPhone = cached.phoneMasked.isEmpty;
      });
    } else {
      await _reload();
    }
  }

  Future<void> _reload() async {
    setState(() => _error = null);
    try {
      final p = await ref.read(meApiProvider).getMe();
      if (!mounted) return;
      setState(() {
        _profile = p;
        _nameController.text = p.fullName ?? '';
        _editingPhone = p.phoneMasked.isEmpty;
      });
      ref.invalidate(customerProfileProvider);
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.displayMessage);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Không tải được hồ sơ.');
    }
  }

  Future<void> _saveName() async {
    final name = _nameController.text.trim();
    if (name.isEmpty) {
      setState(() => _error = 'Nhập họ tên để lưu.');
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
      _savedHint = null;
    });
    try {
      final p = await ref.read(meApiProvider).patchFullName(name);
      if (!mounted) return;
      setState(() {
        _profile = p;
        _nameController.text = p.fullName ?? name;
        _savedHint = 'Đã lưu hồ sơ.';
        _editingName = false;
      });
      ref.invalidate(customerProfileProvider);
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.displayMessage);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Không lưu được. Thử lại.');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _savePhone() async {
    final phone = _phoneController.text.trim();
    if (!isValidVnMobile(phone)) {
      setState(() => _error = 'SĐT di động không hợp lệ (VD: 0901234567).');
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
      _savedHint = null;
    });
    try {
      final p = await ref.read(meApiProvider).patchPhone(phone);
      // Rotate immediately so the new contact number is carried by the JWT
      // used when creating an order.
      await ref.read(authSessionProvider.notifier).refresh();
      if (!mounted) return;
      setState(() {
        _profile = p;
        _phoneController.clear();
        _editingPhone = false;
        _savedHint = 'Đã lưu SĐT liên hệ.';
      });
      ref.invalidate(customerProfileProvider);
    } on AuthApiException catch (e) {
      if (mounted) setState(() => _error = e.displayMessage);
    } catch (_) {
      if (mounted) setState(() => _error = 'Không lưu được. Thử lại.');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _logout() async {
    setState(() => _loggingOut = true);
    await ref.read(googleAuthProvider.notifier).logout();
    if (!mounted) return;
    if (context.mounted) context.go('/welcome');
  }

  static String _initial(String name, String phone) {
    final n = name.trim();
    if (n.isNotEmpty) {
      return String.fromCharCodes(n.runes.take(1)).toUpperCase();
    }
    if (phone.isNotEmpty && phone != '—') {
      return String.fromCharCodes(phone.runes.take(1));
    }
    return '?';
  }

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final session = ref.watch(authSessionProvider);
    final phone = _profile?.phoneMasked.isNotEmpty == true
        ? _profile!.phoneMasked
        : session?.user.phoneMasked.isNotEmpty == true
            ? session!.user.phoneMasked
            : '—';
    final displayName = _profile?.fullName?.trim().isNotEmpty == true
        ? _profile!.fullName!
        : null;
    final email = _profile?.email ?? session?.user.email;
    final identityLabel = email?.trim().isNotEmpty == true ? email! : phone;
    final initial = _initial(_nameController.text, phone);

    return Scaffold(
      backgroundColor: p.bg,
      appBar: AppBar(title: const Text('Hồ sơ')),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.lg,
            AppSpacing.sm,
            AppSpacing.lg,
            AppSpacing.xxl,
          ),
          children: [
            Row(
              children: [
                Container(
                  width: 56,
                  height: 56,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: p.surfaceSubtle,
                    shape: BoxShape.circle,
                    border: Border.all(color: p.border),
                  ),
                  child: Text(initial, style: context.text.titleLarge),
                ),
                const HGap(AppSpacing.lg),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        displayName ?? 'Khách hàng',
                        style: context.text.titleMedium,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      const VGap(2),
                      Text(
                        identityLabel,
                        style: context.text.bodySmall?.copyWith(
                          color: p.inkMuted,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ),
                ),
              ],
            ),
            if (_error != null) ...[
              const VGap(AppSpacing.lg),
              AppErrorBanner(message: _error!),
            ],
            if (_savedHint != null) ...[
              const VGap(AppSpacing.lg),
              Row(
                children: [
                  Icon(Icons.check_circle_outline, size: 16, color: p.success),
                  const HGap(AppSpacing.sm),
                  Text(
                    _savedHint!,
                    style: context.text.bodyMedium?.copyWith(color: p.success),
                  ),
                ],
              ),
            ],
            const VGap(AppSpacing.xl),
            AppSection(
              title: 'Họ tên',
              trailing: _editingName
                  ? null
                  : AppButton.text(
                      label: displayName == null ? 'Thêm' : 'Sửa',
                      onPressed: () => setState(() => _editingName = true),
                    ),
              children: [
                if (!_editingName)
                  Text(
                    displayName ?? 'Chưa có tên',
                    style: context.text.bodyLarge?.copyWith(
                      color: displayName == null ? p.inkMuted : p.ink,
                    ),
                  )
                else ...[
                  AppTextField(
                    controller: _nameController,
                    hint: 'Nguyễn Văn A',
                    enabled: !_saving,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _saveName(),
                  ),
                  const VGap(AppSpacing.md),
                  Row(
                    children: [
                      AppButton.primary(
                        label: 'Lưu',
                        loading: _saving,
                        onPressed: _saving ? null : _saveName,
                      ),
                      const HGap(AppSpacing.sm),
                      AppButton.text(
                        label: 'Hủy',
                        onPressed: _saving
                            ? null
                            : () => setState(() {
                                  _editingName = false;
                                  _nameController.text =
                                      _profile?.fullName ?? '';
                                }),
                      ),
                    ],
                  ),
                ],
              ],
            ),
            const VGap(AppSpacing.lg),
            AppSection(
              title: 'Số điện thoại liên hệ',
              trailing: _editingPhone
                  ? null
                  : AppButton.text(
                      label: 'Đổi',
                      onPressed: () => setState(() => _editingPhone = true),
                    ),
              children: [
                if (!_editingPhone)
                  Text(phone, style: context.text.bodyLarge)
                else ...[
                  AppTextField(
                    controller: _phoneController,
                    hint: '0901234567',
                    enabled: !_saving,
                    keyboardType: TextInputType.phone,
                    textInputAction: TextInputAction.done,
                    inputFormatters: [
                      FilteringTextInputFormatter.digitsOnly,
                      LengthLimitingTextInputFormatter(11),
                    ],
                    helper: 'Cửa hàng gọi số này khi giao gas.',
                    onSubmitted: (_) => _savePhone(),
                  ),
                  const VGap(AppSpacing.md),
                  Row(
                    children: [
                      AppButton.primary(
                        label: 'Lưu SĐT',
                        loading: _saving,
                        onPressed: _saving ? null : _savePhone,
                      ),
                      if (phone != '—') ...[
                        const HGap(AppSpacing.sm),
                        AppButton.text(
                          label: 'Hủy',
                          onPressed: _saving
                              ? null
                              : () => setState(() {
                                    _editingPhone = false;
                                    _phoneController.clear();
                                  }),
                        ),
                      ],
                    ],
                  ),
                ],
              ],
            ),
            const VGap(AppSpacing.xl),
            AppNavTile(
              icon: Icons.receipt_long_outlined,
              title: 'Đơn hàng của tôi',
              onTap: () => context.go('/orders'),
            ),
            const VGap(AppSpacing.sm),
            AppNavTile(
              icon: Icons.logout_rounded,
              title: 'Đăng xuất',
              destructive: true,
              trailing: _loggingOut ? const AppInlineSpinner(size: 16) : null,
              onTap: _loggingOut ? null : _logout,
            ),
          ],
        ),
      ),
    );
  }
}
