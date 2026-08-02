import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'catalog_api.dart';
import 'catalog_models.dart';

/// Create or edit a catalog product (admin).
class AdminProductFormPage extends ConsumerStatefulWidget {
  const AdminProductFormPage({
    super.key,
    required this.onDone,
    required this.onBack,
    this.productId,
  });

  /// When null → create mode; otherwise load + patch.
  final String? productId;
  final VoidCallback onDone;
  final VoidCallback onBack;

  bool get isEdit => productId != null && productId!.isNotEmpty;

  @override
  ConsumerState<AdminProductFormPage> createState() =>
      _AdminProductFormPageState();
}

class _AdminProductFormPageState extends ConsumerState<AdminProductFormPage> {
  final _formKey = GlobalKey<FormState>();
  final _skuController = TextEditingController();
  final _nameController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _unitController = TextEditingController(text: 'binh');
  final _priceController = TextEditingController();
  final _imageUrlController = TextEditingController();

  bool _active = true;
  bool _loading = false;
  bool _bootstrapping = false;
  String? _error;
  Product? _original;

  @override
  void initState() {
    super.initState();
    if (widget.isEdit) {
      _bootstrapping = true;
      _loadExisting();
    }
  }

  @override
  void dispose() {
    _skuController.dispose();
    _nameController.dispose();
    _descriptionController.dispose();
    _unitController.dispose();
    _priceController.dispose();
    _imageUrlController.dispose();
    super.dispose();
  }

  Future<void> _loadExisting() async {
    try {
      final p =
          await ref.read(catalogApiProvider).getAdminProduct(widget.productId!);
      if (!mounted) return;
      _original = p;
      _skuController.text = p.sku;
      _nameController.text = p.name;
      _descriptionController.text = p.description ?? '';
      _unitController.text = p.unit;
      _priceController.text = p.salePrice.toString();
      _imageUrlController.text = p.imageUrl ?? '';
      setState(() {
        _active = p.active;
        _bootstrapping = false;
      });
    } on CatalogApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _bootstrapping = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Có lỗi xảy ra. Thử lại.';
        _bootstrapping = false;
      });
    }
  }

  int? _parsePrice(String raw) {
    final cleaned = raw.replaceAll(RegExp(r'[^\d]'), '');
    if (cleaned.isEmpty) return null;
    return int.tryParse(cleaned);
  }

  Future<void> _submit() async {
    setState(() => _error = null);
    if (!(_formKey.currentState?.validate() ?? false)) return;

    final sku = _skuController.text.trim();
    final name = _nameController.text.trim();
    final unit = _unitController.text.trim().isEmpty
        ? 'binh'
        : _unitController.text.trim();
    final price = _parsePrice(_priceController.text);
    if (price == null) {
      setState(() => _error = 'Giá bán không hợp lệ.');
      return;
    }
    final description = _descriptionController.text.trim();
    final imageUrl = _imageUrlController.text.trim();

    setState(() => _loading = true);
    try {
      final api = ref.read(catalogApiProvider);
      if (widget.isEdit) {
        await api.patchProduct(
          widget.productId!,
          sku: sku,
          name: name,
          unit: unit,
          salePrice: price,
          active: _active,
          description: description,
          imageUrl: imageUrl,
        );
      } else {
        await api.createProduct(
          sku: sku,
          name: name,
          unit: unit,
          salePrice: price,
          active: _active,
          description: description.isEmpty ? null : description,
          imageUrl: imageUrl.isEmpty ? null : imageUrl,
        );
      }
      if (!mounted) return;
      widget.onDone();
    } on CatalogApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.displayMessage);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Có lỗi xảy ra. Thử lại.');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final title = widget.isEdit ? 'Sửa sản phẩm' : 'Thêm sản phẩm';

    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: _loading ? null : widget.onBack,
        ),
      ),
      body: SafeArea(
        child: _bootstrapping
            ? const Center(child: CircularProgressIndicator())
            : (_original == null && widget.isEdit && _error != null)
                ? Center(
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
                            onPressed: widget.onBack,
                            child: const Text('Quay lại'),
                          ),
                        ],
                      ),
                    ),
                  )
                : Form(
                    key: _formKey,
                    child: ListView(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 24,
                        vertical: 16,
                      ),
                      children: [
                        Text(
                          widget.isEdit
                              ? 'Cập nhật thông tin và giá bán.'
                              : 'Nhập mã, tên và giá bán cho sản phẩm mới.',
                          style: theme.textTheme.bodyLarge?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 24),
                        TextFormField(
                          controller: _skuController,
                          enabled: !_loading,
                          textInputAction: TextInputAction.next,
                          decoration: const InputDecoration(
                            labelText: 'Mã SKU',
                            hintText: 'GAS12',
                            border: OutlineInputBorder(),
                          ),
                          validator: (value) {
                            if ((value?.trim() ?? '').isEmpty) {
                              return 'Vui lòng nhập mã SKU.';
                            }
                            return null;
                          },
                        ),
                        const SizedBox(height: 16),
                        TextFormField(
                          controller: _nameController,
                          enabled: !_loading,
                          textInputAction: TextInputAction.next,
                          decoration: const InputDecoration(
                            labelText: 'Tên sản phẩm',
                            hintText: 'Gas 12kg',
                            border: OutlineInputBorder(),
                          ),
                          validator: (value) {
                            if ((value?.trim() ?? '').isEmpty) {
                              return 'Vui lòng nhập tên sản phẩm.';
                            }
                            return null;
                          },
                        ),
                        const SizedBox(height: 16),
                        TextFormField(
                          controller: _descriptionController,
                          enabled: !_loading,
                          textInputAction: TextInputAction.next,
                          maxLines: 2,
                          decoration: const InputDecoration(
                            labelText: 'Mô tả (tuỳ chọn)',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 16),
                        TextFormField(
                          controller: _unitController,
                          enabled: !_loading,
                          textInputAction: TextInputAction.next,
                          decoration: const InputDecoration(
                            labelText: 'Đơn vị',
                            hintText: 'binh',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 16),
                        TextFormField(
                          controller: _priceController,
                          enabled: !_loading,
                          keyboardType: TextInputType.number,
                          textInputAction: TextInputAction.next,
                          inputFormatters: [
                            FilteringTextInputFormatter.digitsOnly,
                          ],
                          decoration: const InputDecoration(
                            labelText: 'Giá bán (VND)',
                            hintText: '450000',
                            border: OutlineInputBorder(),
                            prefixIcon: Icon(Icons.payments_outlined),
                          ),
                          validator: (value) {
                            final price = _parsePrice(value ?? '');
                            if (price == null) {
                              return 'Vui lòng nhập giá bán.';
                            }
                            if (price < 0) {
                              return 'Giá bán phải ≥ 0.';
                            }
                            return null;
                          },
                        ),
                        const SizedBox(height: 16),
                        TextFormField(
                          controller: _imageUrlController,
                          enabled: !_loading,
                          textInputAction: TextInputAction.done,
                          decoration: const InputDecoration(
                            labelText: 'URL ảnh (tuỳ chọn)',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 8),
                        SwitchListTile(
                          contentPadding: EdgeInsets.zero,
                          title: const Text('Đang bán'),
                          subtitle: Text(
                            _active
                                ? 'Khách có thể chọn sản phẩm này.'
                                : 'Ẩn khỏi danh sách bán (không xóa).',
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                          value: _active,
                          onChanged: _loading
                              ? null
                              : (v) => setState(() => _active = v),
                        ),
                        if (_error != null) ...[
                          const SizedBox(height: 8),
                          Text(
                            _error!,
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: theme.colorScheme.error,
                            ),
                          ),
                        ],
                        const SizedBox(height: 24),
                        FilledButton(
                          onPressed: _loading ? null : _submit,
                          style: FilledButton.styleFrom(
                            minimumSize: const Size.fromHeight(56),
                            textStyle: theme.textTheme.titleMedium?.copyWith(
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          child: _loading
                              ? SizedBox(
                                  height: 22,
                                  width: 22,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2.5,
                                    color: theme.colorScheme.onPrimary,
                                  ),
                                )
                              : Text(widget.isEdit ? 'Lưu thay đổi' : 'Tạo sản phẩm'),
                        ),
                      ],
                    ),
                  ),
      ),
    );
  }
}
