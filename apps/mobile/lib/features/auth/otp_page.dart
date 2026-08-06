import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import '_auth_widgets.dart';
import 'auth_api.dart';
import 'auth_models.dart';
import 'auth_session.dart';

/// Step 2 — 6-digit OTP input with individual digit boxes.
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
  final _focusNode = FocusNode();
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
    _controller.addListener(_onCodeChanged);
    _focusNode.addListener(_onFocusChanged);
  }

  @override
  void dispose() {
    _timer?.cancel();
    _controller.removeListener(_onCodeChanged);
    _focusNode.removeListener(_onFocusChanged);
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  void _onCodeChanged() {
    setState(() {
      if (_error != null) _error = null;
    });
    if (_controller.text.trim().length == 6 && !_busy) {
      _focusNode.unfocus();
      _verify();
    }
  }

  void _onFocusChanged() => setState(() {});

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
    if (_busy) return;
    final code = _controller.text.trim();
    if (code.length != 6) {
      setState(() => _error = 'Nhập đủ 6 số của mã OTP.');
      _focusNode.requestFocus();
      return;
    }
    setState(() {
      _error = null;
      _loading = true;
    });
    try {
      final result = await ref.read(authApiProvider).verifyOtp(
            phone: widget.args.phone,
            code: code,
          );
      if (!mounted) return;
      await ref
          .read(authSessionProvider.notifier)
          .setSession(AuthSession.fromVerify(result));
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
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(
            'Đã gửi lại tới ${result.phoneMasked.isNotEmpty ? result.phoneMasked : widget.args.phoneMasked}',
          ),
        ));
      }
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.displayMessage);
      if (e.retryAfterSec != null && e.retryAfterSec! > 0) {
        _startCooldown(e.retryAfterSec!);
      }
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Không gửi lại được. Thử lại.');
    } finally {
      if (mounted) setState(() => _resending = false);
    }
  }

  String get _digits => _controller.text.trim();
  bool get _busy => _loading || _resending;

  @override
  Widget build(BuildContext context) {
    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle.light,
      child: Scaffold(
        backgroundColor: AppColors.obsidian,
        body: Stack(
          children: [
            const Positioned.fill(
              child: CustomPaint(painter: FlameAmbientPainter()),
            ),
            SafeArea(
              child: AuthScrollBody(
                top: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Padding(
                      padding: const EdgeInsets.fromLTRB(8, 8, 16, 0),
                      child: Row(
                        children: [
                          if (widget.onBack != null)
                            IconButton(
                              icon: const Icon(Icons.arrow_back_ios_new_rounded,
                                  color: AppColors.onDark, size: 20),
                              onPressed: _busy ? null : widget.onBack,
                            ),
                          const Spacer(),
                          const AuthStepChip(step: 2, total: 2),
                        ],
                      ),
                    ),
                    const SizedBox(height: 32),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 28),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Nhập mã\nxác thực',
                            style: Theme.of(context)
                                .textTheme
                                .displaySmall
                                ?.copyWith(
                                  color: AppColors.onDark,
                                  fontWeight: FontWeight.w900,
                                  letterSpacing: -1.5,
                                  height: 1.0,
                                ),
                          ),
                          const SizedBox(height: 14),
                          RichText(
                            text: TextSpan(
                              style: Theme.of(context)
                                  .textTheme
                                  .bodyLarge
                                  ?.copyWith(
                                    color: AppColors.onDark
                                        .withValues(alpha: 0.60),
                                  ),
                              children: [
                                const TextSpan(text: 'Mã 6 số đã gửi tới '),
                                TextSpan(
                                  text: widget.args.phoneMasked,
                                  style: const TextStyle(
                                    color: AppColors.amber,
                                    fontWeight: FontWeight.w700,
                                  ),
                                ),
                              ],
                            ),
                          ),
                          if (_devCode != null && _devCode!.isNotEmpty) ...[
                            const SizedBox(height: 12),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 14, vertical: 8),
                              decoration: BoxDecoration(
                                color: AppColors.amber.withValues(alpha: 0.12),
                                borderRadius: AppRadius.sm,
                                border: Border.all(
                                    color:
                                        AppColors.amber.withValues(alpha: 0.3)),
                              ),
                              child: Text(
                                'Dev mode: mã OTP là $_devCode',
                                style: const TextStyle(
                                  color: AppColors.amber,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
                bottom: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 24, 16, 0),
                  child: AuthCard(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        // The digit boxes are decoration only; a transparent,
                        // full-size text field sits on top so a tap focuses it
                        // directly. A zero-size field never gets a browser
                        // keyboard on mobile web.
                        SizedBox(
                          height: kOtpBoxHeight,
                          child: Stack(
                            fit: StackFit.expand,
                            children: [
                              IgnorePointer(
                                child: OtpBoxRow(
                                  digits: _digits,
                                  focused: _focusNode.hasFocus,
                                ),
                              ),
                              TextField(
                                controller: _controller,
                                focusNode: _focusNode,
                                // Autofocus keeps the keyboard from step 1 up
                                // on mobile web, where a programmatic focus
                                // without a user gesture cannot reopen it.
                                autofocus: true,
                                enabled: !_loading,
                                keyboardType: TextInputType.number,
                                textInputAction: TextInputAction.done,
                                autofillHints: const [
                                  AutofillHints.oneTimeCode,
                                ],
                                inputFormatters: [
                                  FilteringTextInputFormatter.digitsOnly,
                                  LengthLimitingTextInputFormatter(6),
                                ],
                                textAlign: TextAlign.center,
                                showCursor: false,
                                enableInteractiveSelection: false,
                                enableSuggestions: false,
                                style: const TextStyle(
                                  color: Colors.transparent,
                                  fontSize: 24,
                                  height: 1.0,
                                ),
                                decoration: const InputDecoration(
                                  contentPadding: EdgeInsets.zero,
                                  isCollapsed: true,
                                  filled: false,
                                  border: InputBorder.none,
                                  focusedBorder: InputBorder.none,
                                  enabledBorder: InputBorder.none,
                                  disabledBorder: InputBorder.none,
                                ),
                                onSubmitted: (_) => _verify(),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 10),
                        AnimatedOpacity(
                          opacity: _focusNode.hasFocus ? 0 : 1,
                          duration: const Duration(milliseconds: 200),
                          child: Center(
                            child: Text(
                              'Chạm vào ô để mở bàn phím',
                              style: TextStyle(
                                color: AppColors.onDark.withValues(alpha: 0.45),
                                fontSize: 12,
                              ),
                            ),
                          ),
                        ),
                        if (_error != null) ...[
                          const SizedBox(height: 12),
                          AuthErrorText(_error!),
                        ],
                        const SizedBox(height: 12),
                        Center(
                          child: TextButton(
                            onPressed:
                                (_cooldown > 0 || _busy) ? null : _resend,
                            style: TextButton.styleFrom(
                              foregroundColor: AppColors.amber,
                            ),
                            child: Text(
                              _cooldown > 0
                                  ? 'Gửi lại sau ${_cooldown}s'
                                  : (_resending ? 'Đang gửi…' : 'Gửi lại mã'),
                              style:
                                  const TextStyle(fontWeight: FontWeight.w600),
                            ),
                          ),
                        ),
                        const SizedBox(height: 8),
                        GradientCTAButton(
                          label: 'Xác nhận',
                          loading: _loading,
                          onTap: _verify,
                          enabled: _digits.length == 6,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
