import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/catalog/catalog_models.dart';
import 'package:gas_tam_de/features/catalog/product_image.dart';

void main() {
  Product productWithImage(String? imageUrl) => Product(
        id: 'gas-12',
        sku: 'GAS12',
        name: 'Gas 12kg',
        unit: 'bình',
        salePrice: 450000,
        active: true,
        createdAt: '',
        updatedAt: '',
        imageUrl: imageUrl,
      );

  testWidgets('uses the configured HTTP image URL', (tester) async {
    const url = 'https://cdn.example.com/gas-12.jpg';
    await tester.pumpWidget(
      MaterialApp(
        home: SizedBox(
          width: 200,
          height: 120,
          child: ProductImage(product: productWithImage('  $url  ')),
        ),
      ),
    );

    final image = tester.widget<Image>(
      find.byKey(const ValueKey('product-image-gas-12')),
    );
    expect((image.image as NetworkImage).url, url);
  });

  testWidgets('shows fallback for an invalid image URL', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SizedBox(
          width: 200,
          height: 120,
          child: ProductImage(product: productWithImage('not-a-url')),
        ),
      ),
    );

    expect(find.byIcon(Icons.propane_tank_outlined), findsOneWidget);
    expect(find.byType(Image), findsNothing);
  });
}
