import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'desk_settings_api.dart';
import 'desk_settings_models.dart';
import 'new_order_voice.dart';

/// Admin — ngưỡng màu chờ + interval TTS Order Desk.
class AdminDeskSettingsPage extends ConsumerStatefulWidget {
  const AdminDeskSettingsPage({super.key, required this.onBack});

  final VoidCallback onBack;

  @override
  ConsumerState<AdminDeskSettingsPage> createState() =>
      _AdminDeskSettingsPageState();
}

class _AdminDeskSettingsPageState extends ConsumerState<AdminDeskSettingsPage> {
  final _blue = TextEditingController();
  final _orange = TextEditingController();
  final _red = TextEditingController();
  final _interval = TextEditingController();
  bool _alertEnabled = true;
  bool _loading = true;
  bool _saving = false;
  String? _error;
  bool _testingVoice = false;
  String? _voiceStatus;
  bool _voiceIsVietnamese = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _blue.dispose();
    _orange.dispose();
    _red.dispose();
    _interval.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final s = await ref.read(deskSettingsApiProvider).get();
      if (!mounted) return;
      _apply(s);
      setState(() => _loading = false);
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Không tải được cấu hình desk.';
        _loading = false;
      });
    }
  }

  void _apply(DeskSettings s) {
    _blue.text = '${s.waitBlueMaxMin}';
    _orange.text = '${s.waitOrangeMaxMin}';
    _red.text = '${s.waitRedMaxMin}';
    _interval.text = '${s.alertIntervalSec}';
    _alertEnabled = s.alertEnabled;
  }

  Future<void> _save() async {
    final blue = int.tryParse(_blue.text.trim());
    final orange = int.tryParse(_orange.text.trim());
    final red = int.tryParse(_red.text.trim());
    final interval = int.tryParse(_interval.text.trim());
    if (blue == null || orange == null || red == null || interval == null) {
      setState(() => _error = 'Nhập số hợp lệ.');
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final saved = await ref.read(deskSettingsApiProvider).put(
            DeskSettings(
              waitBlueMaxMin: blue,
              waitOrangeMaxMin: orange,
              waitRedMaxMin: red,
              alertEnabled: _alertEnabled,
              alertIntervalSec: interval,
            ),
          );
      if (!mounted) return;
      _apply(saved);
      ref.invalidate(deskSettingsProvider);
      setState(() => _saving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Đã lưu cấu hình Order Desk.')),
      );
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _saving = false;
        _error = 'Lưu thất bại. Kiểm tra 0 < xanh < cam < đỏ và interval ≥ 30.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Cấu hình Order Desk'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: _saving ? null : widget.onBack,
        ),
      ),
      body: SafeArea(
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.fromLTRB(24, 16, 24, 32),
                children: [
                  Text(
                    'Màu badge thời gian chờ (phút)',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Xanh dương nếu chờ < ngưỡng xanh; cam nếu < cam; đỏ từ ngưỡng cam trở lên.',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 16),
                  _numField(_blue, 'Ngưỡng xanh (phút)', enabled: !_saving),
                  const SizedBox(height: 12),
                  _numField(_orange, 'Ngưỡng cam (phút)', enabled: !_saving),
                  const SizedBox(height: 12),
                  _numField(_red, 'Ngưỡng đỏ (phút)', enabled: !_saving),
                  const SizedBox(height: 28),
                  Text(
                    'Thông báo giọng nói',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    title: const Text('Bật thông báo đơn chờ'),
                    subtitle: const Text('Đọc «Bạn có N đơn chưa giao» theo chu kỳ'),
                    value: _alertEnabled,
                    onChanged: _saving
                        ? null
                        : (v) => setState(() => _alertEnabled = v),
                  ),
                  _numField(
                    _interval,
                    'Chu kỳ thông báo (giây)',
                    enabled: !_saving,
                    helper: 'Tối thiểu 30 giây (mặc định 300 = 5 phút)',
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    onPressed: _testingVoice ? null : _testVoice,
                    icon: _testingVoice
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.volume_up),
                    label: const Text('Nghe thử giọng đọc'),
                  ),
                  if (_voiceStatus != null) ...[
                    const SizedBox(height: 8),
                    Text(
                      _voiceStatus!,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: _voiceIsVietnamese
                            ? theme.colorScheme.onSurfaceVariant
                            : theme.colorScheme.error,
                      ),
                    ),
                  ],
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(
                      _error!,
                      style: TextStyle(color: theme.colorScheme.error),
                    ),
                  ],
                  const SizedBox(height: 24),
                  FilledButton(
                    onPressed: _saving ? null : _save,
                    style: FilledButton.styleFrom(
                      minimumSize: const Size.fromHeight(52),
                    ),
                    child: _saving
                        ? const SizedBox(
                            width: 22,
                            height: 22,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Text('Lưu cấu hình'),
                  ),
                ],
              ),
      ),
    );
  }

  /// Speaks a sample and reports which voice the device actually used. When no
  /// Vietnamese voice exists the engine silently reads Vietnamese text with an
  /// English voice, so the shop needs to be told to install one — the app
  /// cannot ship a voice itself.
  Future<void> _testVoice() async {
    setState(() {
      _testingVoice = true;
      _voiceStatus = null;
    });
    final voice = await NewOrderVoice.speakSample();
    if (!mounted) return;
    setState(() {
      _testingVoice = false;
      _voiceIsVietnamese = voice != null;
      _voiceStatus = voice != null
          ? 'Đang dùng giọng: ${voice.label}'
          : 'Máy này chưa có giọng tiếng Việt nên hệ thống đọc bằng giọng '
              'mặc định (tiếng Anh). Cài giọng tiếng Việt rồi tải lại trang: '
              'Android → Cài đặt › Ngôn ngữ › Đầu ra văn bản sang lời nói › '
              'cài dữ liệu Tiếng Việt. Windows → Settings › Time & language › '
              'Speech › Add voices › Vietnamese.';
    });
  }

  Widget _numField(
    TextEditingController c,
    String label, {
    required bool enabled,
    String? helper,
  }) {
    return TextField(
      controller: c,
      enabled: enabled,
      keyboardType: TextInputType.number,
      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
      decoration: InputDecoration(
        labelText: label,
        helperText: helper,
        border: const OutlineInputBorder(),
      ),
    );
  }
}
