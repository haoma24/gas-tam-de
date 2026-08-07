import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/app_theme.dart';

/// Step badge.
class AuthStepChip extends StatelessWidget {
  const AuthStepChip({super.key, required this.step, required this.total});
  final int step;
  final int total;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
      decoration: BoxDecoration(
        color: AppColors.ash.withValues(alpha: 0.5),
        borderRadius: AppRadius.pill,
      ),
      child: Text(
        'Bước $step / $total',
        style: const TextStyle(
          color: AppColors.onDark,
          fontSize: 12,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

/// Auth page body: header block on top, input card pushed to the bottom.
///
/// Scrollable so the on-screen keyboard (which shrinks the viewport) neither
/// overflows the layout nor hides the field being typed into — the focused
/// field is scrolled into view instead.
class AuthScrollBody extends StatelessWidget {
  const AuthScrollBody({
    super.key,
    required this.top,
    required this.bottom,
    this.bottomPadding = 32,
  });

  final Widget top;
  final Widget bottom;
  final double bottomPadding;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final minHeight = constraints.maxHeight.isFinite
            ? math.max(0.0, constraints.maxHeight - bottomPadding)
            : 0.0;
        return SingleChildScrollView(
          padding: EdgeInsets.only(bottom: bottomPadding),
          child: ConstrainedBox(
            constraints: BoxConstraints(minHeight: minHeight),
            // spaceBetween (not Spacer) — Expanded cannot resolve inside the
            // unbounded height of a scroll view.
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [top, bottom],
            ),
          ),
        );
      },
    );
  }
}

/// Dark card container used on auth screens.
class AuthCard extends StatelessWidget {
  const AuthCard({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(24, 28, 24, 28),
      decoration: BoxDecoration(
        color: AppColors.coal.withValues(alpha: 0.92),
        borderRadius: AppRadius.xl,
        border: Border.all(
          color: AppColors.ash.withValues(alpha: 0.5),
          width: 1,
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.4),
            blurRadius: 32,
            offset: const Offset(0, 16),
          ),
        ],
      ),
      child: child,
    );
  }
}

/// Amber→Fire gradient CTA button with press scale animation.
class GradientCTAButton extends StatefulWidget {
  const GradientCTAButton({
    super.key,
    required this.label,
    required this.onTap,
    this.loading = false,
    this.enabled = true,
  });

  final String label;
  final VoidCallback onTap;
  final bool loading;
  final bool enabled;

  @override
  State<GradientCTAButton> createState() => _GradientCTAButtonState();
}

class _GradientCTAButtonState extends State<GradientCTAButton>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double> _scale;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 100));
    _scale = Tween(begin: 1.0, end: 0.96)
        .animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut));
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final active = widget.enabled && !widget.loading;
    return GestureDetector(
      onTapDown: active ? (_) => _ctrl.forward() : null,
      onTapUp: active
          ? (_) {
              _ctrl.reverse();
              widget.onTap();
            }
          : null,
      onTapCancel: () => _ctrl.reverse(),
      child: ScaleTransition(
        scale: _scale,
        child: AnimatedOpacity(
          opacity: active ? 1.0 : 0.5,
          duration: const Duration(milliseconds: 200),
          child: Container(
            height: 56,
            decoration: BoxDecoration(
              gradient: active
                  ? const LinearGradient(
                      colors: [AppColors.amber, AppColors.fire],
                      begin: Alignment.centerLeft,
                      end: Alignment.centerRight,
                    )
                  : null,
              color: active ? null : AppColors.ash,
              borderRadius: AppRadius.pill,
              boxShadow: active
                  ? [
                      BoxShadow(
                        color: AppColors.fire.withValues(alpha: 0.4),
                        blurRadius: 18,
                        offset: const Offset(0, 6),
                      ),
                    ]
                  : null,
            ),
            child: Center(
              child: widget.loading
                  ? const SizedBox(
                      width: 22,
                      height: 22,
                      child: CircularProgressIndicator(
                        strokeWidth: 2.5,
                        color: AppColors.obsidian,
                      ),
                    )
                  : Text(
                      widget.label,
                      style: const TextStyle(
                        color: AppColors.obsidian,
                        fontWeight: FontWeight.w800,
                        fontSize: 16,
                        letterSpacing: 0.2,
                      ),
                    ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Dark-background text field for auth screens.
class DarkTextField extends StatelessWidget {
  const DarkTextField({
    super.key,
    required this.controller,
    required this.enabled,
    required this.keyboardType,
    required this.textInputAction,
    required this.hint,
    required this.prefixIcon,
    this.autofillHints,
    this.inputFormatters,
    this.validator,
    this.onSubmitted,
  });

  final TextEditingController controller;
  final bool enabled;
  final TextInputType keyboardType;
  final TextInputAction textInputAction;
  final String hint;
  final Widget prefixIcon;
  final Iterable<String>? autofillHints;
  final List<TextInputFormatter>? inputFormatters;
  final FormFieldValidator<String>? validator;
  final void Function(String)? onSubmitted;

  @override
  Widget build(BuildContext context) {
    return TextFormField(
      controller: controller,
      enabled: enabled,
      keyboardType: keyboardType,
      textInputAction: textInputAction,
      autofillHints: autofillHints,
      inputFormatters: inputFormatters,
      style: const TextStyle(
        color: AppColors.onDark,
        fontSize: 17,
        fontWeight: FontWeight.w600,
      ),
      cursorColor: AppColors.amber,
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(
          color: AppColors.onDark.withValues(alpha: 0.30),
          fontWeight: FontWeight.w400,
          fontSize: 16,
        ),
        prefixIcon: IconTheme(
          data: const IconThemeData(color: AppColors.amber, size: 20),
          child: prefixIcon,
        ),
        filled: true,
        fillColor: AppColors.ash.withValues(alpha: 0.35),
        border: OutlineInputBorder(
          borderRadius: AppRadius.md,
          borderSide: BorderSide(color: AppColors.ash.withValues(alpha: 0.4)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: AppRadius.md,
          borderSide: BorderSide(color: AppColors.ash.withValues(alpha: 0.4)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: AppRadius.md,
          borderSide: const BorderSide(color: AppColors.amber, width: 1.5),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: AppRadius.md,
          borderSide: const BorderSide(color: AppColors.danger),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: AppRadius.md,
          borderSide: const BorderSide(color: AppColors.danger, width: 1.5),
        ),
        errorStyle: const TextStyle(color: AppColors.danger, fontSize: 12),
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      ),
      validator: validator,
      onFieldSubmitted: onSubmitted,
    );
  }
}

/// Height of a single [OtpBoxRow] digit box.
const double kOtpBoxHeight = 58;

/// Visible OTP entry: digit boxes plus a real, tappable text field.
///
/// Mobile browsers (especially iOS Safari) often refuse to open the keyboard for
/// zero-size or fully transparent inputs. A [Listener] on pointer-down keeps
/// focus in the same user gesture when the user taps the digit row.
class OtpEntryBlock extends StatelessWidget {
  const OtpEntryBlock({
    super.key,
    required this.controller,
    required this.focusNode,
    required this.enabled,
    required this.digits,
    required this.focused,
    this.autofocus = false,
    this.onSubmitted,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final bool enabled;
  final String digits;
  final bool focused;
  final bool autofocus;
  final VoidCallback? onSubmitted;

  @override
  Widget build(BuildContext context) {
    return Listener(
      behavior: HitTestBehavior.translucent,
      onPointerDown: enabled ? (_) => focusNode.requestFocus() : null,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          OtpBoxRow(digits: digits, focused: focused),
          const SizedBox(height: 14),
          TextField(
            controller: controller,
            focusNode: focusNode,
            autofocus: autofocus,
            enabled: enabled,
            keyboardType: TextInputType.number,
            textInputAction: TextInputAction.done,
            autofillHints: const [AutofillHints.oneTimeCode],
            inputFormatters: [
              FilteringTextInputFormatter.digitsOnly,
              LengthLimitingTextInputFormatter(6),
            ],
            textAlign: TextAlign.center,
            style: const TextStyle(
              color: AppColors.onDark,
              fontSize: 22,
              fontWeight: FontWeight.w800,
              letterSpacing: 6,
            ),
            cursorColor: AppColors.amber,
            decoration: InputDecoration(
              hintText: 'Nhập 6 số OTP',
              hintStyle: TextStyle(
                color: AppColors.onDark.withValues(alpha: 0.35),
                fontWeight: FontWeight.w500,
                fontSize: 16,
                letterSpacing: 0,
              ),
              filled: true,
              fillColor: AppColors.ash.withValues(alpha: 0.35),
              border: OutlineInputBorder(
                borderRadius: AppRadius.md,
                borderSide:
                    BorderSide(color: AppColors.ash.withValues(alpha: 0.4)),
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: AppRadius.md,
                borderSide:
                    BorderSide(color: AppColors.ash.withValues(alpha: 0.4)),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: AppRadius.md,
                borderSide:
                    const BorderSide(color: AppColors.amber, width: 1.5),
              ),
              contentPadding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
            ),
            onSubmitted: onSubmitted != null ? (_) => onSubmitted!() : null,
          ),
        ],
      ),
    );
  }
}

/// 6 individual digit boxes reflecting controller value.
class OtpBoxRow extends StatelessWidget {
  const OtpBoxRow({super.key, required this.digits, this.focused = true});
  final String digits;
  final bool focused;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: List.generate(6, (i) {
        final filled = i < digits.length;
        final active = focused && i == digits.length;
        return AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          width: 46,
          height: kOtpBoxHeight,
          decoration: BoxDecoration(
            color: filled
                ? AppColors.amber.withValues(alpha: 0.15)
                : AppColors.ash.withValues(alpha: 0.25),
            borderRadius: AppRadius.sm,
            border: Border.all(
              color: active
                  ? AppColors.amber
                  : filled
                      ? AppColors.amber.withValues(alpha: 0.5)
                      : AppColors.ash.withValues(alpha: 0.4),
              width: active ? 2 : 1,
            ),
          ),
          alignment: Alignment.center,
          child: Text(
            filled ? digits[i] : '',
            style: const TextStyle(
              color: AppColors.onDark,
              fontSize: 24,
              fontWeight: FontWeight.w800,
            ),
          ),
        );
      }),
    );
  }
}

/// Error text with icon for auth screens.
class AuthErrorText extends StatelessWidget {
  const AuthErrorText(this.message, {super.key});
  final String message;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Padding(
          padding: EdgeInsets.only(top: 2),
          child: Icon(Icons.error_outline_rounded,
              size: 15, color: AppColors.danger),
        ),
        const SizedBox(width: 6),
        Flexible(
          child: Text(
            message,
            style: const TextStyle(
              color: AppColors.danger,
              fontSize: 13,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      ],
    );
  }
}
