import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';
import '../order/order_cart.dart';
import 'catalog_api.dart';
import 'catalog_models.dart';
import 'product_image.dart';

/// Resolves `/products/:id` — uses the product passed as `extra` when the user
/// tapped a row, otherwise refetches (deep link, hard reload).
class ProductDetailRoutePage extends ConsumerStatefulWidget {
  const ProductDetailRoutePage({
    super.key,
    required this.productId,
    this.initialProduct,
  });

  final String productId;
  final Product? initialProduct;

  @override
  ConsumerState<ProductDetailRoutePage> createState() =>
      _ProductDetailRoutePageState();
}

class _ProductDetailRoutePageState
    extends ConsumerState<ProductDetailRoutePage> {
  Product? _product;
  String? _error;

  @override
  void initState() {
    super.initState();
    _product = widget.initialProduct;
    if (_product == null) _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final items = await ref.read(catalogApiProvider).listActiveProducts();
      final product =
          items.where((item) => item.id == widget.productId).firstOrNull;
      if (!mounted) return;
      setState(() {
        _product = product;
        _error = product == null ? 'Sản phẩm không còn được mở bán.' : null;
      });
    } catch (_) {
      if (mounted) setState(() => _error = 'Không tải được sản phẩm.');
    }
  }

  @override
  Widget build(BuildContext context) {
    final product = _product;
    if (product != null) return ProductDetailPage(product: product);

    return AppScaffold(
      body: _error == null
          ? const AppLoading()
          : AppErrorView(message: _error!, onRetry: _load),
    );
  }
}

/// Product detail — one image, the price, and a quantity to add.
///
/// The e-commerce furniture (350px expanding image carousel, dot indicators,
/// static "Chính hãng / An toàn / Giao nhanh" trust chips) is gone: a gas shop
/// sells a handful of SKUs the customer already knows.
class ProductDetailPage extends ConsumerStatefulWidget {
  const ProductDetailPage({super.key, required this.product});

  final Product product;

  @override
  ConsumerState<ProductDetailPage> createState() => _ProductDetailPageState();
}

class _ProductDetailPageState extends ConsumerState<ProductDetailPage> {
  int _qty = 1;

  void _addToOrder() {
    ref.read(orderCartProvider.notifier).setQuantity(widget.product, _qty);
    context.go('/order');
  }

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final product = widget.product;
    final description = product.description?.trim();

    return AppScaffold(
      title: product.name,
      backFallback: '/',
      padBody: false,
      body: ListView(
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.lg,
          AppSpacing.sm,
          AppSpacing.lg,
          AppSpacing.xxl,
        ),
        children: [
          ClipRRect(
            borderRadius: AppRadius.md,
            child: AspectRatio(
              aspectRatio: 4 / 3,
              child: ProductImage(product: product, borderRadius: AppRadius.md),
            ),
          ),
          const VGap(AppSpacing.xl),
          Text(product.name, style: context.text.titleLarge),
          const VGap(AppSpacing.sm),
          Row(
            children: [
              MoneyText(product.salePrice, emphasis: MoneyEmphasis.total),
              Text(
                ' / ${product.unit}',
                style: context.text.bodyMedium?.copyWith(color: p.inkMuted),
              ),
            ],
          ),
          if (description != null && description.isNotEmpty) ...[
            const VGap(AppSpacing.xl),
            Text(
              description,
              style: context.text.bodyLarge?.copyWith(color: p.inkMuted),
            ),
          ],
        ],
      ),
      bottomBar: Row(
        children: [
          QtyStepper(
            value: _qty,
            min: 1,
            onChanged: (v) => setState(() => _qty = v),
          ),
          const HGap(AppSpacing.md),
          Expanded(
            child: AppButton.primary(
              label: 'Thêm vào đơn',
              expand: true,
              onPressed: _addToOrder,
            ),
          ),
        ],
      ),
    );
  }
}
