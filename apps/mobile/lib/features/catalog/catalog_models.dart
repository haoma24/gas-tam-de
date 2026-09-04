/// Catalog product as returned by admin CRUD APIs.
class Product {
  const Product({
    required this.id,
    required this.sku,
    required this.name,
    required this.unit,
    required this.salePrice,
    required this.active,
    required this.createdAt,
    required this.updatedAt,
    this.description,
    this.imageUrl,
    this.imageUrls = const [],
  });

  final String id;
  final String sku;
  final String name;
  final String? description;
  final String unit;
  final int salePrice;
  final bool active;
  final String? imageUrl;
  final List<String> imageUrls;
  final String createdAt;
  final String updatedAt;

  factory Product.fromJson(Map<String, dynamic> json) {
    final rawImages = json['image_urls'];
    final images = rawImages is List
        ? rawImages
            .whereType<String>()
            .map((value) => value.trim())
            .where((value) => value.isNotEmpty)
            .toList(growable: false)
        : const <String>[];
    final legacyImage = (json['image_url'] as String?)?.trim();
    return Product(
      id: json['id'] as String? ?? '',
      sku: json['sku'] as String? ?? '',
      name: json['name'] as String? ?? '',
      description: json['description'] as String?,
      unit: json['unit'] as String? ?? 'binh',
      salePrice: (json['sale_price'] as num?)?.toInt() ?? 0,
      active: json['active'] as bool? ?? true,
      imageUrl: legacyImage,
      imageUrls: images.isNotEmpty
          ? images
          : legacyImage != null && legacyImage.isNotEmpty
              ? [legacyImage]
              : const [],
      createdAt: json['created_at'] as String? ?? '',
      updatedAt: json['updated_at'] as String? ?? '',
    );
  }

  List<String> get galleryImages {
    final values = <String>{};
    for (final value in [if (imageUrl != null) imageUrl!, ...imageUrls]) {
      final normalized = value.trim();
      if (normalized.isNotEmpty) values.add(normalized);
    }
    return values.toList(growable: false);
  }
}

/// Formats VND integer with thousand separators (e.g. `450000` → `450.000 ₫`).
String formatVnd(int amount) {
  final negative = amount < 0;
  final digits = amount.abs().toString();
  final buf = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 == 0) {
      buf.write('.');
    }
    buf.write(digits[i]);
  }
  return '${negative ? '-' : ''}${buf.toString()} ₫';
}
