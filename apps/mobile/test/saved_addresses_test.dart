import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/catalog/catalog_models.dart';
import 'package:gas_tam_de/features/order/saved_addresses.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('saved addresses are isolated by user and keep default', () async {
    SharedPreferences.setMockInitialValues({});
    final prefs = await SharedPreferences.getInstance();
    final store = SavedAddressStore(prefs);
    const home = SavedAddress(
      id: 'home',
      name: 'Nhà',
      label: '1 Lê Lợi, TP.HCM',
      lat: 10.77,
      lng: 106.70,
      isDefault: true,
    );

    await store.save('customer-a', [home]);

    expect(store.load('customer-a'), hasLength(1));
    expect(store.load('customer-a').single.isDefault, isTrue);
    expect(store.load('customer-a').single.selection.label, home.label);
    expect(store.load('customer-b'), isEmpty);
  });

  test('product gallery supports image_urls and removes duplicates', () {
    final product = Product.fromJson({
      'id': 'p1',
      'sku': 'GAS12',
      'name': 'Gas 12kg',
      'unit': 'bình',
      'sale_price': 450000,
      'image_url': 'https://img.test/cover.jpg',
      'image_urls': [
        'https://img.test/cover.jpg',
        'https://img.test/detail.jpg',
      ],
    });

    expect(product.galleryImages, [
      'https://img.test/cover.jpg',
      'https://img.test/detail.jpg',
    ]);
  });
}
