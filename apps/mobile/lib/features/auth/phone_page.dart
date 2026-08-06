import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import '_auth_widgets.dart';
import 'auth_api.dart';
import 'auth_models.dart';
import 'phone_utils.dart';

/// Step 1 — dark-branded phone entry screen.
class PhonePage extends ConsumerStatefulWidget {
  const PhonePage({
    super.key,
    required this.onOtpSent,
    this.onBack,
  });

  final void Function(OtpNavArgs args) onOtpSent;
  final VoidCallback? onBack;

  @override
  ConsumerState<PhonePage> createState() => _PhonePageState();
}

class _PhonePageState extends ConsumerState<PhonePage> {
  final _controller = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _loading = false;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() => _error = null);
    if (!(_formKey.currentState?.validate() ?? false)) return;
    final phone = _controller.text.trim();
    setState(() => _loading = true);
    try {
      final result = await ref.read(authApiProvider).requestOtp(phone);
      if (!mounted) return;
      widget.onOtpSent(OtpNavArgs(
        phone: phone,
        phoneMasked: result.phoneMasked.isNotEmpty
            ? result.phoneMasked
            : maskVnPhone(phone),
        resendAfterSec: result.resendAfterSec,
        expiresInSec: result.expiresInSec,
        devCode: result.devCode,
      ));
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
                              onPressed: _loading ? null : widget.onBack,
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
                          Text(
                            'Nhập SĐT để nhận mã OTP 6 chữ số.',
                            style: Theme.of(context)
                                .textTheme
                                .bodyLarge
                                ?.copyWith(
                                  color:
                                      AppColors.onDark.withValues(alpha: 0.6),
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
                      key: _formKey,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          DarkTextField(
                            controller: _controller,
                            enabled: !_loading,
                            keyboardType: TextInputType.phone,
                            textInputAction: TextInputAction.done,
                            autofillHints: const [
                              AutofillHints.telephoneNumber
                            ],
                            inputFormatters: [
                              FilteringTextInputFormatter.allow(
                                  RegExp(r'[\d+\s\-.]')),
                              LengthLimitingTextInputFormatter(16),
                            ],
                            hint: '09xx xxx xxx',
                            prefixIcon:
                                const Icon(Icons.phone_outlined, size: 20),
                            validator: (v) {
                              final s = v?.trim() ?? '';
                              if (s.isEmpty) return 'Vui lòng nhập SĐT.';
                              if (!isValidVnMobile(s)) {
                                return 'SĐT không hợp lệ.';
                              }
                              return null;
                            },
                            onSubmitted: (_) {
                              if (!_loading) _submit();
                            },
                          ),
                          if (_error != null) ...[
                            const SizedBox(height: 10),
                            AuthErrorText(_error!),
                          ],
                          const SizedBox(height: 20),
                          GradientCTAButton(
                            label: 'Gửi mã OTP',
                            loading: _loading,
                            onTap: _submit,
                          ),
                        ],
                      ),
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
