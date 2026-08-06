import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../catalog/catalog_models.dart';
import 'desk_settings_api.dart';
import 'desk_settings_models.dart';
import 'navigation_link.dart';
import 'new_order_voice.dart';
import 'order_api.dart';
import 'order_models.dart';
import 'wait_time_badge.dart';

/// Poll interval for Order Desk new-order detection (MVP).
///
/// Prefer polling over SSE/NATS bridge: works the same on Flutter Web +
/// Android/iOS without gateway WebSocket/SSE plumbing. A future NATS→SSE
/// bridge can replace this without changing the list API.
const Duration kAdminOrdersPollInterval = Duration(seconds: 10);

/// Admin Order Desk — FIFO list from `GET /v1/admin/orders` (oldest first).
///
/// Auto-refreshes on [kAdminOrdersPollInterval]; pull-to-refresh / app-bar
/// refresh still available. Pauses while the app is backgrounded.
class AdminOrdersPage extends ConsumerStatefulWidget {
  const AdminOrdersPage({
    super.key,
    required this.onBack,
    required this.onOpenOrder,
  });

  final VoidCallback onBack;
  final void Function(AdminOrder order) onOpenOrder;

  @override
  ConsumerState<AdminOrdersPage> createState() => _AdminOrdersPageState();
}

class _AdminOrdersPageState extends ConsumerState<AdminOrdersPage>
    with WidgetsBindingObserver {
  List<AdminOrder>? _items;
  bool _loading = true;
  String? _error;
  Timer? _pollTimer;
  Timer? _alertTimer;
  Timer? _tickTimer;
  bool _fetchInFlight = false;
  bool _hasSynced = false;
  Set<String> _knownIds = {};
  DeskSettings _desk = DeskSettings.defaults;
  DateTime _now = DateTime.now();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _loadDeskSettings();
    _load();
    _startPolling();
    _tickTimer = Timer.periodic(const Duration(seconds: 30), (_) {
      if (mounted) setState(() => _now = DateTime.now());
    });
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _alertTimer?.cancel();
    _tickTimer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  Future<void> _loadDeskSettings() async {
    try {
      final s = await ref.read(deskSettingsApiProvider).get();
      if (!mounted) return;
      setState(() => _desk = s);
      _restartAlertTimer();
    } catch (_) {}
  }

  void _restartAlertTimer() {
    _alertTimer?.cancel();
    _alertTimer = null;
    if (!_desk.alertEnabled) return;
    final every = Duration(seconds: _desk.alertIntervalSec);
    _alertTimer = Timer.periodic(every, (_) {
      final n = _items?.length ?? 0;
      if (n > 0) NewOrderVoice.announcePending(n);
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _load(silent: true);
      _startPolling();
    } else if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.detached) {
      _pollTimer?.cancel();
      _pollTimer = null;
    }
  }

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(kAdminOrdersPollInterval, (_) {
      _load(silent: true);
    });
  }

  /// [silent]: background poll — no full-page spinner; keep prior list on error.
  Future<void> _load({bool silent = false}) async {
    if (_fetchInFlight) return;
    _fetchInFlight = true;
    if (!silent) {
      setState(() {
        _loading = true;
        _error = null;
      });
    }
    try {
      final items = await ref.read(orderApiProvider).listAdminOrders();
      if (!mounted) return;
      final nextIds = items.map((o) => o.id).toSet();
      // After an empty first sync, `_knownIds` is still empty — use `_hasSynced`
      // so the first new orders still trigger a SnackBar.
      final newCount =
          !_hasSynced ? 0 : nextIds.difference(_knownIds).length;
      setState(() {
        _items = items;
        _loading = false;
        _error = null;
        _knownIds = nextIds;
        _hasSynced = true;
      });
      if (newCount > 0) {
        _notifyNewOrders(newCount);
      }
    } on OrderApiException catch (e) {
      if (!mounted) return;
      if (silent && _items != null) {
        // Keep showing cached list; avoid noisy snackbars on transient poll errors.
      } else {
        setState(() {
          _error = e.displayMessage;
          _loading = false;
        });
      }
    } catch (_) {
      if (!mounted) return;
      if (silent && _items != null) {
        // Keep cached list.
      } else {
        setState(() {
          _error = 'Có lỗi xảy ra. Thử lại.';
          _loading = false;
        });
      }
    } finally {
      _fetchInFlight = false;
    }
  }

  void _notifyNewOrders(int count) {
    final messenger = ScaffoldMessenger.maybeOf(context);
    if (messenger != null) {
      messenger
        ..hideCurrentSnackBar()
        ..showSnackBar(
          SnackBar(
            content: Text(
              count == 1 ? 'Có 1 đơn mới' : 'Có $count đơn mới',
            ),
            behavior: SnackBarBehavior.floating,
            duration: const Duration(seconds: 3),
          ),
        );
    }
    if (_desk.alertEnabled) {
      final pending = _items?.length ?? count;
      NewOrderVoice.announcePending(pending);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Order Desk'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: widget.onBack,
        ),
        actions: [
          IconButton(
            tooltip: 'Tải lại',
            icon: const Icon(Icons.refresh),
            onPressed: _loading ? null : () => _load(),
          ),
        ],
      ),
      body: SafeArea(child: _buildBody(theme)),
    );
  }

  Widget _buildBody(ThemeData theme) {
    if (_loading && _items == null) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null && _items == null) {
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
                onPressed: () => _load(),
                child: const Text('Thử lại'),
              ),
            ],
          ),
        ),
      );
    }

    final items = _items ?? const <AdminOrder>[];
    if (items.isEmpty) {
      return RefreshIndicator(
        onRefresh: () => _load(),
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.all(24),
          children: [
            const SizedBox(height: 80),
            Text(
              'Không có đơn chờ giao',
              textAlign: TextAlign.center,
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              'Đơn mới sẽ tự hiện (làm mới mỗi '
              '${kAdminOrdersPollInterval.inSeconds}s) theo thứ tự cũ nhất trước.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            Center(
              child: OutlinedButton.icon(
                onPressed: () => _load(),
                icon: const Icon(Icons.refresh),
                label: const Text('Tải lại'),
              ),
            ),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () => _load(),
      child: ListView.separated(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        itemCount: items.length,
        separatorBuilder: (_, __) => const SizedBox(height: 8),
        itemBuilder: (context, index) {
          final order = items[index];
          return _OrderDeskTile(
            order: order,
            settings: _desk,
            now: _now,
            onTap: () => widget.onOpenOrder(order),
          );
        },
      ),
    );
  }
}

class _OrderDeskTile extends StatelessWidget {
  const _OrderDeskTile({
    required this.order,
    required this.settings,
    required this.now,
    required this.onTap,
  });

  final AdminOrder order;
  final DeskSettings settings;
  final DateTime now;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final muted = theme.colorScheme.onSurfaceVariant;
    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _SttBadge(stt: order.stt),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            order.customerName.isEmpty
                                ? '—'
                                : order.customerName,
                            style: theme.textTheme.titleMedium?.copyWith(
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                        WaitTimeBadge(
                          createdAt: order.createdAt,
                          settings: settings,
                          now: now,
                        ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(
                      order.phoneMasked.isEmpty ? '—' : order.phoneMasked,
                      style: theme.textTheme.bodyMedium?.copyWith(color: muted),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      order.addressText.isEmpty ? '—' : order.addressText,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.bodyMedium,
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        Icon(Icons.straighten, size: 16, color: muted),
                        const SizedBox(width: 4),
                        Text(
                          _fmtKm(order.distanceKm),
                          style: theme.textTheme.labelLarge?.copyWith(
                            color: muted,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(width: 16),
                        Icon(Icons.schedule, size: 16, color: muted),
                        const SizedBox(width: 4),
                        Expanded(
                          child: Text(
                            formatOrderTime(order.createdAt),
                            style: theme.textTheme.labelLarge?.copyWith(
                              color: muted,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              Icon(
                Icons.chevron_right,
                color: muted,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SttBadge extends StatelessWidget {
  const _SttBadge({required this.stt});

  final int stt;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: 40,
      height: 40,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(
        '$stt',
        style: theme.textTheme.titleMedium?.copyWith(
          fontWeight: FontWeight.w800,
          color: theme.colorScheme.onPrimaryContainer,
        ),
      ),
    );
  }
}

/// Read-only order detail from list payload (no `GET /admin/orders/{id}` yet).
///
/// «Dẫn đường» (T5.2.3) opens Maps to [AdminOrder.lat]/[AdminOrder.lng]
/// via [openNavigationTo]. «Hoàn tất» (T6.1.4) opens payment dialog →
/// `POST /v1/admin/orders/{id}/complete`.
class AdminOrderDetailPage extends ConsumerWidget {
  const AdminOrderDetailPage({
    super.key,
    required this.order,
    required this.onBack,
    required this.onCompleted,
  });

  final AdminOrder order;
  final VoidCallback onBack;

  /// Called after successful complete so the desk list can reload (PENDING gone).
  final VoidCallback onCompleted;

  /// Missing / unset delivery pin — model defaults null API coords to `0`.
  static bool hasDeliveryCoords(double lat, double lng) {
    if (!lat.isFinite || !lng.isFinite) return false;
    return !(lat == 0 && lng == 0);
  }

  Future<void> _onOpenDirections(BuildContext context) async {
    final messenger = ScaffoldMessenger.maybeOf(context);
    if (!hasDeliveryCoords(order.lat, order.lng)) {
      messenger
        ?..hideCurrentSnackBar()
        ..showSnackBar(
          const SnackBar(
            content: Text('Đơn không có toạ độ điểm giao.'),
            behavior: SnackBarBehavior.floating,
          ),
        );
      return;
    }

    final result = await openNavigationTo(order.lat, order.lng);
    if (!context.mounted) return;
    if (!result.isOk) {
      ScaffoldMessenger.maybeOf(context)
        ?..hideCurrentSnackBar()
        ..showSnackBar(
          SnackBar(
            content: Text(result.errorMessage!),
            behavior: SnackBarBehavior.floating,
          ),
        );
    }
  }

  Future<void> _onComplete(BuildContext context, WidgetRef ref) async {
    final result = await showDialog<CompletedOrder>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => _CompleteOrderDialog(
        order: order,
        api: ref.read(orderApiProvider),
      ),
    );
    if (result == null || !context.mounted) return;

    ScaffoldMessenger.maybeOf(context)
      ?..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Text(_completeSuccessMessage(result)),
          behavior: SnackBarBehavior.floating,
        ),
      );
    onCompleted();
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final muted = theme.colorScheme.onSurfaceVariant;
    final shortId = order.id.length > 8 ? order.id.substring(0, 8) : order.id;
    final canComplete = order.status.toUpperCase() == 'PENDING';

    return Scaffold(
      appBar: AppBar(
        title: Text('Đơn #$shortId'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: onBack,
        ),
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(24, 8, 24, 32),
          children: [
            Row(
              children: [
                _SttBadge(stt: order.stt),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    'Thứ tự giao (FIFO)',
                    style: theme.textTheme.bodyMedium?.copyWith(color: muted),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            _DetailField(label: 'Khách hàng', value: order.customerName),
            _DetailField(label: 'SĐT', value: order.phoneMasked),
            _DetailField(label: 'Địa chỉ', value: order.addressText),
            _DetailField(label: 'Khoảng cách', value: _fmtKm(order.distanceKm)),
            _DetailField(
              label: 'Thời gian đặt',
              value: formatOrderTime(order.createdAt),
            ),
            const SizedBox(height: 8),
            FilledButton.icon(
              onPressed: () => _onOpenDirections(context),
              icon: const Icon(Icons.directions),
              label: const Text('Dẫn đường'),
            ),
            const SizedBox(height: 12),
            FilledButton.tonalIcon(
              onPressed: canComplete ? () => _onComplete(context, ref) : null,
              icon: const Icon(Icons.check_circle_outline),
              label: const Text('Hoàn tất'),
            ),
            const SizedBox(height: 16),
            Divider(color: theme.colorScheme.outlineVariant),
            const SizedBox(height: 8),
            Text(
              'Sản phẩm',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            if (order.items.isEmpty)
              Text(
                'Không có dòng hàng',
                style: theme.textTheme.bodyMedium?.copyWith(color: muted),
              )
            else
              ...order.items.map(
                (it) => Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Text(
                          '${it.productName} × ${it.qty}',
                          style: theme.textTheme.bodyLarge,
                        ),
                      ),
                      Text(
                        formatVnd(it.lineTotal),
                        style: theme.textTheme.bodyLarge?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            const SizedBox(height: 12),
            _MoneyRow(label: 'Tạm tính', value: formatVnd(order.subtotal)),
            _MoneyRow(label: 'Phí giao', value: formatVnd(order.deliveryFee)),
            const SizedBox(height: 4),
            _MoneyRow(
              label: 'Tổng',
              value: formatVnd(order.total),
              emphasize: true,
            ),
          ],
        ),
      ),
    );
  }
}

String _completeSuccessMessage(CompletedOrder result) {
  switch (result.paymentType) {
    case OrderPaymentType.partial:
      return 'Đã hoàn tất. Thu ${formatVnd(result.amountPaid)}, '
          'nợ ${formatVnd(result.debt)}.';
    case OrderPaymentType.unpaid:
      return 'Đã hoàn tất. Công nợ ${formatVnd(result.debt)}.';
    case OrderPaymentType.full:
    default:
      return 'Đã hoàn tất. Thu đủ ${formatVnd(result.amountPaid)}.';
  }
}

/// Dialog: chọn FULL / PARTIAL / UNPAID (+ amount_paid) rồi gọi complete API.
class _CompleteOrderDialog extends StatefulWidget {
  const _CompleteOrderDialog({
    required this.order,
    required this.api,
  });

  final AdminOrder order;
  final OrderApi api;

  @override
  State<_CompleteOrderDialog> createState() => _CompleteOrderDialogState();
}

class _CompleteOrderDialogState extends State<_CompleteOrderDialog> {
  String _paymentType = OrderPaymentType.full;
  final _amountCtrl = TextEditingController();
  String? _localError;
  bool _submitting = false;

  @override
  void dispose() {
    _amountCtrl.dispose();
    super.dispose();
  }

  int? _parseAmountPaid() {
    final raw = _amountCtrl.text.trim().replaceAll('.', '').replaceAll(',', '');
    if (raw.isEmpty) return null;
    return int.tryParse(raw);
  }

  int _previewDebt() {
    final total = widget.order.total;
    switch (_paymentType) {
      case OrderPaymentType.partial:
        final paid = _parseAmountPaid();
        if (paid == null) return total;
        if (paid <= 0 || paid >= total) return total;
        return total - paid;
      case OrderPaymentType.unpaid:
        return total;
      case OrderPaymentType.full:
      default:
        return 0;
    }
  }

  String? _validateLocal() {
    final total = widget.order.total;
    if (_paymentType == OrderPaymentType.partial) {
      final paid = _parseAmountPaid();
      if (paid == null) {
        return 'Nhập số tiền đã thu.';
      }
      if (paid <= 0 || paid >= total) {
        return 'Số tiền phải lớn hơn 0 và nhỏ hơn tổng '
            '(${formatVnd(total)}).';
      }
    }
    return null;
  }

  Future<void> _submit() async {
    final err = _validateLocal();
    if (err != null) {
      setState(() => _localError = err);
      return;
    }

    setState(() {
      _localError = null;
      _submitting = true;
    });

    try {
      final request = CompleteOrderRequest(
        paymentType: _paymentType,
        amountPaid: _paymentType == OrderPaymentType.partial
            ? _parseAmountPaid()
            : null,
      );
      final result = await widget.api.completeOrder(widget.order.id, request);
      if (!mounted) return;
      Navigator.of(context).pop(result);
    } on OrderApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _localError = e.displayMessage;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _localError = 'Có lỗi xảy ra. Thử lại.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final total = widget.order.total;
    final debt = _previewDebt();

    return AlertDialog(
      title: const Text('Hoàn tất giao hàng'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Tổng đơn: ${formatVnd(total)}',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 12),
            Text(
              'Trạng thái thu tiền',
              style: theme.textTheme.labelLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 4),
            AbsorbPointer(
              absorbing: _submitting,
              child: RadioGroup<String>(
                groupValue: _paymentType,
                onChanged: (v) {
                  if (v == null) return;
                  setState(() {
                    _paymentType = v;
                    _localError = null;
                  });
                },
                child: const Column(
                  children: [
                    RadioListTile<String>(
                      contentPadding: EdgeInsets.zero,
                      dense: true,
                      title: Text('Đã thu đủ'),
                      value: OrderPaymentType.full,
                    ),
                    RadioListTile<String>(
                      contentPadding: EdgeInsets.zero,
                      dense: true,
                      title: Text('Thu một phần'),
                      value: OrderPaymentType.partial,
                    ),
                    RadioListTile<String>(
                      contentPadding: EdgeInsets.zero,
                      dense: true,
                      title: Text('Chưa thu (nợ)'),
                      value: OrderPaymentType.unpaid,
                    ),
                  ],
                ),
              ),
            ),
            if (_paymentType == OrderPaymentType.partial) ...[
              const SizedBox(height: 8),
              TextField(
                controller: _amountCtrl,
                enabled: !_submitting,
                keyboardType: TextInputType.number,
                inputFormatters: [
                  FilteringTextInputFormatter.digitsOnly,
                ],
                decoration: const InputDecoration(
                  labelText: 'Số tiền đã thu (₫)',
                  border: OutlineInputBorder(),
                ),
                onChanged: (_) => setState(() => _localError = null),
              ),
            ],
            const SizedBox(height: 12),
            Text(
              'Công nợ dự kiến: ${formatVnd(debt)}',
              style: theme.textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            if (_localError != null) ...[
              const SizedBox(height: 10),
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
      actions: [
        TextButton(
          onPressed: _submitting ? null : () => Navigator.of(context).pop(),
          child: const Text('Hủy'),
        ),
        FilledButton(
          onPressed: _submitting ? null : _submit,
          child: _submitting
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('Xác nhận'),
        ),
      ],
    );
  }
}

class _DetailField extends StatelessWidget {
  const _DetailField({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: theme.textTheme.labelLarge?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 2),
          Text(
            value.isEmpty ? '—' : value,
            style: theme.textTheme.bodyLarge?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

class _MoneyRow extends StatelessWidget {
  const _MoneyRow({
    required this.label,
    required this.value,
    this.emphasize = false,
  });

  final String label;
  final String value;
  final bool emphasize;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final style = emphasize
        ? theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800)
        : theme.textTheme.bodyLarge;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          Expanded(child: Text(label, style: style)),
          Text(value, style: style),
        ],
      ),
    );
  }
}

String _fmtKm(double km) {
  if (km == km.roundToDouble()) return '${km.toStringAsFixed(0)} km';
  return '${km.toStringAsFixed(2)} km';
}

/// Formats API `created_at` (RFC3339) for desk display; falls back to raw.
String formatOrderTime(String raw) {
  final parsed = DateTime.tryParse(raw);
  if (parsed == null) return raw.isEmpty ? '—' : raw;
  final local = parsed.toLocal();
  String two(int n) => n.toString().padLeft(2, '0');
  return '${two(local.day)}/${two(local.month)}/${local.year} '
      '${two(local.hour)}:${two(local.minute)}';
}
