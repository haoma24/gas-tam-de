import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'auth_models.dart';
import 'auth_session.dart';
import 'me_api.dart';

/// Customer personal profile — name, phone, orders shortcut, logout.
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

  static String _initialLetter(String name, String phone) {
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
    final theme = Theme.of(context);
    final session = ref.watch(authSessionProvider);
    final phone = _profile?.phoneMasked.isNotEmpty == true
        ? _profile!.phoneMasked
        : (session?.user.phoneMasked ?? '—');
    final initial = _initialLetter(_nameController.text, phone);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Hồ sơ cá nhân'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: _loggingOut ? null : widget.onBack,
        ),
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 32),
          children: [
            Center(
              child: CircleAvatar(
                radius: 40,
                backgroundColor: theme.colorScheme.primaryContainer,
                foregroundColor: theme.colorScheme.onPrimaryContainer,
                child: Text(
                  initial,
                  style: theme.textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 12),
            Text(
              'Tài khoản khách',
              textAlign: TextAlign.center,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              phone,
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 28),
            Text(
              'Họ và tên',
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _nameController,
              enabled: !_saving && !_loggingOut,
              textCapitalization: TextCapitalization.words,
              textInputAction: TextInputAction.done,
              decoration: const InputDecoration(
                hintText: 'VD: Nguyễn Văn A',
                border: OutlineInputBorder(),
                prefixIcon: Icon(Icons.badge_outlined),
              ),
              onSubmitted: (_) {
                if (!_saving) _saveName();
              },
            ),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: (_saving || _loggingOut) ? null : _saveName,
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(48),
              ),
              child: _saving
                  ? SizedBox(
                      height: 22,
                      width: 22,
                      child: CircularProgressIndicator(
                        strokeWidth: 2.5,
                        color: theme.colorScheme.onPrimary,
                      ),
                    )
                  : const Text('Lưu hồ sơ'),
            ),
            if (_savedHint != null) ...[
              const SizedBox(height: 8),
              Text(
                _savedHint!,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.primary,
                ),
              ),
            ],
            if (_error != null) ...[
              const SizedBox(height: 8),
              Text(
                _error!,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ],
            const SizedBox(height: 28),
            ListTile(
              contentPadding: EdgeInsets.zero,
              leading: const Icon(Icons.receipt_long_outlined),
              title: const Text('Đơn hàng của tôi'),
              subtitle: const Text('Xem lịch sử và hủy đơn đang chờ'),
              trailing: const Icon(Icons.chevron_right),
              onTap: _loggingOut ? null : widget.onMyOrders,
            ),
            const Divider(height: 32),
            OutlinedButton.icon(
              onPressed: _loggingOut ? null : _logout,
              style: OutlinedButton.styleFrom(
                minimumSize: const Size.fromHeight(48),
                foregroundColor: theme.colorScheme.error,
              ),
              icon: _loggingOut
                  ? SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: theme.colorScheme.error,
                      ),
                    )
                  : const Icon(Icons.logout),
              label: Text(_loggingOut ? 'Đang thoát…' : 'Đăng xuất'),
            ),
          ],
        ),
      ),
    );
  }
}
