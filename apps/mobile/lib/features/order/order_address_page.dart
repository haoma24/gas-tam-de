import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import '../auth/auth_session.dart';
import 'customer_order_prefill.dart';
import 'geo_api.dart';
import 'geo_models.dart';
import 'location_permission.dart';
import 'order_address_selection.dart';
import 'saved_addresses.dart';

class OrderAddressPage extends ConsumerStatefulWidget {
  const OrderAddressPage({
    super.key,
    required this.onBack,
    required this.onContinue,
  });

  final VoidCallback onBack;
  final VoidCallback onContinue;

  @override
  ConsumerState<OrderAddressPage> createState() => _OrderAddressPageState();
}

class _OrderAddressPageState extends ConsumerState<OrderAddressPage> {
  final _searchController = TextEditingController();
  final _searchFocus = FocusNode();
  Timer? _debounce;
  List<GeoPlace> _suggestions = const [];
  List<SavedAddress> _saved = const [];
  SelectedAddress? _selected;
  GeoCheckResult? _check;
  bool _searchLoading = false;
  bool _checkLoading = false;
  bool _gpsLoading = false;
  String? _searchError;
  String? _checkError;
  int _checkSeq = 0;

  String get _userId => ref.read(authSessionProvider)?.user.id ?? 'guest';

  @override
  void initState() {
    super.initState();
    _selected = ref.read(orderAddressProvider);
    _check = ref.read(orderGeoCheckProvider);
    _searchController.addListener(_onSearchChanged);
    WidgetsBinding.instance.addPostFrameCallback((_) => _restoreAddresses());
  }

  Future<void> _restoreAddresses() async {
    final store = ref.read(savedAddressStoreProvider);
    var saved = store.load(_userId);

    if (saved.isEmpty) {
      try {
        final prefill = await ref.read(customerOrderPrefillProvider.future);
        if (prefill.hasLastAddress) {
          final last = prefill.lastAddress!;
          saved = [
            SavedAddress(
              id: 'last-order',
              name: 'Địa chỉ gần đây',
              label: last.addressText!,
              lat: last.lat!,
              lng: last.lng!,
              isDefault: true,
            ),
          ];
          await store.save(_userId, saved);
        }
      } catch (_) {}
    }

    if (!mounted) return;
    setState(() => _saved = saved);
    final current = _selected;
    if (current != null) {
      _setSearchText(current.label);
      await _runGeoCheck(current);
      return;
    }
    final defaults = saved.where((item) => item.isDefault);
    if (defaults.isNotEmpty) {
      _selectAddress(defaults.first.selection);
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _searchController.removeListener(_onSearchChanged);
    _searchController.dispose();
    _searchFocus.dispose();
    super.dispose();
  }

  bool get _canContinue =>
      _selected != null &&
      !_checkLoading &&
      _check?.inRange == true &&
      _checkError == null;

  void _setSearchText(String value) {
    _searchController.removeListener(_onSearchChanged);
    _searchController.text = value;
    _searchController.addListener(_onSearchChanged);
  }

  void _onSearchChanged() {
    _debounce?.cancel();
    final query = _searchController.text.trim();
    if (query.length < 3) {
      setState(() {
        _suggestions = const [];
        _searchLoading = false;
        _searchError = null;
      });
      return;
    }
    setState(() {
      _searchLoading = true;
      _searchError = null;
    });
    _debounce = Timer(const Duration(milliseconds: 550), () => _search(query));
  }

  Future<void> _search(String query) async {
    try {
      final results = await ref.read(geoApiProvider).search(query, limit: 6);
      if (!mounted || _searchController.text.trim() != query) return;
      setState(() {
        _suggestions = results;
        _searchLoading = false;
        _searchError = results.isEmpty
            ? 'Không tìm thấy địa chỉ phù hợp. Hãy nhập thêm số nhà, tên đường hoặc phường.'
            : null;
      });
    } on GeoApiException catch (error) {
      if (!mounted || _searchController.text.trim() != query) return;
      setState(() {
        _suggestions = const [];
        _searchLoading = false;
        _searchError = error.displayMessage;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _suggestions = const [];
        _searchLoading = false;
        _searchError = 'Không tìm được địa chỉ. Vui lòng thử lại.';
      });
    }
  }

  void _selectPlace(GeoPlace place) {
    final address =
        SelectedAddress(lat: place.lat, lng: place.lng, label: place.label);
    _setSearchText(address.label);
    _selectAddress(address);
  }

  void _selectAddress(SelectedAddress address) {
    ref.read(orderAddressProvider.notifier).select(address);
    ref.read(orderGeoCheckProvider.notifier).clear();
    setState(() {
      _selected = address;
      _suggestions = const [];
      _searchError = null;
      _checkError = null;
      _check = null;
    });
    _setSearchText(address.label);
    _searchFocus.unfocus();
    _runGeoCheck(address);
  }

  Future<void> _runGeoCheck(SelectedAddress address) async {
    final seq = ++_checkSeq;
    setState(() {
      _checkLoading = true;
      _checkError = null;
    });
    try {
      final result = await ref
          .read(geoApiProvider)
          .check(lat: address.lat, lng: address.lng);
      if (!mounted || seq != _checkSeq) return;
      ref.read(orderGeoCheckProvider.notifier).set(result);
      setState(() {
        _check = result;
        _checkLoading = false;
        _checkError = result.inRange ? null : result.outOfRangeMessage;
      });
    } on GeoApiException catch (error) {
      if (!mounted || seq != _checkSeq) return;
      setState(() {
        _check = null;
        _checkLoading = false;
        _checkError = error.displayMessage;
      });
    } catch (_) {
      if (!mounted || seq != _checkSeq) return;
      setState(() {
        _check = null;
        _checkLoading = false;
        _checkError = 'Không kiểm tra được phạm vi giao. Vui lòng thử lại.';
      });
    }
  }

  Future<void> _useCurrentLocation() async {
    setState(() {
      _gpsLoading = true;
      _searchError = null;
    });
    final result = await requestLocationAndGetPosition();
    if (!mounted) return;
    setState(() => _gpsLoading = false);
    if (!result.isOk) {
      setState(() => _searchError = result.errorMessage);
      return;
    }
    final position = result.position!;
    _selectAddress(SelectedAddress(
      lat: position.latitude,
      lng: position.longitude,
      label:
          'Vị trí hiện tại (${position.latitude.toStringAsFixed(5)}, ${position.longitude.toStringAsFixed(5)})',
    ));
  }

  Future<void> _saveSelected() async {
    final selected = _selected;
    if (selected == null || _check?.inRange != true) return;
    final controller = TextEditingController(
      text: _saved.isEmpty ? 'Nhà' : 'Địa chỉ ${_saved.length + 1}',
    );
    final name = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Lưu địa chỉ'),
        content: TextField(
          controller: controller,
          autofocus: true,
          maxLength: 30,
          decoration: const InputDecoration(
            labelText: 'Tên gợi nhớ',
            hintText: 'Nhà, Công ty…',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Hủy'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, controller.text.trim()),
            child: const Text('Lưu'),
          ),
        ],
      ),
    );
    controller.dispose();
    if (name == null || name.isEmpty || !mounted) return;

    final duplicate = _saved.indexWhere(
      (item) =>
          (item.lat - selected.lat).abs() < .000001 &&
          (item.lng - selected.lng).abs() < .000001,
    );
    final next = [..._saved];
    final item = SavedAddress(
      id: duplicate >= 0
          ? next[duplicate].id
          : DateTime.now().microsecondsSinceEpoch.toString(),
      name: name,
      label: selected.label,
      lat: selected.lat,
      lng: selected.lng,
      isDefault: next.isEmpty || (duplicate >= 0 && next[duplicate].isDefault),
    );
    if (duplicate >= 0) {
      next[duplicate] = item;
    } else {
      next.add(item);
    }
    await ref.read(savedAddressStoreProvider).save(_userId, next);
    if (mounted) setState(() => _saved = next);
  }

  Future<void> _setDefault(SavedAddress selected) async {
    final next = _saved
        .map((item) => item.copyWith(isDefault: item.id == selected.id))
        .toList(growable: false);
    await ref.read(savedAddressStoreProvider).save(_userId, next);
    if (mounted) setState(() => _saved = next);
  }

  Future<void> _removeSaved(SavedAddress selected) async {
    var next = _saved.where((item) => item.id != selected.id).toList();
    if (selected.isDefault && next.isNotEmpty) {
      next = [next.first.copyWith(isDefault: true), ...next.skip(1)];
    }
    await ref.read(savedAddressStoreProvider).save(_userId, next);
    if (mounted) setState(() => _saved = next);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.surface0,
      appBar: AppBar(
        backgroundColor: AppColors.surface0,
        title: const Text('Địa chỉ giao hàng'),
        leading: IconButton(
          onPressed: widget.onBack,
          icon: const Icon(Icons.arrow_back_rounded),
        ),
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 8, 20, 130),
          children: [
            Text(
              'Giao đến đâu?',
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w900,
                    letterSpacing: -.5,
                  ),
            ),
            const SizedBox(height: 6),
            Text(
              'Tìm và lưu nhiều địa chỉ. Lần sau app sẽ tự chọn địa chỉ mặc định.',
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: const Color(0xFF78716C),
                    height: 1.45,
                  ),
            ),
            const SizedBox(height: 20),
            Material(
              color: Colors.white,
              elevation: 5,
              shadowColor: AppColors.obsidian.withValues(alpha: .08),
              borderRadius: AppRadius.md,
              child: TextField(
                controller: _searchController,
                focusNode: _searchFocus,
                textInputAction: TextInputAction.search,
                decoration: InputDecoration(
                  hintText: 'Số nhà, tên đường, phường…',
                  prefixIcon: const Icon(Icons.search_rounded),
                  suffixIcon: _searchLoading
                      ? const Padding(
                          padding: EdgeInsets.all(14),
                          child: SizedBox.square(
                            dimension: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          ),
                        )
                      : _searchController.text.isEmpty
                          ? null
                          : IconButton(
                              onPressed: _searchController.clear,
                              icon: const Icon(Icons.close_rounded),
                            ),
                  border: InputBorder.none,
                  contentPadding: const EdgeInsets.symmetric(vertical: 17),
                ),
              ),
            ),
            if (_suggestions.isNotEmpty) ...[
              const SizedBox(height: 8),
              Material(
                color: Colors.white,
                borderRadius: AppRadius.md,
                clipBehavior: Clip.antiAlias,
                child: Column(
                  children: [
                    for (var index = 0;
                        index < _suggestions.length;
                        index++) ...[
                      if (index > 0) const Divider(height: 1, indent: 56),
                      ListTile(
                        leading: const CircleAvatar(
                          backgroundColor: Color(0xFFFFF1E7),
                          child: Icon(Icons.location_on_outlined,
                              color: AppColors.fire),
                        ),
                        title: Text(
                          _suggestions[index].label,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        subtitle: Text(_suggestions[index].source),
                        onTap: () => _selectPlace(_suggestions[index]),
                      ),
                    ],
                  ],
                ),
              ),
            ],
            if (_searchError != null) ...[
              const SizedBox(height: 10),
              _MessageCard(
                icon: Icons.cloud_off_outlined,
                text: _searchError!,
                error: true,
                action: TextButton(
                  onPressed: () => _search(_searchController.text.trim()),
                  child: const Text('Thử lại'),
                ),
              ),
            ],
            const SizedBox(height: 10),
            OutlinedButton.icon(
              onPressed: _gpsLoading ? null : _useCurrentLocation,
              icon: _gpsLoading
                  ? const SizedBox.square(
                      dimension: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.my_location_rounded),
              label: const Text('Dùng vị trí hiện tại'),
            ),
            if (_saved.isNotEmpty) ...[
              const SizedBox(height: 28),
              _SectionTitle(
                title: 'Địa chỉ đã lưu',
                trailing: '${_saved.length} địa chỉ',
              ),
              const SizedBox(height: 10),
              for (final address in _saved) ...[
                _SavedAddressCard(
                  address: address,
                  selected: _selected?.lat == address.lat &&
                      _selected?.lng == address.lng,
                  onSelect: () => _selectAddress(address.selection),
                  onDefault: () => _setDefault(address),
                  onDelete: () => _removeSaved(address),
                ),
                const SizedBox(height: 10),
              ],
            ],
            if (_selected != null) ...[
              const SizedBox(height: 20),
              const _SectionTitle(title: 'Địa chỉ đang chọn'),
              const SizedBox(height: 10),
              _MessageCard(
                icon: _check?.inRange == true
                    ? Icons.check_circle_rounded
                    : Icons.location_on_rounded,
                text: _selected!.label,
                detail: _checkLoading
                    ? 'Đang kiểm tra phạm vi giao…'
                    : _check?.inRange == true
                        ? _check!.inRangeHint
                        : _checkError,
                error: _checkError != null,
                action: _check?.inRange == true
                    ? TextButton.icon(
                        onPressed: _saveSelected,
                        icon: const Icon(Icons.bookmark_add_outlined),
                        label: const Text('Lưu'),
                      )
                    : null,
              ),
            ],
          ],
        ),
      ),
      bottomNavigationBar: Material(
        color: AppColors.surface0,
        elevation: 16,
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
            child: FilledButton(
              onPressed: _canContinue ? widget.onContinue : null,
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(54),
                backgroundColor: AppColors.fire,
                foregroundColor: Colors.white,
              ),
              child: Text(
                  _checkLoading ? 'Đang kiểm tra…' : 'Giao đến địa chỉ này'),
            ),
          ),
        ),
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle({required this.title, this.trailing});
  final String title;
  final String? trailing;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Text(title,
              style: Theme.of(context)
                  .textTheme
                  .titleMedium
                  ?.copyWith(fontWeight: FontWeight.w800)),
        ),
        if (trailing != null)
          Text(trailing!,
              style: const TextStyle(color: Color(0xFF78716C), fontSize: 12)),
      ],
    );
  }
}

class _SavedAddressCard extends StatelessWidget {
  const _SavedAddressCard({
    required this.address,
    required this.selected,
    required this.onSelect,
    required this.onDefault,
    required this.onDelete,
  });

  final SavedAddress address;
  final bool selected;
  final VoidCallback onSelect;
  final VoidCallback onDefault;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: selected ? const Color(0xFFFFF4EC) : Colors.white,
      borderRadius: AppRadius.md,
      child: InkWell(
        borderRadius: AppRadius.md,
        onTap: onSelect,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(14, 14, 6, 14),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(
                address.name.toLowerCase().contains('công')
                    ? Icons.business_rounded
                    : Icons.home_rounded,
                color: selected ? AppColors.fire : const Color(0xFF78716C),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Flexible(
                          child: Text(address.name,
                              style:
                                  const TextStyle(fontWeight: FontWeight.w800)),
                        ),
                        if (address.isDefault) ...[
                          const SizedBox(width: 6),
                          const Text('Mặc định',
                              style: TextStyle(
                                color: AppColors.fire,
                                fontSize: 11,
                                fontWeight: FontWeight.w700,
                              )),
                        ],
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(address.label,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                            color: Color(0xFF57534E), height: 1.4)),
                  ],
                ),
              ),
              PopupMenuButton<String>(
                onSelected: (value) =>
                    value == 'default' ? onDefault() : onDelete(),
                itemBuilder: (_) => [
                  if (!address.isDefault)
                    const PopupMenuItem(
                        value: 'default', child: Text('Đặt làm mặc định')),
                  const PopupMenuItem(
                      value: 'delete', child: Text('Xóa địa chỉ')),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MessageCard extends StatelessWidget {
  const _MessageCard({
    required this.icon,
    required this.text,
    this.detail,
    this.error = false,
    this.action,
  });

  final IconData icon;
  final String text;
  final String? detail;
  final bool error;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final color = error ? Theme.of(context).colorScheme.error : AppColors.fire;
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: error ? const Color(0xFFFFECEB) : Colors.white,
        borderRadius: AppRadius.md,
        border: Border.all(color: color.withValues(alpha: .16)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: color),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(text, style: const TextStyle(fontWeight: FontWeight.w700)),
                if (detail != null) ...[
                  const SizedBox(height: 5),
                  Text(detail!,
                      style:
                          TextStyle(color: color, fontSize: 12.5, height: 1.4)),
                ],
              ],
            ),
          ),
          if (action != null) action!,
        ],
      ),
    );
  }
}
