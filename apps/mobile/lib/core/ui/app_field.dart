import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'app_tokens.dart';

/// Labelled text field. The label sits above the box (not floating) so forms
/// stay readable at a glance.
class AppTextField extends StatelessWidget {
  const AppTextField({
    super.key,
    this.label,
    this.hint,
    this.controller,
    this.keyboardType,
    this.textInputAction,
    this.obscureText = false,
    this.enabled = true,
    this.maxLines = 1,
    this.minLines,
    this.prefixIcon,
    this.suffix,
    this.helper,
    this.errorText,
    this.onChanged,
    this.onSubmitted,
    this.autofillHints,
    this.inputFormatters,
    this.focusNode,
    this.autofocus = false,
  });

  final String? label;
  final String? hint;
  final TextEditingController? controller;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final bool obscureText;
  final bool enabled;
  final int maxLines;
  final int? minLines;
  final IconData? prefixIcon;
  final Widget? suffix;
  final String? helper;
  final String? errorText;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final Iterable<String>? autofillHints;
  final List<TextInputFormatter>? inputFormatters;
  final FocusNode? focusNode;
  final bool autofocus;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (label != null) ...[
          Text(
            label!,
            style: context.text.labelLarge?.copyWith(color: p.inkMuted),
          ),
          const VGap(AppSpacing.sm),
        ],
        TextField(
          controller: controller,
          keyboardType: keyboardType,
          textInputAction: textInputAction,
          obscureText: obscureText,
          enabled: enabled,
          maxLines: obscureText ? 1 : maxLines,
          minLines: minLines,
          onChanged: onChanged,
          onSubmitted: onSubmitted,
          autofillHints: autofillHints,
          inputFormatters: inputFormatters,
          focusNode: focusNode,
          autofocus: autofocus,
          style: context.text.bodyLarge,
          decoration: InputDecoration(
            hintText: hint,
            prefixIcon: prefixIcon == null ? null : Icon(prefixIcon, size: 20),
            suffixIcon: suffix,
            errorText: errorText,
          ),
        ),
        if (helper != null && errorText == null) ...[
          const VGap(AppSpacing.xs),
          Text(
            helper!,
            style: context.text.bodySmall?.copyWith(color: p.inkMuted),
          ),
        ],
      ],
    );
  }
}

/// Rounded search input with a clear affordance.
class AppSearchField extends StatelessWidget {
  const AppSearchField({
    super.key,
    required this.controller,
    required this.onChanged,
    this.hint = 'Tìm kiếm…',
    this.onClear,
    this.busy = false,
  });

  final TextEditingController controller;
  final ValueChanged<String> onChanged;
  final String hint;
  final VoidCallback? onClear;
  final bool busy;

  @override
  Widget build(BuildContext context) {
    final hasText = controller.text.isNotEmpty;
    return TextField(
      controller: controller,
      onChanged: onChanged,
      style: context.text.bodyLarge,
      decoration: InputDecoration(
        hintText: hint,
        prefixIcon: const Icon(Icons.search_rounded, size: 20),
        suffixIcon: busy
            ? const Padding(
                padding: EdgeInsets.all(14),
                child: SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              )
            : hasText
                ? IconButton(
                    icon: const Icon(Icons.close_rounded, size: 18),
                    onPressed: () {
                      controller.clear();
                      onChanged('');
                      onClear?.call();
                    },
                  )
                : null,
        border: const OutlineInputBorder(
          borderRadius: AppRadius.full,
          borderSide: BorderSide.none,
        ),
        enabledBorder: const OutlineInputBorder(
          borderRadius: AppRadius.full,
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: AppRadius.full,
          borderSide: BorderSide(color: context.palette.ink, width: 1.5),
        ),
      ),
    );
  }
}

/// The single quantity stepper. Replaces three separate implementations
/// (product detail, order product picker, inventory dialog).
class QtyStepper extends StatelessWidget {
  const QtyStepper({
    super.key,
    required this.value,
    required this.onChanged,
    this.min = 0,
    this.max,
    this.compact = false,
  });

  final int value;
  final ValueChanged<int> onChanged;
  final int min;

  /// Upper bound (e.g. stock on hand). Null means unbounded.
  final int? max;

  final bool compact;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final size = compact ? 30.0 : 36.0;
    final canDecrease = value > min;
    final canIncrease = max == null || value < max!;

    return Container(
      decoration: BoxDecoration(
        color: p.surfaceSubtle,
        borderRadius: AppRadius.full,
        border: Border.all(color: p.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _StepButton(
            icon: Icons.remove_rounded,
            size: size,
            enabled: canDecrease,
            onTap: () => onChanged(value - 1),
          ),
          SizedBox(
            width: compact ? 28 : 34,
            child: Text(
              '$value',
              textAlign: TextAlign.center,
              style: context.text.titleSmall?.copyWith(
                fontFeatures: kTabularFigures,
              ),
            ),
          ),
          _StepButton(
            icon: Icons.add_rounded,
            size: size,
            enabled: canIncrease,
            onTap: () => onChanged(value + 1),
          ),
        ],
      ),
    );
  }
}

class _StepButton extends StatelessWidget {
  const _StepButton({
    required this.icon,
    required this.size,
    required this.enabled,
    required this.onTap,
  });

  final IconData icon;
  final double size;
  final bool enabled;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return InkWell(
      onTap: enabled ? onTap : null,
      customBorder: const CircleBorder(),
      child: SizedBox(
        width: size,
        height: size,
        child: Icon(icon, size: 16, color: enabled ? p.ink : p.inkFaint),
      ),
    );
  }
}
