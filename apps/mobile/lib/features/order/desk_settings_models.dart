/// Admin Order Desk settings (`GET/PUT /v1/admin/desk-settings`).
class DeskSettings {
  const DeskSettings({
    required this.waitBlueMaxMin,
    required this.waitOrangeMaxMin,
    required this.waitRedMaxMin,
    required this.alertEnabled,
    required this.alertIntervalSec,
    this.updatedAt,
  });

  final int waitBlueMaxMin;
  final int waitOrangeMaxMin;
  final int waitRedMaxMin;
  final bool alertEnabled;
  final int alertIntervalSec;
  final String? updatedAt;

  factory DeskSettings.fromJson(Map<String, dynamic> json) {
    return DeskSettings(
      waitBlueMaxMin: (json['wait_blue_max_min'] as num?)?.toInt() ?? 5,
      waitOrangeMaxMin: (json['wait_orange_max_min'] as num?)?.toInt() ?? 15,
      waitRedMaxMin: (json['wait_red_max_min'] as num?)?.toInt() ?? 30,
      alertEnabled: json['alert_enabled'] == true || json['alert_enabled'] == 1,
      alertIntervalSec: (json['alert_interval_sec'] as num?)?.toInt() ?? 300,
      updatedAt: json['updated_at'] as String?,
    );
  }

  Map<String, dynamic> toPutJson() => {
        'wait_blue_max_min': waitBlueMaxMin,
        'wait_orange_max_min': waitOrangeMaxMin,
        'wait_red_max_min': waitRedMaxMin,
        'alert_enabled': alertEnabled,
        'alert_interval_sec': alertIntervalSec,
      };

  static const DeskSettings defaults = DeskSettings(
    waitBlueMaxMin: 5,
    waitOrangeMaxMin: 15,
    waitRedMaxMin: 30,
    alertEnabled: true,
    alertIntervalSec: 300,
  );
}

enum WaitUrgency { blue, orange, red }

WaitUrgency waitUrgencyFor(Duration waited, DeskSettings s) {
  final m = waited.inMinutes;
  if (m < s.waitBlueMaxMin) return WaitUrgency.blue;
  if (m < s.waitOrangeMaxMin) return WaitUrgency.orange;
  return WaitUrgency.red;
}
