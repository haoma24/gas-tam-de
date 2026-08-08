import 'package:flutter/material.dart';

import '../../core/app_theme.dart';
import 'catalog_models.dart';

/// Displays a catalog image and falls back to the product icon when the URL is
/// empty, invalid, or cannot be loaded.
class ProductImage extends StatelessWidget {
  const ProductImage({
    super.key,
    required this.product,
    this.fit = BoxFit.cover,
    this.borderRadius,
  });

  final Product product;
  final BoxFit fit;
  final BorderRadius? borderRadius;

  @override
  Widget build(BuildContext context) {
    final rawUrl = product.imageUrl?.trim() ?? '';
    final uri = Uri.tryParse(rawUrl);
    final hasNetworkImage = rawUrl.isNotEmpty &&
        uri != null &&
        (uri.scheme == 'http' || uri.scheme == 'https') &&
        uri.host.isNotEmpty;

    final image = hasNetworkImage
        ? Image.network(
            rawUrl,
            key: ValueKey('product-image-${product.id}'),
            fit: fit,
            width: double.infinity,
            height: double.infinity,
            loadingBuilder: (context, child, progress) {
              if (progress == null) return child;
              return const _ProductImageFallback(showProgress: true);
            },
            errorBuilder: (_, __, ___) => const _ProductImageFallback(),
          )
        : const _ProductImageFallback();

    if (borderRadius == null) return image;
    return ClipRRect(borderRadius: borderRadius!, child: image);
  }
}

class _ProductImageFallback extends StatelessWidget {
  const _ProductImageFallback({this.showProgress = false});

  final bool showProgress;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [AppColors.ash, AppColors.coal],
        ),
      ),
      child: Center(
        child: showProgress
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  color: AppColors.amber,
                ),
              )
            : const Icon(
                Icons.propane_tank_rounded,
                color: AppColors.amber,
                size: 44,
              ),
      ),
    );
  }
}
