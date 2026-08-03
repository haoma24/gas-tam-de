import 'package:flutter/material.dart';

/// Centralized design tokens for Gas Tam Đệ.
abstract final class AppColors {
  // --- Fire palette (brand) ---
  static const obsidian = Color(0xFF1C1917); // darkest surface
  static const coal     = Color(0xFF292524);
  static const ash      = Color(0xFF44403C);
  static const brick    = Color(0xFF9A3412);
  static const fire     = Color(0xFFEA580C); // orange accent
  static const amber    = Color(0xFFF59E0B); // highlight
  static const gold     = Color(0xFFFBBF24);

  // --- Neutral surfaces (light mode body) ---
  static const surface0 = Color(0xFFFAF7F4); // page background
  static const surface1 = Color(0xFFF5EFE8); // card fill
  static const surface2 = Color(0xFFEDE4D7); // secondary card

  // --- Semantic ---
  static const success  = Color(0xFF16A34A);
  static const danger   = Color(0xFFDC2626);

  // --- On-dark text ---
  static const onDark   = Colors.white;

  // --- Hero gradient stops (top-left → bottom-right) ---
  static const heroGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [obsidian, ash, brick],
    stops: [0.0, 0.50, 1.0],
  );

  static const subtleHeroGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [coal, ash],
    stops: [0.0, 1.0],
  );
}

abstract final class AppTextStyles {
  static TextStyle displayBrand(BuildContext context) =>
      Theme.of(context).textTheme.displaySmall!.copyWith(
        color: AppColors.onDark,
        fontWeight: FontWeight.w900,
        letterSpacing: -1.5,
        height: 1.0,
      );

  static TextStyle tagline(BuildContext context) =>
      Theme.of(context).textTheme.titleMedium!.copyWith(
        color: AppColors.onDark.withValues(alpha: 0.82),
        fontWeight: FontWeight.w400,
        height: 1.4,
      );

  static TextStyle sectionTitle(BuildContext context) =>
      Theme.of(context).textTheme.titleLarge!.copyWith(
        fontWeight: FontWeight.w800,
        letterSpacing: -0.3,
      );

  static TextStyle price(BuildContext context) =>
      Theme.of(context).textTheme.titleMedium!.copyWith(
        fontWeight: FontWeight.w800,
        color: AppColors.fire,
      );

  static TextStyle priceSmall(BuildContext context) =>
      Theme.of(context).textTheme.labelLarge!.copyWith(
        fontWeight: FontWeight.w700,
        color: AppColors.fire,
      );
}

abstract final class AppRadius {
  static const sm  = BorderRadius.all(Radius.circular(12));
  static const md  = BorderRadius.all(Radius.circular(16));
  static const lg  = BorderRadius.all(Radius.circular(24));
  static const xl  = BorderRadius.all(Radius.circular(32));
  static const pill = BorderRadius.all(Radius.circular(100));
}

abstract final class AppShadow {
  static List<BoxShadow> card = [
    BoxShadow(
      color: AppColors.ash.withValues(alpha: 0.18),
      blurRadius: 20,
      offset: const Offset(0, 8),
    ),
  ];

  static List<BoxShadow> hero = [
    BoxShadow(
      color: AppColors.brick.withValues(alpha: 0.35),
      blurRadius: 40,
      spreadRadius: -8,
      offset: const Offset(0, 20),
    ),
  ];
}

/// Reusable fire-flame ambient background painter.
class FlameAmbientPainter extends CustomPainter {
  const FlameAmbientPainter({this.opacity = 1.0});
  final double opacity;

  @override
  void paint(Canvas canvas, Size size) {
    // Top-right ambient orb
    canvas.drawCircle(
      Offset(size.width * 0.82, size.height * 0.15),
      size.width * 0.55,
      Paint()
        ..color = AppColors.fire.withValues(alpha: 0.08 * opacity)
        ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 60),
    );
    // Bottom-left warm glow
    canvas.drawCircle(
      Offset(size.width * 0.12, size.height * 0.88),
      size.width * 0.45,
      Paint()
        ..color = AppColors.amber.withValues(alpha: 0.06 * opacity)
        ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 50),
    );
    // Diagonal streak
    final paint = Paint()
      ..style = PaintingStyle.fill
      ..color = AppColors.amber.withValues(alpha: 0.05 * opacity);
    canvas.drawPath(
      Path()
        ..moveTo(size.width * 0.6, 0)
        ..lineTo(size.width, 0)
        ..lineTo(size.width, size.height * 0.5)
        ..close(),
      paint,
    );
  }

  @override
  bool shouldRepaint(covariant FlameAmbientPainter old) =>
      old.opacity != opacity;
}
