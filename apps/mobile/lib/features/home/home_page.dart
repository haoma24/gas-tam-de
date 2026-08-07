import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/app_theme.dart';

/// Guest landing — full-screen immersive brand. Single «Đăng nhập» CTA.
class HomePage extends StatelessWidget {
  const HomePage({super.key, required this.onLogin});

  final VoidCallback onLogin;

  @override
  Widget build(BuildContext context) {
    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle.light,
      child: Scaffold(
        backgroundColor: AppColors.obsidian,
        body: Stack(
          children: [
            // Ambient flame background.
            const Positioned.fill(
              child: CustomPaint(painter: FlameAmbientPainter()),
            ),
            // Content.
            SafeArea(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Top badge.
                  Padding(
                    padding: const EdgeInsets.fromLTRB(28, 24, 28, 0),
                    child: Row(
                      children: [
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 12, vertical: 6),
                      decoration: BoxDecoration(
                        color: AppColors.amber.withValues(alpha: 0.18),
                        borderRadius: AppRadius.pill,
                        border: Border.all(
                          color: AppColors.amber.withValues(alpha: 0.35),
                          width: 1,
                        ),
                      ),
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.local_fire_department,
                              color: AppColors.amber, size: 14),
                          SizedBox(width: 6),
                          Flexible(
                            child: Text(
                              'Giao nhanh trong ngày',
                              style: TextStyle(
                                color: AppColors.amber,
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                                letterSpacing: 0.2,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                      ],
                    ),
                  ),

                  const Spacer(flex: 2),

                  // Brand section.
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 28),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Flame icon mark.
                        Container(
                          width: 64,
                          height: 64,
                          decoration: BoxDecoration(
                            gradient: const LinearGradient(
                              begin: Alignment.topLeft,
                              end: Alignment.bottomRight,
                              colors: [AppColors.amber, AppColors.fire],
                            ),
                            borderRadius: AppRadius.md,
                            boxShadow: [
                              BoxShadow(
                                color: AppColors.fire.withValues(alpha: 0.45),
                                blurRadius: 20,
                                offset: const Offset(0, 8),
                              ),
                            ],
                          ),
                          child: const Icon(
                            Icons.local_fire_department_rounded,
                            color: Colors.white,
                            size: 34,
                          ),
                        ),
                        const SizedBox(height: 24),
                        Text(
                          'Gas\nTam Đệ',
                          style: Theme.of(context)
                              .textTheme
                              .displayMedium
                              ?.copyWith(
                                color: AppColors.onDark,
                                fontWeight: FontWeight.w900,
                                letterSpacing: -2,
                                height: 0.95,
                              ),
                        ),
                        const SizedBox(height: 18),
                        Text(
                          'Giao gas tận nơi — nhanh, rõ phí,\nđúng địa chỉ.',
                          style: Theme.of(context)
                              .textTheme
                              .titleMedium
                              ?.copyWith(
                                color:
                                    AppColors.onDark.withValues(alpha: 0.70),
                                fontWeight: FontWeight.w400,
                                height: 1.5,
                              ),
                        ),
                      ],
                    ),
                  ),

                  const Spacer(flex: 3),

                  // Bottom card.
                  Container(
                    margin: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                    padding: const EdgeInsets.fromLTRB(24, 28, 24, 28),
                    decoration: BoxDecoration(
                      color: AppColors.coal.withValues(alpha: 0.9),
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
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(
                          'Bắt đầu đặt hàng',
                          style: Theme.of(context)
                              .textTheme
                              .titleMedium
                              ?.copyWith(
                                color: AppColors.onDark,
                                fontWeight: FontWeight.w700,
                              ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'Xác thực số điện thoại, không cần tạo tài khoản.',
                          style: Theme.of(context)
                              .textTheme
                              .bodyMedium
                              ?.copyWith(
                                color:
                                    AppColors.onDark.withValues(alpha: 0.55),
                              ),
                        ),
                        const SizedBox(height: 20),
                        _LoginButton(onLogin: onLogin),
                      ],
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

class _LoginButton extends StatefulWidget {
  const _LoginButton({required this.onLogin});
  final VoidCallback onLogin;

  @override
  State<_LoginButton> createState() => _LoginButtonState();
}

class _LoginButtonState extends State<_LoginButton>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double> _scale;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 120));
    _scale = Tween(begin: 1.0, end: 0.96).animate(
        CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut));
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTapDown: (_) => _ctrl.forward(),
      onTapUp: (_) {
        _ctrl.reverse();
        widget.onLogin();
      },
      onTapCancel: () => _ctrl.reverse(),
      child: ScaleTransition(
        scale: _scale,
        child: Container(
          height: 56,
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              colors: [AppColors.amber, AppColors.fire],
              begin: Alignment.centerLeft,
              end: Alignment.centerRight,
            ),
            borderRadius: AppRadius.pill,
            boxShadow: [
              BoxShadow(
                color: AppColors.fire.withValues(alpha: 0.45),
                blurRadius: 20,
                offset: const Offset(0, 8),
              ),
            ],
          ),
          child: const Center(
            child: Text(
              'Đăng nhập',
              style: TextStyle(
                color: AppColors.obsidian,
                fontWeight: FontWeight.w800,
                fontSize: 16,
                letterSpacing: 0.3,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
