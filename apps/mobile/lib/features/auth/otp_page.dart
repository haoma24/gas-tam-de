import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'auth_api.dart';
import 'auth_models.dart';
import 'auth_session.dart';

/// Step 2 — enter 6-digit OTP and verify → JWT session.
class OtpPage extends ConsumerStatefulWidget {
  const OtpPage({
    super.key,
    required this.args,
    required this.onVerified,
    this.onBack,
  });

  final OtpNavArgs args;
  final VoidCallback onVerified;
  final VoidCallback? onBack;

  @override
  ConsumerState<OtpPage> createState() => _OtpPageState();
}

class _OtpPageState extends ConsumerState<OtpPage> {
  final _controller = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _loading = false;
  bool _resending = false;
  String? _error;
  String? _devCode;
  late int _cooldown;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _devCode = widget.args.devCode;
    _startCooldown(widget.args.resendAfterSec);
  }

  @override
  void dispose() {
    _timer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _startCooldown(int seconds) {
    _timer?.cancel();
    setState(() => _cooldown = seconds < 0 ? 0 : seconds);
    if (_cooldown <= 0) return;
    _timer = Timer.periodic(const Duration(seconds: 1), (t) {
      if (!mounted) {
        t.cancel();
        return;
      }
      if (_cooldown <= 1) {
        t.cancel();
        setState(() => _cooldown = 0);
      } else {
        setState(() => _cooldown -= 1);
      }
    });
  }

  Future<void> _verify() async {
    setState(() => _error = null);
    if (!(_formKey.currentState?.validate() ?? false)) return;

    final code = _controller.text.trim();
    setState(() => _loading = true);
    try {
      final result = await ref.read(authApiProvider).verifyOtp(
            phone: widget.args.phone,
            code: code,
          );
      if (!mounted) return;
      ref.read(authSessionProvider.notifier).state =
          AuthSession.fromVerify(result);
      widget.onVerified();
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.displayMessage);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Có lỗi xảy ra. Thử lại.');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _resend() async {
    if (_cooldown > 0 || _resending || _loading) return;
    setState(() {
      _resending = true;
      _error = null;
    });
    try {
      final result =
          await ref.read(authApiProvider).requestOtp(widget.args.phone);
      if (!mounted) return;
      setState(() => _devCode = result.devCode);
      _startCooldown(result.resendAfterSec);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            'Đã gửi lại mã tới ${result.phoneMasked.isNotEmpty ? result.phoneMasked : widget.args.phoneMasked}',
          ),
        ),
      );
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.displayMessage);
      if (e.retryAfterSec != null && e.retryAfterSec! > 0) {
        _startCooldown(e.retryAfterSec!);
      }
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Không gửi lại được mã. Thử lại.');
    } finally {
      if (mounted) setState(() => _resending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final busy = _loading || _resending;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Nhập mã OTP'),
        leading: widget.onBack == null
            ? null
            : IconButton(
                icon: const Icon(Icons.arrow_back),
                onPressed: busy ? null : widget.onBack,
              ),
      ),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Xác nhận OTP',
                  style: theme.textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Mã 6 số đã gửi tới ${widget.args.phoneMasked}.',
                  style: theme.textTheme.bodyLarge?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                if (_devCode != null && _devCode!.isNotEmpty) ...[
                  const SizedBox(height: 16),
                  DecoratedBox(
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Text(
                        'Dev: mã OTP là $_devCode (chỉ hiện khi OTP_DEV_REVEAL).',
                        style: theme.textTheme.bodyMedium,
                      ),
                    ),
                  ),
                ],
                const SizedBox(height: 28),
                TextFormField(
                  controller: _controller,
                  enabled: !_loading,
                  keyboardType: TextInputType.number,
                  textInputAction: TextInputAction.done,
                  autofillHints: const [AutofillHints.oneTimeCode],
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly,
                    LengthLimitingTextInputFormatter(6),
                  ],
                  decoration: const InputDecoration(
                    labelText: 'Mã OTP',
                    hintText: '••••••',
                    border: OutlineInputBorder(),
                    prefixIcon: Icon(Icons.pin_outlined),
                  ),
                  validator: (value) {
                    final v = value?.trim() ?? '';
                    if (v.length != 6) return 'Nhập đủ 6 chữ số.';
                    return null;
                  },
                  onFieldSubmitted: (_) {
                    if (!_loading) _verify();
                  },
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
                const SizedBox(height: 8),
                Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton(
                    onPressed: (_cooldown > 0 || busy) ? null : _resend,
                    child: Text(
                      _cooldown > 0
                          ? 'Gửi lại sau ${_cooldown}s'
                          : (_resending ? 'Đang gửi…' : 'Gửi lại mã'),
                    ),
                  ),
                ),
                const Spacer(),
                FilledButton(
                  onPressed: _loading ? null : _verify,
                  style: FilledButton.styleFrom(
                    minimumSize: const Size.fromHeight(56),
                    textStyle: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  child: _loading
                      ? SizedBox(
                          height: 22,
                          width: 22,
                          child: CircularProgressIndicator(
                            strokeWidth: 2.5,
                            color: theme.colorScheme.onPrimary,
                          ),
                        )
                      : const Text('Xác nhận'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
