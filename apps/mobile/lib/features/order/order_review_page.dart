import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_session.dart';
import '../auth/me_api.dart';
import '../catalog/catalog_models.dart';
import 'customer_order_prefill.dart';
import 'order_address_selection.dart';
import 'order_api.dart';
import 'order_cart.dart';
import 'order_models.dart';

/// Order flow step 3 — review cart + address + live quote, then place order.
class OrderReviewPage extends ConsumerStatefulWidget {
  const OrderReviewPage({
    super.key,
    required this.onBack,
    required this.onPlaced,
  });

  final VoidCallback onBack;
  final void Function(PlacedOrder order) onPlaced;

  @override
  ConsumerState<OrderReviewPage> createState() => _OrderReviewPageState();
}

class _OrderReviewPageState extends ConsumerState<OrderReviewPage> {
  final _nameController = TextEditingController();
  final _nameFocus = FocusNode();
  bool _submitting = false;
  bool _quoteLoading = true;
  OrderQuote? _quote;
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await _loadPrefillName();
      if (mounted) await _loadQuote();
    });
  }

  Future<void> _loadPrefillName() async {
    try {
      final prefill = await ref.read(customerOrderPrefillProvider.future);
      if (!mounted) return;
      if (prefill.hasName && _nameController.text.trim().isEmpty) {
        setState(() => _nameController.text = prefill.fullName!.trim());
      }
    } catch (_) {}
  }

  @override
  void dispose() {
    _nameController.dispose();
    _nameFocus.dispose();
    super.dispose();
  }

  List<CreateOrderItem> _cartItems() {
    return ref
        .read(orderCartProvider)
        .lines
        .map(
          (l) => CreateOrderItem(
            productId: l.product.id,
            qty: l.quantity,
          ),
        )
        .toList();
  }

  Future<void> _loadQuote() async {
    final cart = ref.read(orderCartProvider);
    final address = ref.read(orderAddressProvider);
    final session = ref.read(authSessionProvider);

    if (cart.isEmpty) {
      setState(() {
        _quoteLoading = false;
        _quote = null;
        _error = 'Giỏ hàng trống. Quay lại chọn sản phẩm.';
      });
      return;
    }
    if (address == null) {
      setState(() {
        _quoteLoading = false;
        _quote = null;
        _error = 'Chưa chọn địa chỉ giao.';
      });
      return;
    }
    if (session == null) {
      setState(() {
        _quoteLoading = false;
        _quote = null;
        _error = 'Chưa đăng nhập. Vui lòng đăng nhập Google trước khi đặt đơn.';
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
        if (!quote.inRange) {
          _error = 'Địa chỉ ngoài phạm vi giao '
              '(khoảng ${_fmtKm(quote.distanceKm)} km, '
              'tối đa ${_fmtKm(quote.maxRadiusKm)} km). '
              'Quay lại chọn vị trí gần hơn.';
        }
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
      setState(() => _error = 'Giỏ hàng trống. Quay lại chọn sản phẩm.');
      return;
    }
    if (address == null) {
      setState(() => _error = 'Chưa chọn địa chỉ giao.');
      return;
    }
    if (session == null) {
      setState(
        () => _error =
            'Chưa đăng nhập. Vui lòng đăng nhập Google trước khi đặt đơn.',
      );
      return;
    }

    setState(() {
      _submitting = true;
      _error = null;
    });

    try {
      // Fresh quote before place so totals match server fee engine.
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
          _error = 'Địa chỉ ngoài phạm vi giao '
              '(khoảng ${_fmtKm(quote.distanceKm)} km, '
              'tối đa ${_fmtKm(quote.maxRadiusKm)} km). '
              'Quay lại chọn vị trí gần hơn.';
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
      ref.read(orderCartProvider.notifier).clear();
      ref.read(orderAddressProvider.notifier).clear();
      ref.read(orderGeoCheckProvider.notifier).clear();
      widget.onPlaced(order);
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
        _error = 'Có lỗi xảy ra. Thử lại.';
      });
    }
  }

  bool get _canPlace {
    final quote = _quote;
    return !_submitting &&
        !_quoteLoading &&
        quote != null &&
        quote.inRange &&
        ref.read(orderCartProvider).isNotEmpty &&
        ref.read(orderAddressProvider) != null;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cart = ref.watch(orderCartProvider);
    final address = ref.watch(orderAddressProvider);
    final geo = ref.watch(orderGeoCheckProvider);
    final session = ref.watch(authSessionProvider);
    final quote = _quote;
    final prefillAsync = ref.watch(customerOrderPrefillProvider);
    final prefillHint = prefillAsync.maybeWhen(
      data: (p) => p.hasName
          ? 'Đã nhớ tên trong hồ sơ — chỉnh nếu cần.'
          : 'Lần đầu đặt gas: nhập họ tên để cửa hàng gọi giao.',
      orElse: () => 'Lần đầu đặt gas: nhập họ tên để cửa hàng gọi giao.',
    );

    final subtotal = quote?.subtotal ?? cart.totalAmount;
    final deliveryFee = quote?.deliveryFee;
    final total = quote?.total ?? (subtotal + (deliveryFee ?? 0));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Xác nhận đơn'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: _submitting ? null : widget.onBack,
        ),
        actions: [
          IconButton(
            tooltip: 'Tải lại báo giá',
            onPressed: (_submitting || _quoteLoading) ? null : _loadQuote,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          children: [
            Text(
              'Kiểm tra trước khi đặt',
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              'Đơn sẽ được gửi tới cửa hàng sau khi xác nhận.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            _SectionCard(
              title: 'Sản phẩm',
              child: cart.isEmpty
                  ? Text(
                      'Giỏ trống',
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: theme.colorScheme.error,
                      ),
                    )
                  : Column(
                      children: [
                        for (var i = 0; i < cart.lines.length; i++) ...[
                          if (i > 0) const Divider(height: 20),
                          _CartLineRow(line: cart.lines[i]),
                        ],
                      ],
                    ),
            ),
            const SizedBox(height: 12),
            _SectionCard(
              title: 'Địa chỉ giao',
              child: address == null
                  ? Text(
                      'Chưa chọn địa chỉ',
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: theme.colorScheme.error,
                      ),
                    )
                  : Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          address.label,
                          style: theme.textTheme.bodyLarge,
                        ),
                        const SizedBox(height: 6),
                        Text(
                          'Lat: ${address.lat.toStringAsFixed(6)}  ·  '
                          'Lng: ${address.lng.toStringAsFixed(6)}',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                        if (quote != null) ...[
                          const SizedBox(height: 8),
                          Text(
                            quote.inRange
                                ? 'Trong phạm vi giao · khoảng '
                                    '${_fmtKm(quote.distanceKm)} km '
                                    '(tối đa ${_fmtKm(quote.maxRadiusKm)} km).'
                                : 'Ngoài phạm vi · khoảng '
                                    '${_fmtKm(quote.distanceKm)} km '
                                    '(tối đa ${_fmtKm(quote.maxRadiusKm)} km).',
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: quote.inRange
                                  ? theme.colorScheme.primary
                                  : theme.colorScheme.error,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ] else if (geo != null) ...[
                          const SizedBox(height: 8),
                          Text(
                            geo.inRange
                                ? geo.inRangeHint
                                : geo.outOfRangeMessage,
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: geo.inRange
                                  ? theme.colorScheme.primary
                                  : theme.colorScheme.error,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ],
                      ],
                    ),
            ),
            const SizedBox(height: 12),
            _SectionCard(
              title: 'Người nhận',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  TextField(
                    controller: _nameController,
                    focusNode: _nameFocus,
                    textInputAction: TextInputAction.done,
                    enabled: !_submitting,
                    decoration: InputDecoration(
                      labelText: 'Tên người nhận',
                      hintText: 'VD: Nguyễn Văn A',
                      helperText: prefillHint,
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                    ),
                    onSubmitted: (_) {
                      if (_canPlace) _placeOrder();
                    },
                  ),
                  if (session != null &&
                      session.user.phoneMasked.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Text(
                      'SĐT: ${session.user.phoneMasked}',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(height: 12),
            _SectionCard(
              title: 'Thanh toán',
              child: _quoteLoading && quote == null
                  ? const Padding(
                      padding: EdgeInsets.symmetric(vertical: 12),
                      child: Center(
                        child: SizedBox(
                          width: 28,
                          height: 28,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                      ),
                    )
                  : Column(
                      children: [
                        if (quote != null) ...[
                          _MoneyRow(
                            label: 'Khoảng cách',
                            value: '${_fmtKm(quote.distanceKm)} km',
                          ),
                          const SizedBox(height: 8),
                        ],
                        _MoneyRow(
                          label: 'Tạm tính',
                          value: formatVnd(subtotal),
                        ),
                        const SizedBox(height: 8),
                        _MoneyRow(
                          label: 'Phí giao hàng',
                          value: deliveryFee != null
                              ? formatVnd(deliveryFee)
                              : '—',
                          hint: deliveryFee == null
                              ? 'Chưa lấy được báo giá'
                              : deliveryFee == 0
                                  ? 'Miễn phí / chưa áp bậc phí'
                                  : null,
                        ),
                        const Divider(height: 24),
                        _MoneyRow(
                          label: 'Tổng cộng',
                          value: quote != null ? formatVnd(total) : '—',
                          emphasize: true,
                        ),
                      ],
                    ),
            ),
            if (_error != null) ...[
              const SizedBox(height: 16),
              Material(
                color: theme.colorScheme.errorContainer,
                borderRadius: BorderRadius.circular(12),
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Icon(
                        Icons.error_outline,
                        color: theme.colorScheme.onErrorContainer,
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(
                          _error!,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: theme.colorScheme.onErrorContainer,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
            const SizedBox(height: 20),
            FilledButton(
              onPressed: _canPlace ? _placeOrder : null,
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
                textStyle: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              child: _submitting
                  ? const SizedBox(
                      width: 22,
                      height: 22,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(
                      quote != null
                          ? 'Đặt đơn · ${formatVnd(total)}'
                          : 'Đặt đơn',
                    ),
            ),
          ],
        ),
      ),
    );
  }

  static String _fmtKm(double km) {
    if (km == km.roundToDouble()) return km.toStringAsFixed(0);
    return km.toStringAsFixed(2);
  }
}

class _SectionCard extends StatelessWidget {
  const _SectionCard({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              title,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 12),
            child,
          ],
        ),
      ),
    );
  }
}

class _CartLineRow extends StatelessWidget {
  const _CartLineRow({required this.line});

  final CartLine line;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                line.product.name,
                style: theme.textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                '${formatVnd(line.product.salePrice)} × ${line.quantity}',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
        Text(
          formatVnd(line.lineTotal),
          style: theme.textTheme.titleSmall?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ],
    );
  }
}

class _MoneyRow extends StatelessWidget {
  const _MoneyRow({
    required this.label,
    required this.value,
    this.hint,
    this.emphasize = false,
  });

  final String label;
  final String value;
  final String? hint;
  final bool emphasize;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                label,
                style: emphasize
                    ? theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      )
                    : theme.textTheme.bodyLarge,
              ),
            ),
            Text(
              value,
              style: emphasize
                  ? theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                      color: theme.colorScheme.primary,
                    )
                  : theme.textTheme.bodyLarge?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
            ),
          ],
        ),
        if (hint != null) ...[
          const SizedBox(height: 2),
          Text(
            hint!,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ],
    );
  }
}
