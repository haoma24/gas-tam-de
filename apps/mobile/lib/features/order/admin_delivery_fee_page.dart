import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../catalog/catalog_models.dart';
import 'delivery_fee_api.dart';
import 'delivery_fee_models.dart';

/// Admin delivery fee — toggle + distance bands (`GET/PUT /v1/admin/delivery-fee`).
class AdminDeliveryFeePage extends ConsumerStatefulWidget {
  const AdminDeliveryFeePage({
    super.key,
    required this.onBack,
  });

  final VoidCallback onBack;

  @override
  ConsumerState<AdminDeliveryFeePage> createState() =>
      _AdminDeliveryFeePageState();
}

class _AdminDeliveryFeePageState extends ConsumerState<AdminDeliveryFeePage> {
  bool _loading = true;
  bool _saving = false;
  String? _error;
  String? _updatedAt;

  bool _enabled = false;
  final List<_RuleDraft> _drafts = [];

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final d in _drafts) {
      d.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final cfg = await ref.read(deliveryFeeApiProvider).getConfig();
      if (!mounted) return;
      _applyConfig(cfg);
      setState(() => _loading = false);
    } on DeliveryFeeApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Có lỗi xảy ra. Thử lại.';
        _loading = false;
      });
    }
  }

  void _applyConfig(DeliveryFeeConfig cfg) {
    for (final d in _drafts) {
      d.dispose();
    }
    _drafts.clear();
    _enabled = cfg.enabled;
    _updatedAt = cfg.updatedAt;
    for (final rule in cfg.rules) {
      _drafts.add(_RuleDraft.fromRule(rule));
    }
  }

  void _addBand() {
    final lastMax = _drafts.isEmpty
        ? 0.0
        : (double.tryParse(_drafts.last.maxController.text.trim()) ??
            (double.tryParse(_drafts.last.minController.text.trim()) ?? 0) + 5);
    setState(() {
      _drafts.add(
        _RuleDraft(
          id: '',
          minController: TextEditingController(text: _fmtNum(lastMax)),
          maxController: TextEditingController(),
          feeController: TextEditingController(text: '0'),
          active: true,
        ),
      );
    });
  }

  void _removeBand(int index) {
    setState(() {
      _drafts[index].dispose();
      _drafts.removeAt(index);
    });
  }

  String? _validateDrafts() {
    final parsed = <_ParsedRule>[];
    for (var i = 0; i < _drafts.length; i++) {
      final d = _drafts[i];
      final minKm = double.tryParse(d.minController.text.trim().replaceAll(',', '.'));
      if (minKm == null || minKm < 0) {
        return 'Bậc ${i + 1}: min km không hợp lệ.';
      }
      final maxRaw = d.maxController.text.trim().replaceAll(',', '.');
      double? maxKm;
      if (maxRaw.isNotEmpty) {
        maxKm = double.tryParse(maxRaw);
        if (maxKm == null || maxKm <= minKm) {
          return 'Bậc ${i + 1}: max km phải > min km (hoặc để trống = không giới hạn).';
        }
      }
      final fee = int.tryParse(
        d.feeController.text.trim().replaceAll(RegExp(r'[^\d]'), ''),
      );
      if (fee == null || fee < 0) {
        return 'Bậc ${i + 1}: phí VND không hợp lệ.';
      }
      parsed.add(_ParsedRule(
        id: d.id,
        minKm: minKm,
        maxKm: maxKm,
        feeVnd: fee,
        active: d.active,
        sortOrder: i,
      ));
    }

    final active = parsed.where((r) => r.active).toList()
      ..sort((a, b) => a.minKm.compareTo(b.minKm));
    for (var i = 0; i < active.length; i++) {
      final r = active[i];
      if (r.maxKm == null && i != active.length - 1) {
        return 'Bậc không giới hạn (max trống) phải là bậc active cuối theo min km.';
      }
      if (i > 0) {
        final prev = active[i - 1];
        if (prev.maxKm == null || prev.maxKm! > r.minKm) {
          return 'Các bậc active bị chồng khoảng cách — chỉnh min/max.';
        }
      }
    }
    return null;
  }

  List<DeliveryFeeRule> _buildRules() {
    return [
      for (var i = 0; i < _drafts.length; i++)
        DeliveryFeeRule(
          id: _drafts[i].id,
          minKm: double.parse(
            _drafts[i].minController.text.trim().replaceAll(',', '.'),
          ),
          maxKm: () {
            final raw =
                _drafts[i].maxController.text.trim().replaceAll(',', '.');
            if (raw.isEmpty) return null;
            return double.parse(raw);
          }(),
          feeVnd: int.parse(
            _drafts[i]
                .feeController
                .text
                .trim()
                .replaceAll(RegExp(r'[^\d]'), ''),
          ),
          sortOrder: i,
          active: _drafts[i].active,
        ),
    ];
  }

  Future<void> _save() async {
    setState(() => _error = null);
    final localErr = _validateDrafts();
    if (localErr != null) {
      setState(() => _error = localErr);
      return;
    }

    setState(() => _saving = true);
    try {
      final cfg = await ref.read(deliveryFeeApiProvider).putConfig(
            enabled: _enabled,
            rules: _buildRules(),
          );
      if (!mounted) return;
      _applyConfig(cfg);
      setState(() => _saving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Đã lưu cấu hình phí giao.')),
      );
    } on DeliveryFeeApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _saving = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Có lỗi xảy ra. Thử lại.';
        _saving = false;
      });
    }
  }

  Future<void> _toggleEnabled(bool next) async {
    final prev = _enabled;
    setState(() {
      _enabled = next;
      _error = null;
    });
    try {
      final cfg =
          await ref.read(deliveryFeeApiProvider).putConfig(enabled: next);
      if (!mounted) return;
      setState(() {
        _updatedAt = cfg.updatedAt;
      });
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            next ? 'Đã bật phí giao hàng.' : 'Đã tắt phí giao hàng.',
          ),
        ),
      );
    } on DeliveryFeeApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _enabled = prev;
        _error = e.displayMessage;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _enabled = prev;
        _error = 'Có lỗi xảy ra. Thử lại.';
      });
    }
  }

  static String _fmtNum(double v) {
    if (v == v.roundToDouble()) return v.toInt().toString();
    return v.toString();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Phí giao hàng'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: widget.onBack,
        ),
        actions: [
          IconButton(
            tooltip: 'Tải lại',
            icon: const Icon(Icons.refresh),
            onPressed: (_loading || _saving) ? null : _load,
          ),
        ],
      ),
      floatingActionButton: (!_loading && _error == null) || _drafts.isNotEmpty
          ? FloatingActionButton.extended(
              onPressed: _saving || _loading ? null : _save,
              icon: _saving
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.save_outlined),
              label: Text(_saving ? 'Đang lưu…' : 'Lưu bậc'),
            )
          : null,
      body: SafeArea(child: _buildBody(theme)),
    );
  }

  Widget _buildBody(ThemeData theme) {
    if (_loading && _drafts.isEmpty && _error == null) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null && _drafts.isEmpty && !_loading) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                _error!,
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: _load,
                child: const Text('Thử lại'),
              ),
            ],
          ),
        ),
      );
    }

    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 100),
      children: [
        Material(
          color: theme.colorScheme.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(12),
          child: SwitchListTile(
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
            title: Text(
              'Thu phí giao hàng',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            subtitle: Text(
              _enabled
                  ? 'Khách sẽ bị tính phí theo bậc khoảng cách khi đặt đơn.'
                  : 'Phí giao = 0 ₫ dù đã cấu hình bậc.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            value: _enabled,
            onChanged: _saving ? null : _toggleEnabled,
          ),
        ),
        if (_updatedAt != null && _updatedAt!.isNotEmpty) ...[
          const SizedBox(height: 8),
          Text(
            'Cập nhật: $_updatedAt',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
        const SizedBox(height: 20),
        Row(
          children: [
            Expanded(
              child: Text(
                'Bậc khoảng cách',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            TextButton.icon(
              onPressed: _saving ? null : _addBand,
              icon: const Icon(Icons.add),
              label: const Text('Thêm bậc'),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          'Mỗi bậc là nửa khoảng [min, max) km. Max trống = không giới hạn.',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        if (_error != null) ...[
          const SizedBox(height: 12),
          Text(
            _error!,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.error,
            ),
          ),
        ],
        const SizedBox(height: 12),
        if (_drafts.isEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 24),
            child: Text(
              'Chưa có bậc. Thêm bậc hoặc tải lại từ máy chủ.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          )
        else
          for (var i = 0; i < _drafts.length; i++) ...[
            if (i > 0) const SizedBox(height: 10),
            _RuleCard(
              index: i,
              draft: _drafts[i],
              enabled: !_saving,
              onActiveChanged: (v) => setState(() => _drafts[i].active = v),
              onRemove: _drafts.length > 1 ? () => _removeBand(i) : null,
            ),
          ],
      ],
    );
  }
}

class _ParsedRule {
  const _ParsedRule({
    required this.id,
    required this.minKm,
    required this.maxKm,
    required this.feeVnd,
    required this.active,
    required this.sortOrder,
  });

  final String id;
  final double minKm;
  final double? maxKm;
  final int feeVnd;
  final bool active;
  final int sortOrder;
}

class _RuleDraft {
  _RuleDraft({
    required this.id,
    required this.minController,
    required this.maxController,
    required this.feeController,
    required this.active,
  });

  factory _RuleDraft.fromRule(DeliveryFeeRule rule) {
    String fmt(double v) {
      if (v == v.roundToDouble()) return v.toInt().toString();
      return v.toString();
    }

    return _RuleDraft(
      id: rule.id,
      minController: TextEditingController(text: fmt(rule.minKm)),
      maxController: TextEditingController(
        text: rule.maxKm == null ? '' : fmt(rule.maxKm!),
      ),
      feeController: TextEditingController(text: rule.feeVnd.toString()),
      active: rule.active,
    );
  }

  final String id;
  final TextEditingController minController;
  final TextEditingController maxController;
  final TextEditingController feeController;
  bool active;

  void dispose() {
    minController.dispose();
    maxController.dispose();
    feeController.dispose();
  }
}

class _RuleCard extends StatelessWidget {
  const _RuleCard({
    required this.index,
    required this.draft,
    required this.enabled,
    required this.onActiveChanged,
    this.onRemove,
  });

  final int index;
  final _RuleDraft draft;
  final bool enabled;
  final ValueChanged<bool> onActiveChanged;
  final VoidCallback? onRemove;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final feeHint = int.tryParse(
      draft.feeController.text.trim().replaceAll(RegExp(r'[^\d]'), ''),
    );

    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 8, 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    'Bậc ${index + 1}',
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                if (feeHint != null)
                  Text(
                    formatVnd(feeHint),
                    style: theme.textTheme.labelLarge?.copyWith(
                      color: theme.colorScheme.primary,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                if (onRemove != null)
                  IconButton(
                    tooltip: 'Xóa bậc',
                    onPressed: enabled ? onRemove : null,
                    icon: const Icon(Icons.delete_outline),
                  ),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: draft.minController,
                    enabled: enabled,
                    keyboardType:
                        const TextInputType.numberWithOptions(decimal: true),
                    inputFormatters: [
                      FilteringTextInputFormatter.allow(RegExp(r'[0-9.,]')),
                    ],
                    decoration: const InputDecoration(
                      labelText: 'Min km',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: TextField(
                    controller: draft.maxController,
                    enabled: enabled,
                    keyboardType:
                        const TextInputType.numberWithOptions(decimal: true),
                    inputFormatters: [
                      FilteringTextInputFormatter.allow(RegExp(r'[0-9.,]')),
                    ],
                    decoration: const InputDecoration(
                      labelText: 'Max km',
                      hintText: '∞',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            TextField(
              controller: draft.feeController,
              enabled: enabled,
              keyboardType: TextInputType.number,
              inputFormatters: [FilteringTextInputFormatter.digitsOnly],
              decoration: const InputDecoration(
                labelText: 'Phí (VND)',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 4),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              dense: true,
              title: const Text('Đang áp dụng'),
              value: draft.active,
              onChanged: enabled ? onActiveChanged : null,
            ),
          ],
        ),
      ),
    );
  }
}
