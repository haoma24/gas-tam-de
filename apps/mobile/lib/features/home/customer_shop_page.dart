import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';
import '../auth/auth_session.dart';
import '../auth/me_api.dart';
import '../catalog/catalog_api.dart';
import '../catalog/catalog_models.dart';
import '../catalog/product_image.dart';
import '../order/customer_order_prefill.dart';
import '../order/last_order.dart';
import '../order/order_address_selection.dart';
import '../order/order_cart.dart';
import '../order/geo_models.dart';

/// Customer home — greeting, one-tap reorder, product list.
///
/// The old page opened with a full-viewport gradient hero, three static
/// marketing chips and an image grid before a single price was visible. For a
/// shop with a handful of SKUs the products themselves are the content.
class CustomerShopPage extends ConsumerStatefulWidget {
  const CustomerShopPage({super.key});

  @override
  ConsumerState<CustomerShopPage> createState() => _CustomerShopPageState();
}

class _CustomerShopPageState extends ConsumerState<CustomerShopPage> {
  List<Product>? _products;
  bool _loading = true;
  String? _error;
  final _searchController = TextEditingController();
  String _query = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final items = await ref.read(catalogApiProvider).listActiveProducts();
      if (!mounted) return;
      setState(() {
        _products = items;
        _loading = false;
      });
    } on CatalogApiException catch (e) {
      _fail(e.displayMessage);
    } catch (_) {
      _fail('Không tải được danh mục.');
    }
  }

  void _fail(String message) {
    if (!mounted) return;
    setState(() {
      _error = message;
      _loading = false;
    });
  }

  List<Product> get _filtered {
    final items = _products ?? const <Product>[];
    final q = _query.trim().toLowerCase();
    if (q.isEmpty) return items;
    return items
        .where(
          (p) =>
              p.name.toLowerCase().contains(q) ||
              p.sku.toLowerCase().contains(q) ||
              (p.description?.toLowerCase().contains(q) ?? false),
        )
        .toList(growable: false);
  }

  /// The shop can only call the customer back if a phone number is on file.
  bool _requirePhone() {
    final session = ref.read(authSessionProvider);
    final profile = ref.read(customerProfileProvider).valueOrNull;
    final phone = profile?.phoneMasked ?? session?.user.phoneMasked ?? '';
    if (phone.trim().isNotEmpty) return true;
    context.go('/profile');
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        const SnackBar(
            content: Text('Thêm số điện thoại để cửa hàng gọi giao.')),
      );
    return false;
  }

  void _startOrder() {
    if (_requirePhone()) context.push('/order');
  }

  void _addAndOrder(Product product) {
    if (!_requirePhone()) return;
    ref.read(orderCartProvider.notifier).increment(product);
    context.push('/order');
  }

  /// Refills the cart and the delivery pin from the customer's last order, so
  /// a repeat purchase is «Đặt lại» → «Đặt đơn» instead of a four-screen funnel.
  void _reorder(LastOrderSummary last) {
    if (!_requirePhone()) return;
    final byId = {for (final p in _products ?? const <Product>[]) p.id: p};
    final cart = ref.read(orderCartProvider.notifier);
    cart.clear();
    var added = 0;
    for (final line in last.items) {
      final product = byId[line.productId];
      if (product == null) continue; // delisted since the last order
      cart.setQuantity(product, line.qty);
      added++;
    }
    if (added == 0) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(
          const SnackBar(
            content: Text('Sản phẩm của đơn trước không còn bán.'),
          ),
        );
      return;
    }
    final prefill = ref.read(customerOrderPrefillProvider).valueOrNull;
    final defaults = prefill?.lastAddress;
    if (prefill?.hasLastAddress ?? false) {
      ref.read(orderAddressProvider.notifier).select(
            SelectedAddress(
              lat: defaults!.lat!,
              lng: defaults.lng!,
              label: defaults.addressText!,
            ),
          );
    }
    context.push('/order');
  }

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final profile = ref.watch(customerProfileProvider).valueOrNull;
    final name = profile?.fullName?.trim();
    final greeting = (name != null && name.isNotEmpty) ? name : 'bạn';
    final lastOrder = ref.watch(lastOrderProvider).valueOrNull;

    return Scaffold(
      backgroundColor: p.bg,
      body: SafeArea(
        bottom: false,
        child: RefreshIndicator(
          onRefresh: () async {
            ref.invalidate(lastOrderProvider);
            await _load();
          },
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(
              AppSpacing.lg,
              AppSpacing.lg,
              AppSpacing.lg,
              96, // clears the FAB
            ),
            children: [
              Text(
                'Chào $greeting',
                style: context.text.headlineSmall,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const VGap(AppSpacing.xs),
              Text(
                'Giao gas tận nơi, rõ phí trước khi đặt.',
                style: context.text.bodyMedium?.copyWith(color: p.inkMuted),
              ),
              if (lastOrder != null) ...[
                const VGap(AppSpacing.lg),
                _ReorderCard(
                  last: lastOrder,
                  onReorder: () => _reorder(lastOrder),
                ),
              ],
              const VGap(AppSpacing.xl),
              AppSearchField(
                controller: _searchController,
                hint: 'Tìm bình gas, phụ kiện…',
                onChanged: (v) => setState(() => _query = v),
              ),
              const VGap(AppSpacing.xl),
              AppSectionTitle(
                'Sản phẩm',
                trailing: _loading && _products != null
                    ? const AppInlineSpinner(size: 14)
                    : null,
              ),
              ..._productList(),
            ],
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _startOrder,
        icon: const Icon(Icons.add_rounded),
        label: const Text('Đặt giao gas'),
      ),
    );
  }

  List<Widget> _productList() {
    if (_loading && _products == null) {
      return [const Padding(padding: EdgeInsets.all(48), child: AppLoading())];
    }
    if (_error != null && _products == null) {
      return [
        Padding(
          padding: const EdgeInsets.symmetric(vertical: AppSpacing.xl),
          child: AppErrorView(message: _error!, onRetry: _load),
        ),
      ];
    }
    final items = _filtered;
    if (items.isEmpty) {
      return [
        Padding(
          padding: const EdgeInsets.symmetric(vertical: AppSpacing.xl),
          child: AppEmpty(
            icon: Icons.inventory_2_outlined,
            title: _query.isEmpty
                ? 'Cửa hàng chưa mở bán sản phẩm'
                : 'Không tìm thấy sản phẩm',
            body: _query.isEmpty ? null : 'Thử một từ khoá khác.',
          ),
        ),
      ];
    }
    return [
      for (var i = 0; i < items.length; i++) ...[
        if (i > 0) const VGap(AppSpacing.sm),
        _ProductRow(
          product: items[i],
          onOpen: () =>
              context.push('/products/${items[i].id}', extra: items[i]),
          onAdd: () => _addAndOrder(items[i]),
        ),
      ],
    ];
  }
}

/// One-tap repeat of the previous order.
class _ReorderCard extends StatelessWidget {
  const _ReorderCard({required this.last, required this.onReorder});

  final LastOrderSummary last;
  final VoidCallback onReorder;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return AppSection(
      children: [
        Row(
          children: [
            Icon(Icons.history_rounded, size: 18, color: p.inkMuted),
            const HGap(AppSpacing.sm),
            Text(
              'Đơn gần nhất',
              style: context.text.labelLarge?.copyWith(color: p.inkMuted),
            ),
          ],
        ),
        const VGap(AppSpacing.sm),
        Text(last.summaryLine, style: context.text.bodyLarge),
        if (last.addressText.isNotEmpty) ...[
          const VGap(2),
          Text(
            last.addressText,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: context.text.bodySmall?.copyWith(color: p.inkMuted),
          ),
        ],
        const VGap(AppSpacing.lg),
        Row(
          children: [
            MoneyText(last.total, emphasis: MoneyEmphasis.total),
            const Spacer(),
            AppButton.primary(label: 'Đặt lại', onPressed: onReorder),
          ],
        ),
      ],
    );
  }
}

/// Compact product row — image, name, price, add.
class _ProductRow extends StatelessWidget {
  const _ProductRow({
    required this.product,
    required this.onOpen,
    required this.onAdd,
  });

  final Product product;
  final VoidCallback onOpen;
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Material(
      color: p.surface,
      borderRadius: AppRadius.md,
      child: InkWell(
        onTap: onOpen,
        borderRadius: AppRadius.md,
        child: Ink(
          decoration: BoxDecoration(
            borderRadius: AppRadius.md,
            border: Border.all(color: p.border),
          ),
          padding: const EdgeInsets.all(AppSpacing.md),
          child: Row(
            children: [
              SizedBox(
                width: 56,
                height: 56,
                child: ProductImage(
                  product: product,
                  borderRadius: AppRadius.sm,
                ),
              ),
              const HGap(AppSpacing.md),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      product.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: context.text.bodyLarge?.copyWith(
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const VGap(2),
                    Row(
                      children: [
                        MoneyText(product.salePrice),
                        Text(
                          ' / ${product.unit}',
                          style: context.text.bodySmall?.copyWith(
                            color: p.inkMuted,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              const HGap(AppSpacing.sm),
              IconButton(
                onPressed: onAdd,
                tooltip: 'Thêm vào đơn',
                icon: const Icon(Icons.add_rounded),
                style: IconButton.styleFrom(
                  backgroundColor: p.surfaceSubtle,
                  foregroundColor: p.ink,
                  shape: RoundedRectangleBorder(
                    borderRadius: AppRadius.sm,
                    side: BorderSide(color: p.border),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
