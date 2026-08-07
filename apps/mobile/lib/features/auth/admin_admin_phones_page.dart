import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'admin_phones_api.dart';
import 'auth_models.dart';

/// Admin — số điện thoại được cấp quyền admin khi đăng nhập bằng OTP.
class AdminPhonesPage extends ConsumerStatefulWidget {
  const AdminPhonesPage({super.key, required this.onBack});

  final VoidCallback onBack;

  @override
  ConsumerState<AdminPhonesPage> createState() => _AdminPhonesPageState();
}

class _AdminPhonesPageState extends ConsumerState<AdminPhonesPage> {
  final _phone = TextEditingController();
  final _label = TextEditingController();

  List<AdminPhone>? _items;
  bool _loading = true;
  bool _adding = false;
  String? _busyId;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _phone.dispose();
    _label.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final items = await ref.read(adminPhonesApiProvider).list();
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
        _error = 'Không tải được danh sách admin.';
        _loading = false;
      });
    }
  }

  Future<void> _add() async {
    final phone = _phone.text.trim();
    if (phone.isEmpty) {
      setState(() => _error = 'Nhập số điện thoại để thêm.');
      return;
    }
    setState(() {
      _adding = true;
      _error = null;
    });
    try {
      final added = await ref
          .read(adminPhonesApiProvider)
          .add(phone: phone, label: _label.text);
      if (!mounted) return;
      _phone.clear();
      _label.clear();
      await _load();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Đã thêm ${added.phoneMasked} vào danh sách admin.')),
      );
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.displayMessage);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Thêm số admin thất bại. Thử lại.');
    } finally {
      if (mounted) setState(() => _adding = false);
    }
  }

  Future<void> _remove(AdminPhone item) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Bỏ quyền admin?'),
        content: Text(
          item.isSelf
              ? 'Đây là số bạn đang đăng nhập. Bỏ khỏi danh sách sẽ khiến bạn '
                  'mất quyền admin ở lần làm mới phiên tiếp theo.'
              : 'Số ${item.phoneMasked} sẽ đăng nhập như khách hàng bình thường.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Không'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Bỏ quyền'),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;

    setState(() => _busyId = item.id);
    try {
      await ref.read(adminPhonesApiProvider).remove(item.id);
      if (!mounted) return;
      await _load();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Đã bỏ ${item.phoneMasked} khỏi danh sách admin.')),
      );
    } on AuthApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.displayMessage)),
      );
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Bỏ quyền admin thất bại.')),
      );
    } finally {
      if (mounted) setState(() => _busyId = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final busy = _adding || _busyId != null;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Số điện thoại admin'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: busy ? null : widget.onBack,
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
              'Những số dưới đây đăng nhập bằng OTP như bình thường và vào '
              'thẳng trang quản trị. Số khác vẫn là khách hàng.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            _addCard(theme),
            const SizedBox(height: 24),
            Text(
              'Danh sách admin',
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

  Widget _addCard(ThemeData theme) {
    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              controller: _phone,
              enabled: !_adding,
              keyboardType: TextInputType.phone,
              inputFormatters: [
                FilteringTextInputFormatter.allow(RegExp(r'[0-9+ .-]')),
              ],
              decoration: const InputDecoration(
                labelText: 'Số điện thoại',
                hintText: '0909777020',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _label,
              enabled: !_adding,
              textCapitalization: TextCapitalization.sentences,
              decoration: const InputDecoration(
                labelText: 'Ghi chú (tùy chọn)',
                hintText: 'Chủ cửa hàng',
                border: OutlineInputBorder(),
              ),
              onSubmitted: (_) {
                if (!_adding) _add();
              },
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(
                _error!,
                style: theme.textTheme.bodyMedium
                    ?.copyWith(color: theme.colorScheme.error),
              ),
            ],
            const SizedBox(height: 14),
            FilledButton.icon(
              onPressed: _adding ? null : _add,
              icon: _adding
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.person_add_alt_1),
              label: const Text('Thêm số admin'),
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
          child: Center(child: CircularProgressIndicator()),
        ),
      ];
    }
    final items = _items ?? const <AdminPhone>[];
    if (items.isEmpty) {
      return [
        Text(
          'Chưa có số nào. Thêm ít nhất một số để đăng nhập admin bằng OTP.',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ];
    }

    return [
      for (final item in items)
        Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Material(
            color: theme.colorScheme.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(12),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 8, 12),
              child: Row(
                children: [
                  Icon(
                    Icons.admin_panel_settings_outlined,
                    color: theme.colorScheme.primary,
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Text(
                              item.phoneMasked,
                              style: theme.textTheme.titleMedium
                                  ?.copyWith(fontWeight: FontWeight.w700),
                            ),
                            if (item.isSelf) ...[
                              const SizedBox(width: 8),
                              Text(
                                '(bạn)',
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.primary,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ],
                          ],
                        ),
                        if (item.label != null)
                          Text(
                            item.label!,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: 'Bỏ quyền admin',
                    icon: _busyId == item.id
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Icon(
                            Icons.delete_outline,
                            color: theme.colorScheme.error,
                          ),
                    onPressed:
                        _busyId != null || _adding ? null : () => _remove(item),
                  ),
                ],
              ),
            ),
          ),
        ),
    ];
  }
}
