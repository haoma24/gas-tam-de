import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../catalog/catalog_models.dart';
import 'inventory_api.dart';
import 'inventory_models.dart';

/// Admin tồn kho — list + IN / OUT / ADJUST (`GET/POST /v1/admin/inventory`).
class AdminInventoryPage extends ConsumerStatefulWidget {
  const AdminInventoryPage({
    super.key,
    required this.onBack,
  });

  final VoidCallback onBack;

  @override
  ConsumerState<AdminInventoryPage> createState() => _AdminInventoryPageState();
}

class _AdminInventoryPageState extends ConsumerState<AdminInventoryPage> {
  StockList? _data;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final data = await ref.read(inventoryApiProvider).listStock();
      if (!mounted) return;
      setState(() {
        _data = data;
        _loading = false;
      });
    } on InventoryApiException catch (e) {
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

  Future<void> _openMovement({
    required StockMovementType type,
    StockItem? item,
    bool createNew = false,
  }) async {
    final result = await showDialog<_MovementSubmit>(
      context: context,
      builder: (ctx) => _MovementDialog(
        type: type,
        item: item,
        createNew: createNew,
      ),
    );
    if (result == null || !mounted) return;

    try {
      final out = await ref.read(inventoryApiProvider).postMovement(
            movementType: result.type,
            productId: result.productId,
            qty: result.qty,
            delta: result.delta,
            unitCost: result.unitCost,
            sku: result.sku,
            name: result.name,
            reorderLevel: result.reorderLevel,
            note: result.note,
          );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            '${result.type.labelVi}: «${out.item.name}» → tồn ${out.item.onHand}',
          ),
        ),
      );
      await _load();
    } on InventoryApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(e.displayMessage)),
      );
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Có lỗi xảy ra. Thử lại.')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Tồn kho'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: widget.onBack,
        ),
        actions: [
          IconButton(
            tooltip: 'Tải lại',
            icon: const Icon(Icons.refresh),
            onPressed: _loading ? null : _load,
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _openMovement(
          type: StockMovementType.inn,
          createNew: true,
        ),
        icon: const Icon(Icons.add),
        label: const Text('Nhập mới'),
      ),
      body: SafeArea(child: _buildBody(theme)),
    );
  }

  Widget _buildBody(ThemeData theme) {
    if (_loading && _data == null) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null && _data == null) {
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

    final items = _data?.items ?? const <StockItem>[];
    if (items.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                'Chưa có tồn kho',
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                'Nhập kho lần đầu để tạo dòng tồn theo mã sản phẩm.',
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 20),
              FilledButton.icon(
                onPressed: () => _openMovement(
                  type: StockMovementType.inn,
                  createNew: true,
                ),
                icon: const Icon(Icons.add),
                label: const Text('Nhập kho mới'),
              ),
            ],
          ),
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 88),
        itemCount: items.length,
        separatorBuilder: (_, __) => const SizedBox(height: 8),
        itemBuilder: (context, index) {
          final item = items[index];
          return _StockTile(
            item: item,
            onIn: () => _openMovement(
              type: StockMovementType.inn,
              item: item,
            ),
            onOut: () => _openMovement(
              type: StockMovementType.out,
              item: item,
            ),
            onAdjust: () => _openMovement(
              type: StockMovementType.adjust,
              item: item,
            ),
          );
        },
      ),
    );
  }
}

class _StockTile extends StatelessWidget {
  const _StockTile({
    required this.item,
    required this.onIn,
    required this.onOut,
    required this.onAdjust,
  });

  final StockItem item;
  final VoidCallback onIn;
  final VoidCallback onOut;
  final VoidCallback onAdjust;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final muted = theme.colorScheme.onSurfaceVariant;
    final low = item.isLowStock;

    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 8, 14),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          item.name,
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      if (low) ...[
                        const SizedBox(width: 8),
                        _LowStockChip(),
                      ],
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    item.sku,
                    style: theme.textTheme.bodyMedium?.copyWith(color: muted),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Tồn: ${item.onHand}',
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w700,
                      color: low
                          ? theme.colorScheme.error
                          : theme.colorScheme.primary,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    'Giá vốn: ${formatVnd(item.costPrice)}'
                    '${item.reorderLevel > 0 ? ' · ngưỡng ${item.reorderLevel}' : ''}',
                    style: theme.textTheme.bodySmall?.copyWith(color: muted),
                  ),
                ],
              ),
            ),
            PopupMenuButton<String>(
              tooltip: 'Phiếu kho',
              onSelected: (v) {
                switch (v) {
                  case 'in':
                    onIn();
                  case 'out':
                    onOut();
                  case 'adjust':
                    onAdjust();
                }
              },
              itemBuilder: (ctx) => const [
                PopupMenuItem(value: 'in', child: Text('Nhập kho')),
                PopupMenuItem(value: 'out', child: Text('Xuất kho')),
                PopupMenuItem(value: 'adjust', child: Text('Điều chỉnh')),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _LowStockChip extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: theme.colorScheme.errorContainer,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        'Sắp hết',
        style: theme.textTheme.labelSmall?.copyWith(
          color: theme.colorScheme.onErrorContainer,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _MovementSubmit {
  const _MovementSubmit({
    required this.type,
    required this.productId,
    this.qty,
    this.delta,
    this.unitCost,
    this.sku,
    this.name,
    this.reorderLevel,
    this.note,
  });

  final StockMovementType type;
  final String productId;
  final int? qty;
  final int? delta;
  final int? unitCost;
  final String? sku;
  final String? name;
  final int? reorderLevel;
  final String? note;
}

class _MovementDialog extends StatefulWidget {
  const _MovementDialog({
    required this.type,
    this.item,
    this.createNew = false,
  });

  final StockMovementType type;
  final StockItem? item;
  final bool createNew;

  @override
  State<_MovementDialog> createState() => _MovementDialogState();
}

class _MovementDialogState extends State<_MovementDialog> {
  late final TextEditingController _productIdCtrl;
  late final TextEditingController _skuCtrl;
  late final TextEditingController _nameCtrl;
  late final TextEditingController _qtyCtrl;
  late final TextEditingController _deltaCtrl;
  late final TextEditingController _unitCostCtrl;
  late final TextEditingController _reorderCtrl;
  late final TextEditingController _noteCtrl;
  String? _localError;

  bool get _needsCreateFields =>
      widget.createNew || widget.item == null;

  @override
  void initState() {
    super.initState();
    final item = widget.item;
    _productIdCtrl = TextEditingController(text: item?.productId ?? '');
    _skuCtrl = TextEditingController(text: item?.sku ?? '');
    _nameCtrl = TextEditingController(text: item?.name ?? '');
    _qtyCtrl = TextEditingController();
    _deltaCtrl = TextEditingController();
    _unitCostCtrl = TextEditingController(
      text: item != null && widget.type == StockMovementType.inn
          ? '${item.costPrice}'
          : '',
    );
    _reorderCtrl = TextEditingController(
      text: item != null ? '${item.reorderLevel}' : '0',
    );
    _noteCtrl = TextEditingController();
  }

  @override
  void dispose() {
    _productIdCtrl.dispose();
    _skuCtrl.dispose();
    _nameCtrl.dispose();
    _qtyCtrl.dispose();
    _deltaCtrl.dispose();
    _unitCostCtrl.dispose();
    _reorderCtrl.dispose();
    _noteCtrl.dispose();
    super.dispose();
  }

  void _submit() {
    setState(() => _localError = null);
    final productId = _productIdCtrl.text.trim();
    if (productId.isEmpty) {
      setState(() => _localError = 'Nhập mã sản phẩm (product_id).');
      return;
    }

    final note = _noteCtrl.text.trim();
    final noteOrNull = note.isEmpty ? null : note;

    switch (widget.type) {
      case StockMovementType.inn:
        final qty = int.tryParse(_qtyCtrl.text.trim());
        if (qty == null || qty <= 0) {
          setState(() => _localError = 'Số lượng nhập phải > 0.');
          return;
        }
        final unitCost = int.tryParse(
          _unitCostCtrl.text.trim().replaceAll(RegExp(r'[^\d]'), ''),
        );
        if (unitCost == null || unitCost < 0) {
          setState(() => _localError = 'Giá nhập (VND) không hợp lệ.');
          return;
        }
        String? sku;
        String? name;
        int? reorder;
        if (_needsCreateFields) {
          sku = _skuCtrl.text.trim();
          name = _nameCtrl.text.trim();
          if (sku.isEmpty || name.isEmpty) {
            setState(() => _localError = 'SKU và tên bắt buộc khi tạo tồn mới.');
            return;
          }
          reorder = int.tryParse(_reorderCtrl.text.trim()) ?? 0;
          if (reorder < 0) {
            setState(() => _localError = 'Ngưỡng tồn phải ≥ 0.');
            return;
          }
        }
        Navigator.of(context).pop(
          _MovementSubmit(
            type: StockMovementType.inn,
            productId: productId,
            qty: qty,
            unitCost: unitCost,
            sku: sku,
            name: name,
            reorderLevel: reorder,
            note: noteOrNull,
          ),
        );

      case StockMovementType.out:
        final qty = int.tryParse(_qtyCtrl.text.trim());
        if (qty == null || qty <= 0) {
          setState(() => _localError = 'Số lượng xuất phải > 0.');
          return;
        }
        Navigator.of(context).pop(
          _MovementSubmit(
            type: StockMovementType.out,
            productId: productId,
            qty: qty,
            note: noteOrNull,
          ),
        );

      case StockMovementType.adjust:
        final delta = int.tryParse(_deltaCtrl.text.trim());
        if (delta == null || delta == 0) {
          setState(() => _localError = 'Delta phải khác 0 (âm hoặc dương).');
          return;
        }
        int? unitCost;
        final costRaw = _unitCostCtrl.text.trim();
        if (costRaw.isNotEmpty) {
          unitCost = int.tryParse(costRaw.replaceAll(RegExp(r'[^\d]'), ''));
          if (unitCost == null || unitCost < 0) {
            setState(() => _localError = 'Giá vốn (tuỳ chọn) không hợp lệ.');
            return;
          }
        }
        Navigator.of(context).pop(
          _MovementSubmit(
            type: StockMovementType.adjust,
            productId: productId,
            delta: delta,
            unitCost: unitCost,
            note: noteOrNull,
          ),
        );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final type = widget.type;
    final existing = widget.item != null && !widget.createNew;

    return AlertDialog(
      title: Text(type.labelVi),
      content: SingleChildScrollView(
        child: SizedBox(
          width: 360,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (existing)
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: Text(
                    '${widget.item!.name} · tồn ${widget.item!.onHand}',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
              if (!existing) ...[
                TextField(
                  controller: _productIdCtrl,
                  decoration: const InputDecoration(
                    labelText: 'Mã sản phẩm (product_id)',
                    border: OutlineInputBorder(),
                  ),
                  textInputAction: TextInputAction.next,
                ),
                const SizedBox(height: 12),
                if (type == StockMovementType.inn) ...[
                  TextField(
                    controller: _skuCtrl,
                    decoration: const InputDecoration(
                      labelText: 'SKU',
                      border: OutlineInputBorder(),
                    ),
                    textInputAction: TextInputAction.next,
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _nameCtrl,
                    decoration: const InputDecoration(
                      labelText: 'Tên sản phẩm',
                      border: OutlineInputBorder(),
                    ),
                    textInputAction: TextInputAction.next,
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _reorderCtrl,
                    decoration: const InputDecoration(
                      labelText: 'Ngưỡng cảnh báo (tuỳ chọn)',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.number,
                    inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                    textInputAction: TextInputAction.next,
                  ),
                  const SizedBox(height: 12),
                ],
              ] else
                const SizedBox.shrink(),
              if (type == StockMovementType.adjust)
                TextField(
                  controller: _deltaCtrl,
                  decoration: const InputDecoration(
                    labelText: 'Delta (+ nhập / − xuất)',
                    hintText: 'vd. 2 hoặc -1',
                    border: OutlineInputBorder(),
                  ),
                  keyboardType: const TextInputType.numberWithOptions(
                    signed: true,
                  ),
                  inputFormatters: [
                    FilteringTextInputFormatter.allow(RegExp(r'^-?\d*')),
                  ],
                  textInputAction: TextInputAction.next,
                )
              else
                TextField(
                  controller: _qtyCtrl,
                  decoration: InputDecoration(
                    labelText: type == StockMovementType.inn
                        ? 'Số lượng nhập'
                        : 'Số lượng xuất',
                    border: const OutlineInputBorder(),
                  ),
                  keyboardType: TextInputType.number,
                  inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                  textInputAction: TextInputAction.next,
                  autofocus: existing,
                ),
              if (type == StockMovementType.inn ||
                  type == StockMovementType.adjust) ...[
                const SizedBox(height: 12),
                TextField(
                  controller: _unitCostCtrl,
                  decoration: InputDecoration(
                    labelText: type == StockMovementType.inn
                        ? 'Giá nhập (VND)'
                        : 'Giá vốn mới (tuỳ chọn)',
                    border: const OutlineInputBorder(),
                  ),
                  keyboardType: TextInputType.number,
                  inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                  textInputAction: TextInputAction.next,
                ),
              ],
              const SizedBox(height: 12),
              TextField(
                controller: _noteCtrl,
                decoration: const InputDecoration(
                  labelText: 'Ghi chú (tuỳ chọn)',
                  border: OutlineInputBorder(),
                ),
                maxLines: 2,
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => _submit(),
              ),
              if (_localError != null) ...[
                const SizedBox(height: 12),
                Text(
                  _localError!,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.error,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Hủy'),
        ),
        FilledButton(
          onPressed: _submit,
          child: const Text('Xác nhận'),
        ),
      ],
    );
  }
}
