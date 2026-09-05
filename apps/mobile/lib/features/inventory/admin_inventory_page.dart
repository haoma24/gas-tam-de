import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/ui/ui.dart';

import '../catalog/catalog_api.dart';
import '../catalog/catalog_models.dart';
import 'inventory_api.dart';
import 'inventory_models.dart';

/// Admin tồn kho — list + IN / OUT / ADJUST (`GET/POST /v1/admin/inventory`).
class AdminInventoryPage extends ConsumerStatefulWidget {
  const AdminInventoryPage({
    super.key,
  });

  @override
  ConsumerState<AdminInventoryPage> createState() => _AdminInventoryPageState();
}

class _AdminInventoryPageState extends ConsumerState<AdminInventoryPage> {
  StockList? _data;
  List<Product> _products = const [];
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
    // Catalog is loaded alongside stock so "Nhập mới" can offer real products.
    // It is deliberately not fatal: if catalog is unreachable the stock list
    // still renders and the dialog falls back to typing a product_id.
    final catalog = ref
        .read(catalogApiProvider)
        .listAdminProducts()
        .then<List<Product>>((p) => p)
        .catchError((_) => const <Product>[]);
    try {
      final data = await ref.read(inventoryApiProvider).listStock();
      final products = await catalog;
      if (!mounted) return;
      setState(() {
        _data = data;
        _products = products;
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
        products: _products,
        stockedProductIds: _stockedProductIds,
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

  Set<String> get _stockedProductIds => {
        for (final s in _data?.items ?? const <StockItem>[]) s.productId,
      };

  /// inventory-service creates a stock row from `catalog.product.updated`, so
  /// normally every product already has one and stocking happens on the row
  /// itself. "Nhập mới" is only offered when it can actually do something:
  /// a product still missing a row, or catalog being unreachable (manual path).
  bool get _canAddNewStock {
    if (_products.isEmpty) return true;
    final stocked = _stockedProductIds;
    return _products.any((p) => !stocked.contains(p.id));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Tồn kho'),
        actions: [
          IconButton(
            tooltip: 'Tải lại',
            icon: const Icon(Icons.refresh),
            onPressed: _loading ? null : _load,
          ),
        ],
      ),
      floatingActionButton: _canAddNewStock
          ? FloatingActionButton.extended(
              onPressed: () => _openMovement(
                type: StockMovementType.inn,
                createNew: true,
              ),
              icon: const Icon(Icons.add),
              label: const Text('Nhập mới'),
            )
          : null,
      body: SafeArea(child: _buildBody(theme)),
    );
  }

  Widget _buildBody(ThemeData theme) {
    if (_loading && _data == null) {
      return const AppLoading();
    }
    if (_error != null && _data == null) {
      return AppErrorView(message: _error!, onRetry: _load);
    }

    final items = _data?.items ?? const <StockItem>[];
    if (items.isEmpty) {
      return AppEmpty(
        icon: Icons.inventory_2_outlined,
        title: 'Chưa có tồn kho',
        body: 'Sản phẩm trong danh mục sẽ tự có dòng tồn 0 khi được đồng bộ. '
            'Nhập kho để cộng số lượng thực tế.',
        action: AppButton.primary(
          label: 'Nhập kho mới',
          icon: Icons.add_rounded,
          onPressed: () => _openMovement(
            type: StockMovementType.inn,
            createNew: true,
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
      shape: RoundedRectangleBorder(
        borderRadius: AppRadius.md,
        side: BorderSide(color: theme.colorScheme.outline),
      ),
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

/// Shown when catalog could not be loaded and the admin has to type the id.
/// A typo here creates a stock row checkout can never match, so it is called out.
class _ManualEntryWarning extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.errorContainer,
        borderRadius: AppRadius.sm,
      ),
      child: Text(
        'Không tải được danh mục sản phẩm. Mã nhập tay phải trùng đúng id '
        'sản phẩm, nếu sai thì đơn hàng sẽ báo hết tồn kho.',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onErrorContainer,
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
        borderRadius: AppRadius.sm,
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
    this.products = const [],
    this.stockedProductIds = const {},
  });

  final StockMovementType type;
  final StockItem? item;
  final bool createNew;

  /// Catalog products offered when creating a stock row. Empty means catalog
  /// was unreachable — the dialog then falls back to a manual product_id.
  final List<Product> products;

  /// Products that already have a stock row; excluded from the picker so the
  /// admin uses "Nhập kho" on the existing row instead of creating a duplicate.
  final Set<String> stockedProductIds;

  @override
  State<_MovementDialog> createState() => _MovementDialogState();
}

class _MovementDialogState extends State<_MovementDialog> {
  late final TextEditingController _productIdCtrl;
  late final TextEditingController _skuCtrl;
  late final TextEditingController _nameCtrl;
  Product? _selectedProduct;
  late final TextEditingController _qtyCtrl;
  late final TextEditingController _deltaCtrl;
  late final TextEditingController _unitCostCtrl;
  late final TextEditingController _reorderCtrl;
  late final TextEditingController _noteCtrl;
  String? _localError;

  bool get _needsCreateFields => widget.createNew || widget.item == null;

  /// Catalog products that do not have a stock row yet.
  List<Product> get _selectableProducts => widget.products
      .where((p) => !widget.stockedProductIds.contains(p.id))
      .toList();

  /// True when the admin must type product_id by hand: catalog unreachable, or
  /// empty. Picking from catalog is the only way to guarantee the stock row's
  /// product_id matches the one checkout reserves against.
  bool get _manualProductEntry => widget.products.isEmpty;

  /// Nothing to submit when every catalog product is already stocked — the
  /// admin has to go to the existing row (or add the product first).
  bool get _canSubmit {
    final creating = widget.item == null || widget.createNew;
    if (!creating || _manualProductEntry) return true;
    return _selectableProducts.isNotEmpty;
  }

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

    final creating = widget.item == null || widget.createNew;
    if (creating && !_manualProductEntry) {
      final picked = _selectedProduct;
      if (picked == null) {
        setState(() => _localError = 'Chọn sản phẩm từ danh mục.');
        return;
      }
      // sku/name come from catalog so the stock row can never drift from it.
      _productIdCtrl.text = picked.id;
      _skuCtrl.text = picked.sku;
      _nameCtrl.text = picked.name;
    }

    final productId = _productIdCtrl.text.trim();
    if (productId.isEmpty) {
      setState(() => _localError = 'Chọn sản phẩm từ danh mục.');
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
            setState(
                () => _localError = 'SKU và tên bắt buộc khi tạo tồn mới.');
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
                if (_manualProductEntry) ...[
                  _ManualEntryWarning(),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _productIdCtrl,
                    decoration: const InputDecoration(
                      labelText: 'Mã sản phẩm (product_id)',
                      helperText: 'Phải trùng id trong danh mục sản phẩm',
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
                  ],
                ]
                // Defensive: the FAB is hidden when nothing is selectable, so
                // this is only reachable if the list refreshed while the dialog
                // was being opened.
                else if (_selectableProducts.isEmpty) ...[
                  Text(
                    'Mọi sản phẩm trong danh mục đã có dòng tồn. '
                    'Dùng «Nhập kho» trên dòng có sẵn, hoặc thêm sản phẩm mới '
                    'ở màn Sản phẩm trước.',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 12),
                ] else ...[
                  DropdownButtonFormField<Product>(
                    initialValue: _selectedProduct,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: 'Sản phẩm',
                      border: OutlineInputBorder(),
                    ),
                    items: [
                      for (final p in _selectableProducts)
                        DropdownMenuItem<Product>(
                          value: p,
                          child: Text(
                            p.active
                                ? '${p.name} · ${p.sku}'
                                : '${p.name} · ${p.sku} (ngừng bán)',
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                    ],
                    onChanged: (p) => setState(() {
                      _selectedProduct = p;
                      _localError = null;
                    }),
                  ),
                  const SizedBox(height: 12),
                ],
                if (type == StockMovementType.inn) ...[
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
          onPressed: _canSubmit ? _submit : null,
          child: const Text('Xác nhận'),
        ),
      ],
    );
  }
}
