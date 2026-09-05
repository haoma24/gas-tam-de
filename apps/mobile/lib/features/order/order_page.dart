import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';
import '../auth/auth_session.dart';
import '../auth/me_api.dart';
import '../catalog/catalog_api.dart';
import '../catalog/catalog_models.dart';
import '../inventory/stock_levels_api.dart';
import 'customer_order_prefill.dart';
import 'geo_models.dart';
import 'last_order.dart';
import 'order_address_selection.dart';
import 'order_api.dart';
import 'order_cart.dart';
import 'order_models.dart';

/// The whole order in one screen: products, address, recipient, price.
///
/// Replaces the `/order` → `/order/address` → `/order/review` funnel. The
/// address picker is still its own screen, but it is now a detour taken only
/// when the prefilled address is wrong — not a mandatory step. PRD §2.1: the
/// customer is usually in a hurry and "không muốn form dài".
class OrderPage extends ConsumerStatefulWidget {
  const OrderPage({super.key});

  @override
  ConsumerState<OrderPage> createState() => _OrderPageState();
}

class _OrderPageState extends ConsumerState<OrderPage> {
  final _nameController = TextEditingController();
  final _nameFocus = FocusNode();

  List<Product>? _products;
  Map<String, int> _stock = const {};
  bool _catalogLoading = true;
  String? _catalogError;

  OrderQuote? _quote;
  bool _quoteLoading = false;
  bool _submitting = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await _loadCatalog();
      if (!mounted) return;
      await _applyPrefill();
      if (!mounted) return;
      await _refreshQuote();
    });
  }

  @override
  void dispose() {
    _nameController.dispose();
    _nameFocus.dispose();
    super.dispose();
  }

  Future<void> _loadCatalog() async {
    setState(() {
      _catalogLoading = true;
      _catalogError = null;
    });
    try {
      final products = await ref.read(catalogApiProvider).listActiveProducts();
      // Stock is advisory here — the server is the authority at place time.
      Map<String, int> stock = const {};
      try {
        stock = await ref.read(stockLevelsApiProvider).levels();
      } catch (_) {}
      if (!mounted) return;
      setState(() {
        _products = products;
        _stock = stock;
        _catalogLoading = false;
      });
    } on CatalogApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _catalogError = e.displayMessage;
        _catalogLoading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _catalogError = 'Không tải được danh mục.';
        _catalogLoading = false;
      });
    }
  }

  /// Fills the recipient name and, when nothing is chosen yet, the delivery
  /// address from the customer's last order.
  Future<void> _applyPrefill() async {
    try {
      final prefill = await ref.read(customerOrderPrefillProvider.future);
      if (!mounted) return;
      if (prefill.hasName && _nameController.text.trim().isEmpty) {
        _nameController.text = prefill.fullName!.trim();
      }
      if (ref.read(orderAddressProvider) == null && prefill.hasLastAddress) {
        final d = prefill.lastAddress!;
        ref.read(orderAddressProvider.notifier).select(
              SelectedAddress(
                lat: d.lat!,
                lng: d.lng!,
                label: d.addressText!,
              ),
            );
      }
      setState(() {});
    } catch (_) {
      // Prefill is a convenience; the form still works empty.
    }
  }

  List<CreateOrderItem> _cartItems() => ref
      .read(orderCartProvider)
      .lines
      .map((l) => CreateOrderItem(productId: l.product.id, qty: l.quantity))
      .toList();

  String _fmtKm(double km) => km.toStringAsFixed(1).replaceAll('.', ',');

  String _outOfRangeMessage(OrderQuote q) =>
      'Địa chỉ ngoài phạm vi giao (khoảng ${_fmtKm(q.distanceKm)} km, '
      'tối đa ${_fmtKm(q.maxRadiusKm)} km). Chọn vị trí gần hơn.';

  Future<void> _refreshQuote() async {
    final cart = ref.read(orderCartProvider);
    final address = ref.read(orderAddressProvider);
    if (cart.isEmpty || address == null) {
      setState(() {
        _quote = null;
        _quoteLoading = false;
      });
      return;
    }

    setState(() {
      _quoteLoading = true;
      _error = null;
    });
    try {
      final quote = await ref.read(orderApiProvider).quoteOrder(
            QuoteOrderRequest(
              lat: address.lat,
              lng: address.lng,
              items: _cartItems(),
            ),
          );
      if (!mounted) return;
      setState(() {
        _quote = quote;
        _quoteLoading = false;
        _error = quote.inRange ? null : _outOfRangeMessage(quote);
      });
    } on OrderApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _quote = null;
        _quoteLoading = false;
        _error = e.displayMessage;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _quote = null;
        _quoteLoading = false;
        _error = 'Không lấy được báo giá. Thử lại.';
      });
    }
  }

  void _setQuantity(Product product, int qty) {
    ref.read(orderCartProvider.notifier).setQuantity(product, qty);
    _refreshQuote();
  }

  Future<void> _pickAddress() async {
    await context.push<void>('/order/address');
    if (!mounted) return;
    await _refreshQuote();
  }

  Future<void> _placeOrder() async {
    final name = _nameController.text.trim();
    if (name.isEmpty) {
      setState(() => _error = 'Nhập tên người nhận.');
      _nameFocus.requestFocus();
      return;
    }

    final cart = ref.read(orderCartProvider);
    final address = ref.read(orderAddressProvider);
    final session = ref.read(authSessionProvider);

    if (cart.isEmpty) {
      setState(() => _error = 'Chưa chọn sản phẩm nào.');
      return;
    }
    if (address == null) {
      setState(() => _error = 'Chưa chọn địa chỉ giao.');
      return;
    }
    if (session == null) {
      setState(
        () => _error = 'Chưa đăng nhập. Đăng nhập lại trước khi đặt đơn.',
      );
      return;
    }

    setState(() {
      _submitting = true;
      _error = null;
    });

    try {
      // Fresh quote before place so totals match the server fee engine.
      final quote = await ref.read(orderApiProvider).quoteOrder(
            QuoteOrderRequest(
              lat: address.lat,
              lng: address.lng,
              items: _cartItems(),
            ),
          );
      if (!mounted) return;
      setState(() => _quote = quote);
      if (!quote.inRange) {
        setState(() {
          _submitting = false;
          _error = _outOfRangeMessage(quote);
        });
        return;
      }

      final order = await ref.read(orderApiProvider).createOrder(
            CreateOrderRequest(
              customerName: name,
              addressText: address.label,
              lat: address.lat,
              lng: address.lng,
              items: _cartItems(),
            ),
          );
      if (!mounted) return;
      try {
        await ref.read(meApiProvider).patchFullName(name);
      } catch (_) {
        // Order already placed — profile sync is best-effort.
      }
      ref.invalidate(customerProfileProvider);
      ref.invalidate(customerOrderPrefillProvider);
      ref.invalidate(lastOrderProvider);
      ref.read(orderCartProvider.notifier).clear();
      ref.read(orderAddressProvider.notifier).clear();
      ref.read(orderGeoCheckProvider.notifier).clear();
      if (!mounted) return;
      context.pushReplacement('/order/success', extra: order);
    } on OrderApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _error = e.displayMessage;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _error = 'Không đặt được đơn. Thử lại.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final cart = ref.watch(orderCartProvider);
    final address = ref.watch(orderAddressProvider);
    final quote = _quote;
    final canPlace = cart.isNotEmpty &&
        address != null &&
        quote != null &&
        quote.inRange &&
        !_submitting &&
        !_quoteLoading;

    return AppScaffold(
      title: 'Đặt giao gas',
      backFallback: '/',
      padBody: false,
      body: ListView(
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.lg,
          AppSpacing.xxl,
        ),
        children: [
          _productSection(cart),
          const VGap(AppSpacing.lg),
          _addressSection(address),
          const VGap(AppSpacing.lg),
          _recipientSection(),
          const VGap(AppSpacing.lg),
          _paymentSection(quote),
          if (_error != null) ...[
            const VGap(AppSpacing.lg),
            AppErrorBanner(message: _error!),
          ],
        ],
      ),
      bottomBar: AppButton.primary(
        label: quote != null && cart.isNotEmpty
            ? 'Đặt đơn · ${formatVnd(quote.total)}'
            : 'Đặt đơn',
        expand: true,
        loading: _submitting,
        onPressed: canPlace ? _placeOrder : null,
      ),
    );
  }

  Widget _productSection(OrderCart cart) {
    if (_catalogLoading && _products == null) {
      return const AppSection(
        title: 'Sản phẩm',
        children: [
          Padding(padding: EdgeInsets.all(AppSpacing.xl), child: AppLoading())
        ],
      );
    }
    if (_catalogError != null && _products == null) {
      return AppSection(
        title: 'Sản phẩm',
        children: [
          AppErrorView(message: _catalogError!, onRetry: _loadCatalog),
        ],
      );
    }

    final products = _products ?? const <Product>[];
    return AppSection(
      title: 'Sản phẩm',
      trailing: cart.isEmpty
          ? null
          : Text(
              '${cart.totalQuantity} sp',
              style: context.text.bodySmall?.copyWith(
                color: context.palette.inkMuted,
              ),
            ),
      children: [
        if (products.isEmpty)
          const AppEmpty(
            icon: Icons.inventory_2_outlined,
            title: 'Cửa hàng chưa mở bán sản phẩm',
          )
        else
          for (var i = 0; i < products.length; i++) ...[
            if (i > 0) const Divider(),
            _ProductPickRow(
              product: products[i],
              quantity: cart.quantityOf(products[i].id),
              onHand: _stock[products[i].id],
              onChanged: (q) => _setQuantity(products[i], q),
            ),
          ],
      ],
    );
  }

  Widget _addressSection(SelectedAddress? address) {
    final p = context.palette;
    return AppSection(
      title: 'Giao đến',
      trailing: AppButton.text(
        label: address == null ? 'Chọn' : 'Đổi',
        onPressed: _pickAddress,
      ),
      children: [
        if (address == null)
          Text(
            'Chưa chọn địa chỉ giao.',
            style: context.text.bodyLarge?.copyWith(color: p.inkMuted),
          )
        else ...[
          Text(address.label, style: context.text.bodyLarge),
          if (_quote != null) ...[
            const VGap(AppSpacing.sm),
            AppBadge(
              _quote!.inRange
                  ? 'Trong phạm vi · ${_fmtKm(_quote!.distanceKm)} km'
                  : 'Ngoài phạm vi · ${_fmtKm(_quote!.distanceKm)} km',
              tone:
                  _quote!.inRange ? AppBadgeTone.success : AppBadgeTone.danger,
              icon: _quote!.inRange
                  ? Icons.check_circle_outline
                  : Icons.error_outline_rounded,
            ),
          ],
        ],
      ],
    );
  }

  Widget _recipientSection() {
    final session = ref.watch(authSessionProvider);
    final profile = ref.watch(customerProfileProvider).valueOrNull;
    final phone = profile?.phoneMasked ?? session?.user.phoneMasked ?? '';

    return AppSection(
      title: 'Người nhận',
      children: [
        AppTextField(
          controller: _nameController,
          focusNode: _nameFocus,
          hint: 'Tên người nhận',
          textInputAction: TextInputAction.done,
          onChanged: (_) => setState(() {}),
        ),
        if (phone.trim().isNotEmpty) ...[
          const VGap(AppSpacing.md),
          AppDataRow(label: 'Số điện thoại', value: phone),
        ],
      ],
    );
  }

  Widget _paymentSection(OrderQuote? quote) {
    if (quote == null) {
      return AppSection(
        title: 'Thanh toán',
        children: [
          Text(
            _quoteLoading
                ? 'Đang tính phí giao…'
                : 'Chọn sản phẩm và địa chỉ để xem tổng tiền.',
            style: context.text.bodyLarge?.copyWith(
              color: context.palette.inkMuted,
            ),
          ),
        ],
      );
    }

    return AppSection(
      title: 'Thanh toán',
      trailing: _quoteLoading ? const AppInlineSpinner(size: 14) : null,
      children: [
        MoneyRow(
          label: 'Khoảng cách',
          valueText: '${_fmtKm(quote.distanceKm)} km',
        ),
        MoneyRow(label: 'Tạm tính', amount: quote.subtotal),
        MoneyRow(label: 'Phí giao hàng', amount: quote.deliveryFee),
        const Divider(),
        MoneyRow(
          label: 'Tổng cộng',
          amount: quote.total,
          emphasis: MoneyEmphasis.total,
        ),
        const VGap(AppSpacing.sm),
        Text(
          'Thanh toán tiền mặt khi nhận hàng.',
          style: context.text.bodySmall?.copyWith(
            color: context.palette.inkMuted,
          ),
        ),
      ],
    );
  }
}

class _ProductPickRow extends StatelessWidget {
  const _ProductPickRow({
    required this.product,
    required this.quantity,
    required this.onHand,
    required this.onChanged,
  });

  final Product product;
  final int quantity;
  final int? onHand;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final soldOut = onHand != null && onHand! <= 0;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.md),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  product.name,
                  style: context.text.bodyLarge?.copyWith(
                    fontWeight: FontWeight.w500,
                    color: soldOut ? p.inkFaint : p.ink,
                  ),
                ),
                const VGap(2),
                Row(
                  children: [
                    MoneyText(
                      product.salePrice,
                      emphasis: MoneyEmphasis.muted,
                    ),
                    Text(
                      ' / ${product.unit}',
                      style: context.text.bodySmall?.copyWith(
                        color: p.inkMuted,
                      ),
                    ),
                    if (onHand != null) ...[
                      const HGap(AppSpacing.sm),
                      Text(
                        soldOut ? '· Tạm hết hàng' : '· còn $onHand',
                        style: context.text.bodySmall?.copyWith(
                          color: soldOut ? p.danger : p.inkMuted,
                        ),
                      ),
                    ],
                  ],
                ),
              ],
            ),
          ),
          const HGap(AppSpacing.md),
          if (soldOut)
            const AppBadge('Hết hàng', tone: AppBadgeTone.danger)
          else
            QtyStepper(
              value: quantity,
              max: onHand,
              compact: true,
              onChanged: onChanged,
            ),
        ],
      ),
    );
  }
}
