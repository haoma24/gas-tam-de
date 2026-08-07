import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import '_auth_widgets.dart';
import 'auth_api.dart';
import 'auth_models.dart';
import 'auth_session.dart';
import 'phone_utils.dart';

/// Customer login: phone → OTP on one route.
///
/// Both steps stay mounted in an [IndexedStack] so the OTP field exists in the
/// tree before navigation. After «Gửi mã OTP» we flip the index and call
/// [FocusNode.requestFocus] synchronously — still inside the button's user
/// gesture — which mobile browsers require to open the keyboard. A separate
/// `go()` after `await requestOtp()` breaks that chain.
class CustomerAuthFlowPage extends ConsumerStatefulWidget {
  const CustomerAuthFlowPage({
    super.key,
    required this.onVerified,
    this.onBack,
    this.initialOtpArgs,
  });

  final VoidCallback onVerified;
  final VoidCallback? onBack;

  /// When opening `/auth/otp` with [GoRouter] `extra` (refresh / deep link).
  final OtpNavArgs? initialOtpArgs;

  @override
  ConsumerState<CustomerAuthFlowPage> createState() =>
      _CustomerAuthFlowPageState();
}

class _CustomerAuthFlowPageState extends ConsumerState<CustomerAuthFlowPage> {
  static const _phoneIndex = 0;
  static const _otpIndex = 1;

  late int _index;
  late String _phone;
  late String _phoneMasked;
  late int _resendAfterSec;
  String? _devCode;

  final _phoneController = TextEditingController();
  final _phoneFormKey = GlobalKey<FormState>();
  final _otpController = TextEditingController();
  final _otpFocus = FocusNode();

  bool _otpLoading = false;
  bool _otpSending = false;
  bool _otpResending = false;
  String? _phoneError;
  String? _otpError;
  int _cooldown = 0;
  Timer? _cooldownTimer;

  @override
  void initState() {
    super.initState();
    final seed = widget.initialOtpArgs;
    if (seed != null) {
      _index = _otpIndex;
      _phone = seed.phone;
      _phoneMasked = seed.phoneMasked;
      _resendAfterSec = seed.resendAfterSec;
      _devCode = seed.devCode;
      _cooldown = seed.resendAfterSec;
      if (seed.requestOtpOnMount) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) _sendOtp(initial: true);
        });
      } else if (_cooldown > 0) {
        _startCooldown(_cooldown);
      }
    } else {
      _index = _phoneIndex;
      _phone = '';
      _phoneMasked = '';
      _resendAfterSec = 60;
    }
    _otpController.addListener(_onOtpChanged);
    _otpFocus.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _cooldownTimer?.cancel();
    _otpController.removeListener(_onOtpChanged);
    _phoneController.dispose();
    _otpController.dispose();
    _otpFocus.dispose();
    super.dispose();
  }

  void _onOtpChanged() {
    setState(() {
      if (_otpError != null) _otpError = null;
    });
    if (_otpController.text.trim().length == 6 && !_otpBusy) {
      _otpFocus.unfocus();
      _verifyOtp();
    }
  }

  bool get _otpBusy => _otpLoading || _otpSending || _otpResending;

  void _startCooldown(int seconds) {
    _cooldownTimer?.cancel();
    setState(() => _cooldown = seconds < 0 ? 0 : seconds);
    if (_cooldown <= 0) return;
    _cooldownTimer = Timer.periodic(const Duration(seconds: 1), (t) {
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

  /// Validates phone, switches to OTP step, focuses field, then requests OTP.
  void _continueToOtp() {
    setState(() => _phoneError = null);
    if (!(_phoneFormKey.currentState?.validate() ?? false)) return;

    final phone = _phoneController.text.trim();
    _phone = phone;
    _phoneMasked = maskVnPhone(phone);
    setState(() => _index = _otpIndex);
    _otpFocus.requestFocus();
    unawaited(_sendOtp(initial: true));
  }

  Future<void> _sendOtp({required bool initial}) async {
    if (_otpBusy) return;
    setState(() {
      _otpError = null;
      if (initial) {
        _otpSending = true;
      } else {
        _otpResending = true;
      }
    });
    try {
      final result = await ref.read(authApiProvider).requestOtp(_phone);
      if (!mounted) return;
      setState(() {
        _phoneMasked = result.phoneMasked.isNotEmpty
            ? result.phoneMasked
            : maskVnPhone(_phone);
        _resendAfterSec = result.resendAfterSec;
        _devCode = result.devCode;
      });
      _startCooldown(result.resendAfterSec);
      if (!initial && context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(
            'Đã gửi lại tới ${_phoneMasked.isNotEmpty ? _phoneMasked : _phone}',
          ),
        ));
      }
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _otpError = e.displayMessage);
      if (e.retryAfterSec != null && e.retryAfterSec! > 0) {
        _startCooldown(e.retryAfterSec!);
      }
    } catch (_) {
      if (!mounted) return;
      setState(() => _otpError = 'Không gửi được mã. Thử lại.');
    } finally {
      if (mounted) {
        setState(() {
          _otpSending = false;
          _otpResending = false;
        });
      }
    }
  }

  Future<void> _verifyOtp() async {
    if (_otpBusy) return;
    final code = _otpController.text.trim();
    if (code.length != 6) {
      setState(() => _otpError = 'Nhập đủ 6 số của mã OTP.');
      _otpFocus.requestFocus();
      return;
    }
    setState(() {
      _otpError = null;
      _otpLoading = true;
    });
    try {
      final result = await ref.read(authApiProvider).verifyOtp(
            phone: _phone,
            code: code,
          );
      if (!mounted) return;
      await ref
          .read(authSessionProvider.notifier)
          .setSession(AuthSession.fromVerify(result));
      widget.onVerified();
    } on AuthApiException catch (e) {
      if (!mounted) return;
      setState(() => _otpError = e.displayMessage);
    } catch (_) {
      if (!mounted) return;
      setState(() => _otpError = 'Có lỗi xảy ra. Thử lại.');
    } finally {
      if (mounted) setState(() => _otpLoading = false);
    }
  }

  void _backFromOtp() {
    setState(() {
      _index = _phoneIndex;
      _otpError = null;
      _otpController.clear();
    });
  }

  @override
  Widget build(BuildContext context) {
    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle.light,
      child: Scaffold(
        backgroundColor: AppColors.obsidian,
        resizeToAvoidBottomInset: true,
        body: Stack(
          children: [
            const Positioned.fill(
              child: CustomPaint(painter: FlameAmbientPainter()),
            ),
            SafeArea(
              child: IndexedStack(
                index: _index,
                sizing: StackFit.expand,
                children: [
                  _buildPhoneStep(context),
                  _buildOtpStep(context),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPhoneStep(BuildContext context) {
    return AuthScrollBody(
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
                    onPressed: widget.onBack != null ? widget.onBack : null,
                  ),
                const Spacer(),
                const AuthStepChip(step: 1, total: 2),
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
                  'Số điện\nthoại',
                  style: Theme.of(context).textTheme.displaySmall?.copyWith(
                        color: AppColors.onDark,
                        fontWeight: FontWeight.w900,
                        letterSpacing: -1.5,
                        height: 1.0,
                      ),
                ),
                const SizedBox(height: 14),
                Text(
                  'Nhập SĐT để nhận mã OTP 6 chữ số.',
                  style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                        color: AppColors.onDark.withValues(alpha: 0.6),
                      ),
                ),
              ],
            ),
          ),
        ],
      ),
      bottom: Padding(
        padding: const EdgeInsets.fromLTRB(16, 24, 16, 0),
        child: AuthCard(
          child: Form(
            key: _phoneFormKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              mainAxisSize: MainAxisSize.min,
              children: [
                DarkTextField(
                  controller: _phoneController,
                  enabled: true,
                  keyboardType: TextInputType.phone,
                  textInputAction: TextInputAction.done,
                  autofillHints: const [AutofillHints.telephoneNumber],
                  inputFormatters: [
                    FilteringTextInputFormatter.allow(RegExp(r'[\d+\s\-.]')),
                    LengthLimitingTextInputFormatter(16),
                  ],
                  hint: '09xx xxx xxx',
                  prefixIcon: const Icon(Icons.phone_outlined, size: 20),
                  validator: (v) {
                    final s = v?.trim() ?? '';
                    if (s.isEmpty) return 'Vui lòng nhập SĐT.';
                    if (!isValidVnMobile(s)) return 'SĐT không hợp lệ.';
                    return null;
                  },
                  onSubmitted: (_) => _continueToOtp(),
                ),
                if (_phoneError != null) ...[
                  const SizedBox(height: 10),
                  AuthErrorText(_phoneError!),
                ],
                const SizedBox(height: 20),
                GradientCTAButton(
                  label: 'Gửi mã OTP',
                  loading: false,
                  onTap: _continueToOtp,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildOtpStep(BuildContext context) {
    final digits = _otpController.text.trim();
    return AuthScrollBody(
      top: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(8, 8, 16, 0),
            child: Row(
              children: [
                IconButton(
                  icon: const Icon(Icons.arrow_back_ios_new_rounded,
                      color: AppColors.onDark, size: 20),
                  onPressed: _otpBusy ? null : _backFromOtp,
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
                  style: Theme.of(context).textTheme.displaySmall?.copyWith(
                        color: AppColors.onDark,
                        fontWeight: FontWeight.w900,
                        letterSpacing: -1.5,
                        height: 1.0,
                      ),
                ),
                const SizedBox(height: 14),
                RichText(
                  text: TextSpan(
                    style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                          color: AppColors.onDark.withValues(alpha: 0.60),
                        ),
                    children: [
                      const TextSpan(text: 'Mã 6 số đã gửi tới '),
                      TextSpan(
                        text: _phoneMasked,
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
                    padding:
                        const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                    decoration: BoxDecoration(
                      color: AppColors.amber.withValues(alpha: 0.12),
                      borderRadius: AppRadius.sm,
                      border: Border.all(
                          color: AppColors.amber.withValues(alpha: 0.3)),
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
              if (_otpSending)
                const Padding(
                  padding: EdgeInsets.only(bottom: 16),
                  child: Center(
                    child: SizedBox(
                      width: 28,
                      height: 28,
                      child: CircularProgressIndicator(
                        strokeWidth: 2.5,
                        color: AppColors.amber,
                      ),
                    ),
                  ),
                )
              else
                OtpEntryBlock(
                  controller: _otpController,
                  focusNode: _otpFocus,
                  enabled: !_otpLoading,
                  digits: digits,
                  focused: _otpFocus.hasFocus,
                  onSubmitted: _verifyOtp,
                ),
              if (_otpError != null) ...[
                const SizedBox(height: 12),
                AuthErrorText(_otpError!),
              ],
              const SizedBox(height: 12),
              Center(
                child: TextButton(
                  onPressed: (_cooldown > 0 || _otpBusy)
                      ? null
                      : () => _sendOtp(initial: false),
                  style: TextButton.styleFrom(
                    foregroundColor: AppColors.amber,
                  ),
                  child: Text(
                    _cooldown > 0
                        ? 'Gửi lại sau ${_cooldown}s'
                        : (_otpResending ? 'Đang gửi…' : 'Gửi lại mã'),
                    style: const TextStyle(fontWeight: FontWeight.w600),
                  ),
                ),
              ),
              const SizedBox(height: 8),
              GradientCTAButton(
                label: 'Xác nhận',
                loading: _otpLoading,
                onTap: _verifyOtp,
                enabled: digits.length == 6,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
