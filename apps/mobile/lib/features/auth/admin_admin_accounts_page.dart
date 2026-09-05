import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/ui/ui.dart';
import 'admin_accounts_api.dart';
import 'auth_models.dart';
import 'auth_session.dart';

/// Admin username/password accounts: create managers and update credentials.
class AdminAccountsPage extends ConsumerStatefulWidget {
  const AdminAccountsPage({super.key});

  @override
  ConsumerState<AdminAccountsPage> createState() => _AdminAccountsPageState();
}

class _AdminAccountsPageState extends ConsumerState<AdminAccountsPage> {
  final _username = TextEditingController();
  final _displayName = TextEditingController();
  final _password = TextEditingController();

  List<AdminAccount>? _items;
  bool _loading = true;
  bool _creating = false;
  bool _showPassword = false;
  String? _busyId;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _username.dispose();
    _displayName.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final items = await ref.read(adminAccountsApiProvider).list();
      if (!mounted) return;
      setState(() {
        _items = items;
        _loading = false;
      });
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Không tải được danh sách tài khoản.';
        _loading = false;
      });
    }
  }

  Future<void> _create() async {
    final username = _username.text.trim();
    final password = _password.text;
    if (username.length < 3) {
      setState(() => _error = 'Tên đăng nhập phải có ít nhất 3 ký tự.');
      return;
    }
    if (password.length < 8) {
      setState(() => _error = 'Mật khẩu phải có ít nhất 8 ký tự.');
      return;
    }
    setState(() {
      _creating = true;
      _error = null;
    });
    try {
      final created = await ref.read(adminAccountsApiProvider).create(
            username: username,
            password: password,
            displayName: _displayName.text,
          );
      if (!mounted) return;
      _username.clear();
      _displayName.clear();
      _password.clear();
      await _load();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Đã tạo tài khoản ${created.username}.')),
      );
    } on AuthApiException catch (e) {
      if (mounted) setState(() => _error = e.displayMessage);
    } catch (_) {
      if (mounted) setState(() => _error = 'Tạo tài khoản thất bại. Thử lại.');
    } finally {
      if (mounted) setState(() => _creating = false);
    }
  }

  Future<void> _edit(AdminAccount item) async {
    final value = await showDialog<_AccountEditValue>(
      context: context,
      barrierDismissible: false,
      builder: (_) => _EditAccountDialog(account: item),
    );
    if (value == null || !mounted) return;
    setState(() => _busyId = item.id);
    try {
      final updated = await ref.read(adminAccountsApiProvider).update(
            id: item.id,
            username: value.username,
            displayName: value.displayName,
            newPassword: value.newPassword,
            currentPassword: value.currentPassword,
          );
      if (updated.isSelf) {
        await ref.read(authSessionProvider.notifier).updateAdminIdentity(
              username: updated.username,
              displayName: updated.displayName,
            );
      }
      if (!mounted) return;
      await _load();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Đã cập nhật ${updated.username}.')),
      );
    } on AuthApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.displayMessage)));
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Cập nhật tài khoản thất bại.')),
      );
    } finally {
      if (mounted) setState(() => _busyId = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final busy = _creating || _busyId != null;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Tài khoản quản lý'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: busy ? null : () => popOrGo(context, '/admin/settings'),
        ),
        actions: [
          IconButton(
            tooltip: 'Tải lại',
            icon: const Icon(Icons.refresh),
            onPressed: _loading || busy ? null : _load,
          ),
        ],
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 32),
          children: [
            Text(
              'Tạo tài khoản đăng nhập riêng cho nhân viên quản lý. Mỗi tài '
              'khoản đều có đầy đủ quyền admin.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            _createCard(theme),
            const SizedBox(height: 24),
            Text(
              'Tài khoản đang hoạt động',
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 12),
            ..._listBody(theme),
          ],
        ),
      ),
    );
  }

  Widget _createCard(ThemeData theme) {
    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      shape: RoundedRectangleBorder(
        borderRadius: AppRadius.md,
        side: BorderSide(color: theme.colorScheme.outline),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Tạo tài khoản mới',
                style: theme.textTheme.titleSmall
                    ?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 14),
            TextField(
              controller: _username,
              enabled: !_creating,
              autocorrect: false,
              decoration: const InputDecoration(
                labelText: 'Tên đăng nhập',
                hintText: 'quanly-ca-sang',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _displayName,
              enabled: !_creating,
              textCapitalization: TextCapitalization.words,
              decoration: const InputDecoration(
                labelText: 'Tên hiển thị (tùy chọn)',
                hintText: 'Quản lý ca sáng',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _password,
              enabled: !_creating,
              obscureText: !_showPassword,
              autocorrect: false,
              enableSuggestions: false,
              decoration: InputDecoration(
                labelText: 'Mật khẩu ban đầu',
                helperText: 'Tối thiểu 8 ký tự',
                border: const OutlineInputBorder(),
                suffixIcon: IconButton(
                  tooltip: _showPassword ? 'Ẩn mật khẩu' : 'Hiện mật khẩu',
                  icon: Icon(_showPassword
                      ? Icons.visibility_off_outlined
                      : Icons.visibility_outlined),
                  onPressed: _creating
                      ? null
                      : () => setState(() => _showPassword = !_showPassword),
                ),
              ),
              onSubmitted: (_) {
                if (!_creating) _create();
              },
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!,
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(color: theme.colorScheme.error)),
            ],
            const SizedBox(height: 14),
            FilledButton.icon(
              onPressed: _creating ? null : _create,
              icon: _creating
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.person_add_alt_1),
              label: const Text('Tạo tài khoản quản lý'),
            ),
          ],
        ),
      ),
    );
  }

  List<Widget> _listBody(ThemeData theme) {
    if (_loading && _items == null) {
      return const [
        Padding(
          padding: EdgeInsets.symmetric(vertical: 32),
          child: AppLoading(),
        ),
      ];
    }
    final items = _items ?? const <AdminAccount>[];
    if (items.isEmpty) {
      return [const Text('Chưa có tài khoản quản lý bằng mật khẩu.')];
    }
    return [
      for (final item in items)
        Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            tileColor: theme.colorScheme.surfaceContainerLowest,
            shape: RoundedRectangleBorder(
              borderRadius: AppRadius.md,
              side: BorderSide(color: theme.colorScheme.outline),
            ),
            leading: Icon(Icons.manage_accounts_outlined,
                color: theme.colorScheme.primary),
            title: Row(
              children: [
                Flexible(
                  child: Text(item.displayName ?? item.username,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontWeight: FontWeight.w700)),
                ),
                if (item.isSelf) ...[
                  const SizedBox(width: 8),
                  Text('(bạn)',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.primary,
                        fontWeight: FontWeight.w600,
                      )),
                ],
              ],
            ),
            subtitle: item.displayName == null ? null : Text(item.username),
            trailing: _busyId == item.id
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : IconButton(
                    tooltip: 'Sửa tài khoản / mật khẩu',
                    icon: const Icon(Icons.edit_outlined),
                    onPressed:
                        _busyId != null || _creating ? null : () => _edit(item),
                  ),
          ),
        ),
    ];
  }
}

class _AccountEditValue {
  const _AccountEditValue({
    required this.username,
    required this.displayName,
    this.newPassword,
    this.currentPassword,
  });

  final String username;
  final String displayName;
  final String? newPassword;
  final String? currentPassword;
}

class _EditAccountDialog extends StatefulWidget {
  const _EditAccountDialog({required this.account});

  final AdminAccount account;

  @override
  State<_EditAccountDialog> createState() => _EditAccountDialogState();
}

class _EditAccountDialogState extends State<_EditAccountDialog> {
  late final TextEditingController _username;
  late final TextEditingController _displayName;
  final _newPassword = TextEditingController();
  final _currentPassword = TextEditingController();
  bool _showPasswords = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _username = TextEditingController(text: widget.account.username);
    _displayName = TextEditingController(text: widget.account.displayName);
  }

  @override
  void dispose() {
    _username.dispose();
    _displayName.dispose();
    _newPassword.dispose();
    _currentPassword.dispose();
    super.dispose();
  }

  void _submit() {
    final username = _username.text.trim();
    final newPassword = _newPassword.text;
    final credentialsChanged =
        username != widget.account.username || newPassword.isNotEmpty;
    if (username.length < 3) {
      setState(() => _error = 'Tên đăng nhập phải có ít nhất 3 ký tự.');
      return;
    }
    if (newPassword.isNotEmpty && newPassword.length < 8) {
      setState(() => _error = 'Mật khẩu mới phải có ít nhất 8 ký tự.');
      return;
    }
    if (widget.account.isSelf &&
        credentialsChanged &&
        _currentPassword.text.isEmpty) {
      setState(() => _error = 'Nhập mật khẩu hiện tại để xác nhận.');
      return;
    }
    Navigator.pop(
      context,
      _AccountEditValue(
        username: username,
        displayName: _displayName.text.trim(),
        newPassword: newPassword.isEmpty ? null : newPassword,
        currentPassword:
            _currentPassword.text.isEmpty ? null : _currentPassword.text,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Sửa tài khoản quản lý'),
      content: SingleChildScrollView(
        child: SizedBox(
          width: 420,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _username,
                autocorrect: false,
                decoration: const InputDecoration(
                  labelText: 'Tên đăng nhập',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _displayName,
                decoration: const InputDecoration(
                  labelText: 'Tên hiển thị (tùy chọn)',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _newPassword,
                obscureText: !_showPasswords,
                autocorrect: false,
                enableSuggestions: false,
                decoration: InputDecoration(
                  labelText: 'Mật khẩu mới (để trống nếu không đổi)',
                  helperText: 'Tối thiểu 8 ký tự',
                  border: const OutlineInputBorder(),
                  suffixIcon: IconButton(
                    tooltip: _showPasswords ? 'Ẩn mật khẩu' : 'Hiện mật khẩu',
                    icon: Icon(_showPasswords
                        ? Icons.visibility_off_outlined
                        : Icons.visibility_outlined),
                    onPressed: () =>
                        setState(() => _showPasswords = !_showPasswords),
                  ),
                ),
              ),
              if (widget.account.isSelf) ...[
                const SizedBox(height: 12),
                TextField(
                  controller: _currentPassword,
                  obscureText: !_showPasswords,
                  autocorrect: false,
                  enableSuggestions: false,
                  decoration: const InputDecoration(
                    labelText: 'Mật khẩu hiện tại',
                    helperText: 'Bắt buộc khi đổi tên đăng nhập hoặc mật khẩu',
                    border: OutlineInputBorder(),
                  ),
                ),
              ],
              if (_error != null) ...[
                const SizedBox(height: 12),
                Align(
                  alignment: Alignment.centerLeft,
                  child: Text(_error!,
                      style: TextStyle(
                          color: Theme.of(context).colorScheme.error)),
                ),
              ],
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Hủy'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Lưu thay đổi')),
      ],
    );
  }
}
