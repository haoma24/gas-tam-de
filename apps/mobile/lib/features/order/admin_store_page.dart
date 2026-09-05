import 'dart:async';

import '../../core/ui/ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';

import 'geo_api.dart';
import 'geo_models.dart';
import 'location_permission.dart';

const _kDefaultCenter = LatLng(10.7769, 106.7009);
const _kDefaultZoom = 15.0;

/// Admin — cấu hình vị trí cửa hàng + bán kính giao (`GET/PUT` geo store).
class AdminStorePage extends ConsumerStatefulWidget {
  const AdminStorePage({
    super.key,
  });

  @override
  ConsumerState<AdminStorePage> createState() => _AdminStorePageState();
}

class _AdminStorePageState extends ConsumerState<AdminStorePage> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();
  final _addressController = TextEditingController();
  final _latController = TextEditingController();
  final _lngController = TextEditingController();
  final _radiusController = TextEditingController();
  final _searchController = TextEditingController();
  final _mapController = MapController();

  Timer? _debounce;
  bool _loading = true;
  bool _saving = false;
  bool _gpsLoading = false;
  bool _searchLoading = false;
  bool _mapReady = false;
  String? _error;
  String? _updatedAt;
  List<GeoPlace> _suggestions = const [];
  LatLng _pin = _kDefaultCenter;

  @override
  void initState() {
    super.initState();
    _searchController.addListener(_onSearchChanged);
    _load();
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _searchController.removeListener(_onSearchChanged);
    _nameController.dispose();
    _addressController.dispose();
    _latController.dispose();
    _lngController.dispose();
    _radiusController.dispose();
    _searchController.dispose();
    _mapController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final store = await ref.read(geoApiProvider).getStore();
      if (!mounted) return;
      _applyStore(store);
      setState(() => _loading = false);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && _mapReady) {
          _mapController.move(_pin, _kDefaultZoom);
        }
      });
    } on GeoApiException catch (e) {
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

  void _applyStore(StoreSettings store) {
    _nameController.text = store.name;
    _addressController.text = store.addressText ?? '';
    _latController.text = store.lat.toStringAsFixed(6);
    _lngController.text = store.lng.toStringAsFixed(6);
    _radiusController.text = _fmtNum(store.maxRadiusKm);
    _updatedAt = store.updatedAt;
    _pin = LatLng(store.lat, store.lng);
  }

  void _setPin(LatLng point, {String? addressLabel}) {
    setState(() {
      _pin = point;
      _latController.text = point.latitude.toStringAsFixed(6);
      _lngController.text = point.longitude.toStringAsFixed(6);
      if (addressLabel != null && addressLabel.trim().isNotEmpty) {
        _addressController.text = addressLabel.trim();
      }
    });
    if (_mapReady) {
      _mapController.move(point, _kDefaultZoom);
    }
  }

  void _onSearchChanged() {
    _debounce?.cancel();
    final q = _searchController.text.trim();
    if (q.length < 2) {
      setState(() => _suggestions = const []);
      return;
    }
    _debounce = Timer(const Duration(milliseconds: 350), () => _runSearch(q));
  }

  Future<void> _runSearch(String q) async {
    setState(() => _searchLoading = true);
    try {
      final items = await ref.read(geoApiProvider).search(q);
      if (!mounted) return;
      if (_searchController.text.trim() != q) return;
      setState(() {
        _suggestions = items;
        _searchLoading = false;
      });
    } on GeoApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _searchLoading = false;
        _error = e.displayMessage;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _searchLoading = false;
        _error = 'Không tìm được địa chỉ. Thử lại.';
      });
    }
  }

  Future<void> _useGps() async {
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
    final p = result.position!;
    _setPin(LatLng(p.latitude, p.longitude));
  }

  Future<void> _save() async {
    setState(() => _error = null);
    if (!(_formKey.currentState?.validate() ?? false)) return;

    final lat =
        double.tryParse(_latController.text.trim().replaceAll(',', '.'));
    final lng =
        double.tryParse(_lngController.text.trim().replaceAll(',', '.'));
    final radius =
        double.tryParse(_radiusController.text.trim().replaceAll(',', '.'));
    if (lat == null || lng == null || radius == null) {
      setState(() => _error = 'Tọa độ hoặc bán kính không hợp lệ.');
      return;
    }

    setState(() => _saving = true);
    try {
      final saved = await ref.read(geoApiProvider).putAdminStore(
            name: _nameController.text.trim(),
            lat: lat,
            lng: lng,
            maxRadiusKm: radius,
            addressText: _addressController.text.trim(),
          );
      if (!mounted) return;
      _applyStore(saved);
      setState(() => _saving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Đã lưu vị trí cửa hàng.')),
      );
    } on GeoApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _saving = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Có lỗi xảy ra. Thử lại.';
        _saving = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Vị trí cửa hàng'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: _saving ? null : () => popOrGo(context, '/admin/settings'),
        ),
        actions: [
          IconButton(
            tooltip: 'Tải lại',
            icon: const Icon(Icons.refresh),
            onPressed: (_loading || _saving) ? null : _load,
          ),
        ],
      ),
      body: SafeArea(
        child: _loading
            ? const AppLoading()
            : Form(
                key: _formKey,
                child: ListView(
                  padding: const EdgeInsets.fromLTRB(24, 16, 24, 32),
                  children: [
                    Text(
                      'Tọa độ gốc để tính khoảng cách giao hàng.',
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    if (_updatedAt != null) ...[
                      const SizedBox(height: 6),
                      Text(
                        'Cập nhật: $_updatedAt',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                    const SizedBox(height: 20),
                    TextFormField(
                      controller: _nameController,
                      enabled: !_saving,
                      textInputAction: TextInputAction.next,
                      decoration: const InputDecoration(
                        labelText: 'Tên cửa hàng',
                        border: OutlineInputBorder(),
                      ),
                      validator: (v) {
                        if ((v ?? '').trim().isEmpty) {
                          return 'Nhập tên cửa hàng.';
                        }
                        return null;
                      },
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _addressController,
                      enabled: !_saving,
                      maxLines: 2,
                      decoration: const InputDecoration(
                        labelText: 'Địa chỉ (hiển thị)',
                        border: OutlineInputBorder(),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Expanded(
                          child: TextFormField(
                            controller: _latController,
                            enabled: !_saving,
                            keyboardType: const TextInputType.numberWithOptions(
                              decimal: true,
                              signed: true,
                            ),
                            inputFormatters: [
                              FilteringTextInputFormatter.allow(
                                RegExp(r'[0-9\.\-]'),
                              ),
                            ],
                            decoration: const InputDecoration(
                              labelText: 'Vĩ độ (lat)',
                              border: OutlineInputBorder(),
                            ),
                            onChanged: (_) => _syncPinFromFields(),
                            validator: (v) {
                              final n = double.tryParse(
                                (v ?? '').trim().replaceAll(',', '.'),
                              );
                              if (n == null || n < -90 || n > 90) {
                                return 'Lat -90…90';
                              }
                              return null;
                            },
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: TextFormField(
                            controller: _lngController,
                            enabled: !_saving,
                            keyboardType: const TextInputType.numberWithOptions(
                              decimal: true,
                              signed: true,
                            ),
                            inputFormatters: [
                              FilteringTextInputFormatter.allow(
                                RegExp(r'[0-9\.\-]'),
                              ),
                            ],
                            decoration: const InputDecoration(
                              labelText: 'Kinh độ (lng)',
                              border: OutlineInputBorder(),
                            ),
                            onChanged: (_) => _syncPinFromFields(),
                            validator: (v) {
                              final n = double.tryParse(
                                (v ?? '').trim().replaceAll(',', '.'),
                              );
                              if (n == null || n < -180 || n > 180) {
                                return 'Lng -180…180';
                              }
                              return null;
                            },
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _radiusController,
                      enabled: !_saving,
                      keyboardType: const TextInputType.numberWithOptions(
                        decimal: true,
                      ),
                      decoration: const InputDecoration(
                        labelText: 'Bán kính giao (km)',
                        border: OutlineInputBorder(),
                        helperText:
                            'Khách ngoài bán kính sẽ bị từ chối địa chỉ.',
                      ),
                      validator: (v) {
                        final n = double.tryParse(
                          (v ?? '').trim().replaceAll(',', '.'),
                        );
                        if (n == null || n <= 0) {
                          return 'Bán kính phải > 0.';
                        }
                        return null;
                      },
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'Tìm địa chỉ hoặc chạm bản đồ để đặt pin',
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 8),
                    TextField(
                      controller: _searchController,
                      enabled: !_saving,
                      decoration: InputDecoration(
                        hintText: 'Gõ địa chỉ…',
                        border: const OutlineInputBorder(),
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
                            : null,
                      ),
                    ),
                    if (_suggestions.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Material(
                        color: theme.colorScheme.surfaceContainerLowest,
                        shape: RoundedRectangleBorder(
                          borderRadius: AppRadius.sm,
                          side: BorderSide(color: theme.colorScheme.outline),
                        ),
                        child: Column(
                          children: [
                            for (final p in _suggestions)
                              ListTile(
                                dense: true,
                                title: Text(p.label),
                                onTap: _saving
                                    ? null
                                    : () {
                                        _searchController.text = p.label;
                                        setState(() => _suggestions = const []);
                                        _setPin(
                                          LatLng(p.lat, p.lng),
                                          addressLabel: p.label,
                                        );
                                      },
                              ),
                          ],
                        ),
                      ),
                    ],
                    const SizedBox(height: 12),
                    OutlinedButton.icon(
                      onPressed: (_saving || _gpsLoading) ? null : _useGps,
                      icon: _gpsLoading
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.my_location),
                      label: const Text('Dùng vị trí hiện tại'),
                    ),
                    const SizedBox(height: 12),
                    SizedBox(
                      height: 260,
                      child: ClipRRect(
                        borderRadius: AppRadius.md,
                        child: FlutterMap(
                          mapController: _mapController,
                          options: MapOptions(
                            initialCenter: _pin,
                            initialZoom: _kDefaultZoom,
                            onMapReady: () {
                              _mapReady = true;
                              _mapController.move(_pin, _kDefaultZoom);
                            },
                            onTap:
                                _saving ? null : (tap, point) => _setPin(point),
                          ),
                          children: [
                            TileLayer(
                              urlTemplate:
                                  'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                              userAgentPackageName: 'vn.gastamde.gas_tam_de',
                            ),
                            MarkerLayer(
                              markers: [
                                Marker(
                                  point: _pin,
                                  width: 40,
                                  height: 40,
                                  child: Icon(
                                    Icons.store,
                                    color: theme.colorScheme.primary,
                                    size: 36,
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                    ),
                    if (_error != null) ...[
                      const SizedBox(height: 12),
                      Text(
                        _error!,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.error,
                        ),
                      ),
                    ],
                    const SizedBox(height: 24),
                    FilledButton(
                      onPressed: _saving ? null : _save,
                      style: FilledButton.styleFrom(
                        minimumSize: const Size.fromHeight(52),
                      ),
                      child: _saving
                          ? SizedBox(
                              height: 22,
                              width: 22,
                              child: CircularProgressIndicator(
                                strokeWidth: 2.5,
                                color: theme.colorScheme.onPrimary,
                              ),
                            )
                          : const Text('Lưu vị trí cửa hàng'),
                    ),
                  ],
                ),
              ),
      ),
    );
  }

  void _syncPinFromFields() {
    final lat =
        double.tryParse(_latController.text.trim().replaceAll(',', '.'));
    final lng =
        double.tryParse(_lngController.text.trim().replaceAll(',', '.'));
    if (lat == null || lng == null) return;
    if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return;
    final next = LatLng(lat, lng);
    if ((next.latitude - _pin.latitude).abs() < 1e-9 &&
        (next.longitude - _pin.longitude).abs() < 1e-9) {
      return;
    }
    setState(() => _pin = next);
    if (_mapReady) {
      _mapController.move(next, _kDefaultZoom);
    }
  }

  String _fmtNum(double n) {
    if (n == n.roundToDouble()) return n.toStringAsFixed(0);
    return n.toStringAsFixed(2);
  }
}
