import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import '../order/order_cart.dart';
import 'catalog_api.dart';
import 'catalog_models.dart';
import 'product_image.dart';

class ProductDetailRoutePage extends ConsumerStatefulWidget {
  const ProductDetailRoutePage({
    super.key,
    required this.productId,
    required this.onBack,
    required this.onCheckout,
    this.initialProduct,
  });

  final String productId;
  final Product? initialProduct;
  final VoidCallback onBack;
  final VoidCallback onCheckout;

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
    if (product != null) {
      return ProductDetailPage(
        product: product,
        onBack: widget.onBack,
        onCheckout: widget.onCheckout,
      );
    }
    return Scaffold(
      backgroundColor: AppColors.surface0,
      appBar: AppBar(leading: BackButton(onPressed: widget.onBack)),
      body: Center(
        child: _error == null
            ? const CircularProgressIndicator(color: AppColors.fire)
            : Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(_error!),
                  const SizedBox(height: 12),
                  OutlinedButton(
                      onPressed: _load, child: const Text('Thử lại')),
                ],
              ),
      ),
    );
  }
}

class ProductDetailPage extends ConsumerStatefulWidget {
  const ProductDetailPage({
    super.key,
    required this.product,
    required this.onBack,
    required this.onCheckout,
  });

  final Product product;
  final VoidCallback onBack;
  final VoidCallback onCheckout;

  @override
  ConsumerState<ProductDetailPage> createState() => _ProductDetailPageState();
}

class _ProductDetailPageState extends ConsumerState<ProductDetailPage> {
  final _pageController = PageController();
  int _page = 0;

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final product = widget.product;
    final images = product.galleryImages;
    final quantity = ref.watch(orderCartProvider).quantityOf(product.id);

    return Scaffold(
      backgroundColor: AppColors.surface0,
      body: CustomScrollView(
        slivers: [
          SliverAppBar(
            pinned: true,
            expandedHeight: 350,
            backgroundColor: AppColors.surface0,
            leading: IconButton.filledTonal(
              onPressed: widget.onBack,
              icon: const Icon(Icons.arrow_back_rounded),
            ),
            flexibleSpace: FlexibleSpaceBar(
              background: Stack(
                fit: StackFit.expand,
                children: [
                  PageView.builder(
                    controller: _pageController,
                    itemCount: images.isEmpty ? 1 : images.length,
                    onPageChanged: (value) => setState(() => _page = value),
                    itemBuilder: (_, index) => ProductImage(
                      product: product,
                      imageUrl: images.isEmpty ? null : images[index],
                      fit: BoxFit.contain,
                    ),
                  ),
                  if (images.length > 1)
                    Positioned(
                      left: 0,
                      right: 0,
                      bottom: 18,
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: List.generate(
                          images.length,
                          (index) => AnimatedContainer(
                            duration: const Duration(milliseconds: 180),
                            width: index == _page ? 22 : 7,
                            height: 7,
                            margin: const EdgeInsets.symmetric(horizontal: 3),
                            decoration: BoxDecoration(
                              color: index == _page
                                  ? AppColors.fire
                                  : AppColors.ash.withValues(alpha: .45),
                              borderRadius: AppRadius.pill,
                            ),
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
          SliverPadding(
            padding: const EdgeInsets.fromLTRB(20, 24, 20, 140),
            sliver: SliverList.list(
              children: [
                Text(
                  product.name,
                  style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.w900,
                        letterSpacing: -.5,
                      ),
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        formatVnd(product.salePrice),
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                              color: AppColors.fire,
                              fontWeight: FontWeight.w900,
                            ),
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 12,
                        vertical: 7,
                      ),
                      decoration: BoxDecoration(
                        color: AppColors.amber.withValues(alpha: .14),
                        borderRadius: AppRadius.pill,
                      ),
                      child: Text(
                        'Theo ${product.unit}',
                        style: const TextStyle(fontWeight: FontWeight.w700),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 28),
                Text(
                  'Thông tin sản phẩm',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w800,
                      ),
                ),
                const SizedBox(height: 10),
                Text(
                  product.description?.trim().isNotEmpty == true
                      ? product.description!.trim()
                      : 'Sản phẩm chính hãng, được kiểm tra an toàn trước khi giao đến khách hàng.',
                  style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                        color: const Color(0xFF57534E),
                        height: 1.6,
                      ),
                ),
                const SizedBox(height: 24),
                const Wrap(
                  spacing: 10,
                  runSpacing: 10,
                  children: [
                    _Benefit(icon: Icons.verified_outlined, text: 'Chính hãng'),
                    _Benefit(
                        icon: Icons.shield_outlined, text: 'Kiểm tra an toàn'),
                    _Benefit(icon: Icons.bolt_rounded, text: 'Giao nhanh'),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
      bottomNavigationBar: Material(
        color: AppColors.surface0,
        elevation: 16,
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
            child: Row(
              children: [
                if (quantity > 0) ...[
                  IconButton.filledTonal(
                    onPressed: () =>
                        ref.read(orderCartProvider.notifier).decrement(product),
                    icon: const Icon(Icons.remove_rounded),
                  ),
                  SizedBox(
                    width: 38,
                    child: Text(
                      '$quantity',
                      textAlign: TextAlign.center,
                      style: const TextStyle(fontWeight: FontWeight.w900),
                    ),
                  ),
                ],
                Expanded(
                  child: FilledButton.icon(
                    onPressed: () {
                      if (quantity > 0) {
                        widget.onCheckout();
                      } else {
                        ref.read(orderCartProvider.notifier).increment(product);
                      }
                    },
                    style: FilledButton.styleFrom(
                      minimumSize: const Size.fromHeight(54),
                      backgroundColor: AppColors.fire,
                      foregroundColor: Colors.white,
                    ),
                    icon: Icon(quantity > 0
                        ? Icons.arrow_forward_rounded
                        : Icons.add_shopping_cart_rounded),
                    label: Text(
                        quantity > 0 ? 'Tiếp tục đặt hàng' : 'Thêm vào đơn'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _Benefit extends StatelessWidget {
  const _Benefit({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
      decoration: const BoxDecoration(
        color: AppColors.surface1,
        borderRadius: AppRadius.pill,
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 17, color: AppColors.fire),
          const SizedBox(width: 6),
          Text(text, style: const TextStyle(fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}
