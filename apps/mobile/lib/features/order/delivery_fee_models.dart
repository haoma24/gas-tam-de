/// Admin delivery-fee config from `GET/PUT /v1/admin/delivery-fee`.
class DeliveryFeeConfig {
  const DeliveryFeeConfig({
    required this.enabled,
    required this.rules,
    this.updatedAt,
  });

  final bool enabled;
  final String? updatedAt;
  final List<DeliveryFeeRule> rules;

  factory DeliveryFeeConfig.fromJson(Map<String, dynamic> json) {
    final rawRules = json['rules'];
    final rules = <DeliveryFeeRule>[];
    if (rawRules is List) {
      for (final item in rawRules) {
        if (item is Map) {
          rules.add(
            DeliveryFeeRule.fromJson(Map<String, dynamic>.from(item)),
          );
        }
      }
    }
    return DeliveryFeeConfig(
      enabled: json['enabled'] as bool? ?? false,
      updatedAt: json['updated_at'] as String?,
      rules: rules,
    );
  }
}

/// Distance band: half-open `[min_km, max_km)`; `maxKm == null` → +∞.
class DeliveryFeeRule {
  const DeliveryFeeRule({
    required this.id,
    required this.minKm,
    required this.feeVnd,
    required this.sortOrder,
    required this.active,
    this.maxKm,
  });

  final String id;
  final double minKm;
  final double? maxKm;
  final int feeVnd;
  final int sortOrder;
  final bool active;

  factory DeliveryFeeRule.fromJson(Map<String, dynamic> json) {
    return DeliveryFeeRule(
      id: json['id'] as String? ?? '',
      minKm: (json['min_km'] as num?)?.toDouble() ?? 0,
      maxKm: (json['max_km'] as num?)?.toDouble(),
      feeVnd: (json['fee_vnd'] as num?)?.toInt() ?? 0,
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
      active: json['active'] as bool? ?? true,
    );
  }

  Map<String, dynamic> toPutJson() {
    return {
      if (id.isNotEmpty) 'id': id,
      'min_km': minKm,
      'max_km': maxKm,
      'fee_vnd': feeVnd,
      'sort_order': sortOrder,
      'active': active,
    };
  }
}
