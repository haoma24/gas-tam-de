import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';

import 'geo_api.dart';
import 'geo_models.dart';
import 'location_permission.dart';
import 'order_address_selection.dart';

/// Default map center — Quận 1, TP.HCM (near default store seed).
const _kDefaultCenter = LatLng(10.7769, 106.7009);
const _kDefaultZoom = 14.0;
const _kSelectedZoom = 16.0;

/// Order flow step 2 — địa chỉ giao + kiểm tra bán kính (T3.1.3 / T3.2.3).
///
/// Calls geo-service `GET /v1/geo/search` and `POST /v1/geo/check` only —
/// never OSM/Photon from the client. Map: `flutter_map` + OSM raster tiles.
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
  final _mapController = MapController();

  Timer? _debounce;
  bool _gpsLoading = false;
  bool _searchLoading = false;
  bool _checkLoading = false;
  bool _mapReady = false;
  String? _error;
  List<GeoPlace> _suggestions = const [];
  SelectedAddress? _selected;
  GeoCheckResult? _check;
  int _checkSeq = 0;

  @override
  void initState() {
    super.initState();
    _selected = ref.read(orderAddressProvider);
    _check = ref.read(orderGeoCheckProvider);
    _searchController.addListener(_onSearchChanged);
    final sel = _selected;
    if (sel != null) {
      _searchController.text = sel.label;
      // Re-validate radius when returning to this step.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) _runGeoCheck(sel);
      });
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _searchController.removeListener(_onSearchChanged);
    _searchController.dispose();
    _searchFocus.dispose();
    _mapController.dispose();
    super.dispose();
  }

  bool get _canContinue =>
      _selected != null &&
      !_checkLoading &&
      _check != null &&
      _check!.inRange &&
      _error == null;

  void _onSearchChanged() {
    _debounce?.cancel();
    final q = _searchController.text.trim();
    if (q.length < 2) {
      setState(() {
        _suggestions = const [];
        _searchLoading = false;
      });
      return;
    }
    setState(() => _searchLoading = true);
    _debounce = Timer(const Duration(milliseconds: 400), () {
      _runSearch(q);
    });
  }

  Future<void> _runSearch(String q) async {
    try {
      final items = await ref.read(geoApiProvider).search(q);
      if (!mounted) return;
      // Ignore stale responses if the field changed.
      if (_searchController.text.trim() != q) return;
      setState(() {
        _suggestions = items;
        _searchLoading = false;
        _error = null;
      });
    } on GeoApiException catch (e) {
      if (!mounted) return;
      if (_searchController.text.trim() != q) return;
      setState(() {
        _suggestions = const [];
        _searchLoading = false;
        _error = e.displayMessage;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _suggestions = const [];
        _searchLoading = false;
        _error = 'Không tìm được địa chỉ. Thử lại.';
      });
    }
  }

  void _applySelection(SelectedAddress address, {bool moveMap = true}) {
    ref.read(orderAddressProvider.notifier).select(address);
    ref.read(orderGeoCheckProvider.notifier).clear();
    setState(() {
      _selected = address;
      _suggestions = const [];
      _error = null;
      _check = null;
    });
    _searchFocus.unfocus();
    if (moveMap && _mapReady) {
      _mapController.move(LatLng(address.lat, address.lng), _kSelectedZoom);
    }
    _runGeoCheck(address);
  }

  Future<void> _runGeoCheck(SelectedAddress address) async {
    final seq = ++_checkSeq;
    setState(() => _checkLoading = true);

    try {
      final result = await ref.read(geoApiProvider).check(
            lat: address.lat,
            lng: address.lng,
          );
      if (!mounted || seq != _checkSeq) return;

      ref.read(orderGeoCheckProvider.notifier).set(result);
      setState(() {
        _check = result;
        _checkLoading = false;
        _error = result.inRange ? null : result.outOfRangeMessage;
      });
    } on GeoApiException catch (e) {
      if (!mounted || seq != _checkSeq) return;
      ref.read(orderGeoCheckProvider.notifier).clear();
      setState(() {
        _check = null;
        _checkLoading = false;
        _error = e.displayMessage;
      });
    } catch (_) {
      if (!mounted || seq != _checkSeq) return;
      ref.read(orderGeoCheckProvider.notifier).clear();
      setState(() {
        _check = null;
        _checkLoading = false;
        _error = 'Không kiểm tra được phạm vi giao. Thử lại.';
      });
    }
  }

  void _onSuggestionTap(GeoPlace place) {
    _searchController.removeListener(_onSearchChanged);
    _searchController.text = place.label;
    _searchController.addListener(_onSearchChanged);
    _applySelection(
      SelectedAddress(lat: place.lat, lng: place.lng, label: place.label),
    );
  }

  void _onMapTap(TapPosition tapPosition, LatLng point) {
    final keepLabel = _selected?.label;
    final label = (keepLabel != null &&
            keepLabel.isNotEmpty &&
            keepLabel != 'Vị trí hiện tại')
        ? keepLabel
        : 'Vị trí đã chọn trên bản đồ';
    _applySelection(
      SelectedAddress(lat: point.latitude, lng: point.longitude, label: label),
      moveMap: false,
    );
  }

  Future<void> _useCurrentLocation() async {
    setState(() {
      _gpsLoading = true;
      _error = null;
    });

    final result = await requestLocationAndGetPosition();
    if (!mounted) return;

    setState(() => _gpsLoading = false);

    if (!result.isOk) {
      setState(() => _error = result.errorMessage);
      return;
    }

    final pos = result.position!;
    _searchController.removeListener(_onSearchChanged);
    _searchController.text = 'Vị trí hiện tại';
    _searchController.addListener(_onSearchChanged);
    _applySelection(
      SelectedAddress(
        lat: pos.latitude,
        lng: pos.longitude,
        label: 'Vị trí hiện tại',
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final center = _selected != null
        ? LatLng(_selected!.lat, _selected!.lng)
        : _kDefaultCenter;
    final outOfRange = _check != null && !_check!.inRange;
    final inRange = _check != null && _check!.inRange;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Địa chỉ giao'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: widget.onBack,
        ),
      ),
      body: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    'Chọn vị trí giao hàng',
                    style: theme.textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'Tìm địa chỉ hoặc chạm bản đồ để ghim vị trí.',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _searchController,
                    focusNode: _searchFocus,
                    textInputAction: TextInputAction.search,
                    decoration: InputDecoration(
                      hintText: 'Tìm địa chỉ (tối thiểu 2 ký tự)…',
                      prefixIcon: const Icon(Icons.search),
                      suffixIcon: _searchLoading
                          ? const Padding(
                              padding: EdgeInsets.all(12),
                              child: SizedBox(
                                width: 18,
                                height: 18,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
                              ),
                            )
                          : (_searchController.text.isNotEmpty
                              ? IconButton(
                                  tooltip: 'Xóa',
                                  icon: const Icon(Icons.clear),
                                  onPressed: () {
                                    _searchController.clear();
                                    setState(() {
                                      _suggestions = const [];
                                      if (!outOfRange) _error = null;
                                    });
                                  },
                                )
                              : null),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      isDense: true,
                    ),
                  ),
                  const SizedBox(height: 8),
                  FilledButton.tonalIcon(
                    onPressed: _gpsLoading ? null : _useCurrentLocation,
                    icon: _gpsLoading
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.my_location),
                    label: Text(
                      _gpsLoading
                          ? 'Đang lấy vị trí…'
                          : 'Dùng vị trí hiện tại',
                    ),
                  ),
                ],
              ),
            ),
            if (_suggestions.isNotEmpty)
              Material(
                elevation: 2,
                color: theme.colorScheme.surface,
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxHeight: 180),
                  child: ListView.separated(
                    shrinkWrap: true,
                    padding: EdgeInsets.zero,
                    itemCount: _suggestions.length,
                    separatorBuilder: (_, __) => const Divider(height: 1),
                    itemBuilder: (context, index) {
                      final place = _suggestions[index];
                      return ListTile(
                        dense: true,
                        leading: Icon(
                          Icons.place_outlined,
                          color: theme.colorScheme.primary,
                        ),
                        title: Text(
                          place.label,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        onTap: () => _onSuggestionTap(place),
                      );
                    },
                  ),
                ),
              ),
            if (_checkLoading)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                child: Row(
                  children: [
                    SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: theme.colorScheme.primary,
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(
                        'Đang kiểm tra phạm vi giao…',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            if (_error != null && !_checkLoading)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                child: Material(
                  color: theme.colorScheme.errorContainer,
                  borderRadius: BorderRadius.circular(12),
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(
                          outOfRange
                              ? Icons.wrong_location_outlined
                              : Icons.error_outline,
                          color: theme.colorScheme.onErrorContainer,
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Text(
                            _error!,
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: theme.colorScheme.onErrorContainer,
                              fontWeight: outOfRange
                                  ? FontWeight.w600
                                  : FontWeight.w400,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            if (inRange && !_checkLoading)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                child: Material(
                  color: theme.colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(12),
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(
                          Icons.check_circle_outline,
                          color: theme.colorScheme.onPrimaryContainer,
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Text(
                            _check!.inRangeHint,
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: theme.colorScheme.onPrimaryContainer,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            const SizedBox(height: 8),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(12),
                  child: FlutterMap(
                    mapController: _mapController,
                    options: MapOptions(
                      initialCenter: center,
                      initialZoom: _selected != null
                          ? _kSelectedZoom
                          : _kDefaultZoom,
                      onTap: _onMapTap,
                      onMapReady: () {
                        _mapReady = true;
                        final sel = _selected;
                        if (sel != null) {
                          _mapController.move(
                            LatLng(sel.lat, sel.lng),
                            _kSelectedZoom,
                          );
                        }
                      },
                      interactionOptions: const InteractionOptions(
                        flags: InteractiveFlag.all & ~InteractiveFlag.rotate,
                      ),
                    ),
                    children: [
                      TileLayer(
                        urlTemplate:
                            'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                        userAgentPackageName: 'vn.gastamde.gas_tam_de',
                      ),
                      if (_selected != null)
                        MarkerLayer(
                          markers: [
                            Marker(
                              point: LatLng(_selected!.lat, _selected!.lng),
                              width: 48,
                              height: 48,
                              alignment: Alignment.topCenter,
                              child: Icon(
                                Icons.location_on,
                                size: 48,
                                color: outOfRange
                                    ? theme.colorScheme.error
                                    : theme.colorScheme.primary,
                              ),
                            ),
                          ],
                        ),
                    ],
                  ),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (_selected != null)
                    Material(
                      color: theme.colorScheme.surfaceContainerLowest,
                      borderRadius: BorderRadius.circular(12),
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Đã chọn',
                              style: theme.textTheme.titleMedium?.copyWith(
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            const SizedBox(height: 6),
                            Text(
                              _selected!.label,
                              style: theme.textTheme.bodyLarge,
                            ),
                            const SizedBox(height: 6),
                            Text(
                              'Lat: ${_selected!.lat.toStringAsFixed(6)}  ·  '
                              'Lng: ${_selected!.lng.toStringAsFixed(6)}',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  if (_selected != null) const SizedBox(height: 12),
                  FilledButton(
                    onPressed: _canContinue ? widget.onContinue : null,
                    style: FilledButton.styleFrom(
                      minimumSize: const Size.fromHeight(52),
                      textStyle: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    child: Text(
                      _checkLoading
                          ? 'Đang kiểm tra…'
                          : (outOfRange
                              ? 'Ngoài phạm vi giao'
                              : 'Tiếp tục'),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
