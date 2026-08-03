import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import 'auth_models.dart';
import 'auth_session.dart';
import 'me_api.dart';

/// Customer personal profile — rich header + form + actions.
class CustomerProfilePage extends ConsumerStatefulWidget {
  const CustomerProfilePage({
    super.key,
    required this.onBack,
    required this.onMyOrders,
    required this.onLoggedOut,
  });

  final VoidCallback onBack;
  final VoidCallback onMyOrders;
  final VoidCallback onLoggedOut;

  @override
  ConsumerState<CustomerProfilePage> createState() =>
      _CustomerProfilePageState();
}

class _CustomerProfilePageState extends ConsumerState<CustomerProfilePage> {
  final _nameController = TextEditingController();
  bool _saving = false;
  bool _loggingOut = false;
  bool _editingName = false;
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
    super.dispose();
  }

  Future<void> _bootstrap() async {
    final cached = await ref.read(customerProfileProvider.future);
    if (!mounted) return;
    if (cached != null) {
      setState(() {
        _profile = cached;
        _nameController.text = cached.fullName ?? '';
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

  Future<void> _logout() async {
    setState(() => _loggingOut = true);
    await ref.read(authSessionProvider.notifier).clear();
    if (!mounted) return;
    widget.onLoggedOut();
  }

  static String _initial(String name, String phone) {
    final n = name.trim();
    if (n.isNotEmpty) return String.fromCharCodes(n.runes.take(1)).toUpperCase();
    if (phone.isNotEmpty && phone != '—')
      return String.fromCharCodes(phone.runes.take(1));
    return '?';
  }

  @override
  Widget build(BuildContext context) {
    final session = ref.watch(authSessionProvider);
    final phone = _profile?.phoneMasked.isNotEmpty == true
        ? _profile!.phoneMasked
        : (session?.user.phoneMasked ?? '—');
    final displayName = _profile?.fullName?.trim().isNotEmpty == true
        ? _profile!.fullName!
        : null;
    final initial = _initial(_nameController.text, phone);

    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle.light,
      child: Scaffold(
        backgroundColor: AppColors.surface0,
        body: Column(
          children: [
            // ── Rich header ──────────────────────────────────────
            Container(
              decoration: const BoxDecoration(gradient: AppColors.heroGradient),
              child: Stack(
                children: [
                  const Positioned.fill(
                    child: CustomPaint(painter: FlameAmbientPainter()),
                  ),
                  SafeArea(
                    bottom: false,
                    child: Column(
                      children: [
                        // Back bar
                        Padding(
                          padding:
                              const EdgeInsets.fromLTRB(4, 4, 16, 0),
                          child: Row(
                            children: [
                              IconButton(
                                icon: const Icon(
                                    Icons.arrow_back_ios_new_rounded,
                                    color: AppColors.onDark,
                                    size: 20),
                                onPressed:
                                    _loggingOut ? null : widget.onBack,
                              ),
                              const Spacer(),
                            ],
                          ),
                        ),
                        const SizedBox(height: 12),
                        // Avatar
                        Container(
                          width: 80,
                          height: 80,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            gradient: const LinearGradient(
                              colors: [AppColors.amber, AppColors.fire],
                              begin: Alignment.topLeft,
                              end: Alignment.bottomRight,
                            ),
                            boxShadow: [
                              BoxShadow(
                                color:
                                    AppColors.fire.withValues(alpha: 0.45),
                                blurRadius: 20,
                                offset: const Offset(0, 6),
                              ),
                            ],
                          ),
                          alignment: Alignment.center,
                          child: Text(
                            initial,
                            style: const TextStyle(
                              color: AppColors.obsidian,
                              fontSize: 30,
                              fontWeight: FontWeight.w900,
                            ),
                          ),
                        ),
                        const SizedBox(height: 12),
                        // Name
                        Text(
                          displayName ?? 'Tài khoản khách',
                          style: const TextStyle(
                            color: AppColors.onDark,
                            fontSize: 20,
                            fontWeight: FontWeight.w800,
                            letterSpacing: -0.3,
                          ),
                        ),
                        const SizedBox(height: 4),
                        // Phone
                        Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            const Icon(Icons.phone_outlined,
                                color: AppColors.amber, size: 14),
                            const SizedBox(width: 5),
                            Text(
                              phone,
                              style: TextStyle(
                                color: AppColors.onDark
                                    .withValues(alpha: 0.70),
                                fontSize: 14,
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 28),
                      ],
                    ),
                  ),
                ],
              ),
            ),

            // ── Scrollable body ──────────────────────────────────
            Expanded(
              child: ListView(
                padding:
                    const EdgeInsets.fromLTRB(20, 0, 20, 40),
                children: [
                  // Pull body up over header edge
                  Transform.translate(
                    offset: const Offset(0, -1),
                    child: Container(
                      decoration: const BoxDecoration(
                        color: AppColors.surface0,
                        borderRadius: BorderRadius.only(
                          topLeft: Radius.circular(28),
                          topRight: Radius.circular(28),
                        ),
                      ),
                      height: 28,
                    ),
                  ),
                  // Name card
                  _ProfileCard(
                    children: [
                      _CardHeader(
                        icon: Icons.badge_outlined,
                        label: 'Họ và tên',
                        trailing: !_editingName
                            ? GestureDetector(
                                onTap: () =>
                                    setState(() => _editingName = true),
                                child: const Text(
                                  'Sửa',
                                  style: TextStyle(
                                    color: AppColors.fire,
                                    fontWeight: FontWeight.w700,
                                    fontSize: 13,
                                  ),
                                ),
                              )
                            : null,
                      ),
                      const SizedBox(height: 10),
                      if (_editingName) ...[
                        TextField(
                          controller: _nameController,
                          enabled: !_saving,
                          autofocus: true,
                          textCapitalization: TextCapitalization.words,
                          textInputAction: TextInputAction.done,
                          style: const TextStyle(
                              fontWeight: FontWeight.w600, fontSize: 15),
                          decoration: InputDecoration(
                            hintText: 'Nguyễn Văn A',
                            border: OutlineInputBorder(
                              borderRadius: AppRadius.sm,
                              borderSide: const BorderSide(
                                  color: AppColors.ash),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: AppRadius.sm,
                              borderSide: const BorderSide(
                                  color: AppColors.fire, width: 1.5),
                            ),
                            contentPadding: const EdgeInsets.symmetric(
                                horizontal: 14, vertical: 12),
                          ),
                          onSubmitted: (_) {
                            if (!_saving) _saveName();
                          },
                        ),
                        const SizedBox(height: 10),
                        Row(
                          children: [
                            Expanded(
                              child: OutlinedButton(
                                onPressed: _saving
                                    ? null
                                    : () => setState(
                                        () => _editingName = false),
                                style: OutlinedButton.styleFrom(
                                  side: const BorderSide(
                                      color: AppColors.ash),
                                ),
                                child: const Text('Hủy'),
                              ),
                            ),
                            const SizedBox(width: 10),
                            Expanded(
                              child: _FireButton(
                                label: 'Lưu',
                                loading: _saving,
                                onTap: _saveName,
                              ),
                            ),
                          ],
                        ),
                      ] else
                        Text(
                          displayName ?? 'Chưa có tên',
                          style: TextStyle(
                            color: displayName != null
                                ? const Color(0xFF1C1917)
                                : AppColors.ash,
                            fontWeight: FontWeight.w600,
                            fontSize: 15,
                          ),
                        ),
                      if (_savedHint != null) ...[
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            const Icon(Icons.check_circle_outline_rounded,
                                color: AppColors.success, size: 15),
                            const SizedBox(width: 5),
                            Text(
                              _savedHint!,
                              style: const TextStyle(
                                  color: AppColors.success,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w600),
                            ),
                          ],
                        ),
                      ],
                      if (_error != null) ...[
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            const Icon(Icons.error_outline_rounded,
                                color: AppColors.danger, size: 15),
                            const SizedBox(width: 5),
                            Flexible(
                              child: Text(
                                _error!,
                                style: const TextStyle(
                                    color: AppColors.danger,
                                    fontSize: 13,
                                    fontWeight: FontWeight.w500),
                              ),
                            ),
                          ],
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 12),

                  // Actions card
                  _ProfileCard(
                    children: [
                      _ActionTile(
                        icon: Icons.receipt_long_rounded,
                        iconColor: AppColors.fire,
                        title: 'Đơn hàng của tôi',
                        subtitle: 'Lịch sử và hủy đơn đang chờ',
                        onTap: _loggingOut ? null : widget.onMyOrders,
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),

                  // Logout card
                  _ProfileCard(
                    children: [
                      _ActionTile(
                        icon: Icons.logout_rounded,
                        iconColor: AppColors.danger,
                        title: 'Đăng xuất',
                        titleColor: AppColors.danger,
                        onTap: _loggingOut ? null : _logout,
                        trailing: _loggingOut
                            ? const SizedBox(
                                width: 18,
                                height: 18,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  color: AppColors.danger,
                                ),
                              )
                            : null,
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────
class _ProfileCard extends StatelessWidget {
  const _ProfileCard({required this.children});
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 16, 18, 16),
      decoration: BoxDecoration(
        color: AppColors.surface1,
        borderRadius: AppRadius.md,
        boxShadow: AppShadow.card,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: children,
      ),
    );
  }
}

class _CardHeader extends StatelessWidget {
  const _CardHeader({required this.icon, required this.label, this.trailing});
  final IconData icon;
  final String label;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(icon, size: 16, color: AppColors.fire),
        const SizedBox(width: 6),
        Text(
          label,
          style: const TextStyle(
            fontWeight: FontWeight.w700,
            fontSize: 13,
            color: AppColors.ash,
            letterSpacing: 0.3,
          ),
        ),
        const Spacer(),
        if (trailing != null) trailing!,
      ],
    );
  }
}

class _ActionTile extends StatelessWidget {
  const _ActionTile({
    required this.icon,
    required this.iconColor,
    required this.title,
    this.subtitle,
    this.titleColor,
    this.onTap,
    this.trailing,
  });

  final IconData icon;
  final Color iconColor;
  final String title;
  final String? subtitle;
  final Color? titleColor;
  final VoidCallback? onTap;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: AppRadius.sm,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(
          children: [
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: iconColor.withValues(alpha: 0.10),
                borderRadius: AppRadius.sm,
              ),
              child: Icon(icon, color: iconColor, size: 20),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: TextStyle(
                      fontWeight: FontWeight.w700,
                      fontSize: 15,
                      color: titleColor ?? const Color(0xFF1C1917),
                    ),
                  ),
                  if (subtitle != null)
                    Text(
                      subtitle!,
                      style: const TextStyle(
                          color: AppColors.ash, fontSize: 12),
                    ),
                ],
              ),
            ),
            trailing ??
                Icon(Icons.chevron_right_rounded,
                    color: AppColors.ash.withValues(alpha: 0.5)),
          ],
        ),
      ),
    );
  }
}

class _FireButton extends StatelessWidget {
  const _FireButton({
    required this.label,
    required this.onTap,
    this.loading = false,
  });

  final String label;
  final VoidCallback onTap;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: loading ? null : onTap,
      child: Container(
        height: 44,
        decoration: BoxDecoration(
          gradient: const LinearGradient(
            colors: [AppColors.amber, AppColors.fire],
          ),
          borderRadius: AppRadius.pill,
        ),
        child: Center(
          child: loading
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: AppColors.obsidian,
                  ),
                )
              : Text(
                  label,
                  style: const TextStyle(
                    color: AppColors.obsidian,
                    fontWeight: FontWeight.w800,
                    fontSize: 14,
                  ),
                ),
        ),
      ),
    );
  }
}
