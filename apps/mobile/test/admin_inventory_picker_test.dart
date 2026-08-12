import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/catalog/catalog_api.dart';
import 'package:gas_tam_de/features/catalog/catalog_models.dart';
import 'package:gas_tam_de/features/inventory/admin_inventory_page.dart';
import 'package:gas_tam_de/features/inventory/inventory_api.dart';

/// Stock rows are matched to catalog products by `product_id` at checkout, so a
/// hand-typed id that does not exist makes every order fail with
/// «Không đủ tồn kho». These tests pin the picker that removes the typing.

Map<String, dynamic> _product({
  required String id,
  required String sku,
  required String name,
  bool active = true,
}) {
  return {
    'id': id,
    'sku': sku,
    'name': name,
    'unit': 'binh',
    'sale_price': 450000,
    'active': active,
    'created_at': '2026-08-01T00:00:00Z',
    'updated_at': '2026-08-01T00:00:00Z',
  };
}

Map<String, dynamic> _stockRow({
  required String productId,
  required String sku,
  required String name,
}) {
  return {
    'product_id': productId,
    'sku': sku,
    'name': name,
    'on_hand': 5,
    'cost_price': 300000,
    'reorder_level': 2,
    'updated_at': '2026-08-01T00:00:00Z',
  };
}

/// Dio that answers the two GETs the page needs and records the POST body.
/// [catalogFails] simulates catalog-service being unreachable.
Dio _fakeDio({
  required List<Map<String, dynamic>> products,
  required List<Map<String, dynamic>> stock,
  bool catalogFails = false,
  void Function(Map<String, dynamic> body)? onPost,
}) {
  final dio = Dio();
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        Response<Map<String, dynamic>> ok(Map<String, dynamic> data) {
          return Response<Map<String, dynamic>>(
            requestOptions: options,
            statusCode: 200,
            data: data,
          );
        }

        switch ('${options.method} ${options.path}') {
          case 'GET /v1/admin/inventory':
            handler.resolve(ok({'items': stock, 'count': stock.length}));
          case 'GET /v1/admin/products':
            if (catalogFails) {
              handler.reject(
                DioException(
                  requestOptions: options,
                  type: DioExceptionType.connectionError,
                ),
                true,
              );
              return;
            }
            handler.resolve(ok({'items': products, 'count': products.length}));
          case 'POST /v1/admin/inventory':
            final body = Map<String, dynamic>.from(
              options.data as Map? ?? <String, dynamic>{},
            );
            onPost?.call(body);
            handler.resolve(ok({
              'item': _stockRow(
                productId: '${body['product_id']}',
                sku: '${body['sku'] ?? 'SKU'}',
                name: '${body['name'] ?? 'Sản phẩm'}',
              ),
              'movement': {
                'id': 'm1',
                'product_id': '${body['product_id']}',
                'movement_type': 'IN',
                'qty': body['qty'] ?? 0,
                'delta': body['qty'] ?? 0,
                'created_at': '2026-08-12T00:00:00Z',
              },
            }));
          default:
            handler.reject(
              DioException(
                requestOptions: options,
                response: Response<void>(
                  requestOptions: options,
                  statusCode: 404,
                ),
              ),
              true,
            );
        }
      },
    ),
  );
  return dio;
}

Widget _hostPage(Dio dio) {
  return ProviderScope(
    overrides: [
      inventoryApiProvider.overrideWithValue(InventoryApi(dio)),
      catalogApiProvider.overrideWithValue(CatalogApi(dio)),
    ],
    child: MaterialApp(home: AdminInventoryPage(onBack: () {})),
  );
}

Future<void> _openNewStockDialog(WidgetTester tester) async {
  await tester.pumpAndSettle();
  await tester.tap(find.text('Nhập mới'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('new stock picks a catalog product instead of typing an id',
      (tester) async {
    Map<String, dynamic>? posted;
    final dio = _fakeDio(
      products: [
        _product(id: 'prod-uuid-1', sku: 'GAS12', name: 'Gas 12kg'),
        _product(id: 'prod-uuid-2', sku: 'GAS45', name: 'Gas 45kg'),
      ],
      stock: [],
      onPost: (body) => posted = body,
    );

    await tester.pumpWidget(_hostPage(dio));
    await _openNewStockDialog(tester);

    // The free-text id field is what allowed the mismatch — it must be gone.
    expect(find.text('Mã sản phẩm (product_id)'), findsNothing);
    expect(find.byType(DropdownButtonFormField<Product>), findsOneWidget);

    await tester.tap(find.byType(DropdownButtonFormField<Product>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Gas 45kg · GAS45').last);
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Số lượng nhập'),
      '10',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Giá nhập (VND)'),
      '300000',
    );
    await tester.tap(find.text('Xác nhận'));
    await tester.pumpAndSettle();

    // id, sku and name all come from catalog, so the stock row cannot drift.
    expect(posted, isNotNull);
    expect(posted!['product_id'], 'prod-uuid-2');
    expect(posted!['sku'], 'GAS45');
    expect(posted!['name'], 'Gas 45kg');
    expect(posted!['qty'], 10);
    expect(posted!['unit_cost'], 300000);
  });

  testWidgets('products that already have a stock row are not offered again',
      (tester) async {
    final dio = _fakeDio(
      products: [
        _product(id: 'prod-uuid-1', sku: 'GAS12', name: 'Gas 12kg'),
        _product(id: 'prod-uuid-2', sku: 'GAS45', name: 'Gas 45kg'),
      ],
      stock: [
        _stockRow(productId: 'prod-uuid-1', sku: 'GAS12', name: 'Gas 12kg'),
      ],
    );

    await tester.pumpWidget(_hostPage(dio));
    await _openNewStockDialog(tester);

    await tester.tap(find.byType(DropdownButtonFormField<Product>));
    await tester.pumpAndSettle();

    expect(find.text('Gas 45kg · GAS45'), findsOneWidget);
    expect(find.text('Gas 12kg · GAS12'), findsNothing);
  });

  testWidgets('inactive catalog products stay stockable but are labelled',
      (tester) async {
    final dio = _fakeDio(
      products: [
        _product(
          id: 'prod-uuid-3',
          sku: 'GAS06',
          name: 'Gas 6kg',
          active: false,
        ),
      ],
      stock: [],
    );

    await tester.pumpWidget(_hostPage(dio));
    await _openNewStockDialog(tester);

    await tester.tap(find.byType(DropdownButtonFormField<Product>));
    await tester.pumpAndSettle();

    expect(find.text('Gas 6kg · GAS06 (ngừng bán)'), findsOneWidget);
  });

  testWidgets('catalog unreachable falls back to manual id with a warning',
      (tester) async {
    final dio = _fakeDio(products: [], stock: [], catalogFails: true);

    await tester.pumpWidget(_hostPage(dio));
    await _openNewStockDialog(tester);

    // Losing catalog must not block stocking, but the risk is spelled out.
    expect(find.byType(DropdownButtonFormField<Product>), findsNothing);
    expect(find.text('Mã sản phẩm (product_id)'), findsOneWidget);
    expect(
      find.textContaining('Không tải được danh mục sản phẩm'),
      findsOneWidget,
    );
  });

  testWidgets('«Nhập mới» is hidden once every product has a stock row',
      (tester) async {
    // inventory-service syncs a row per catalog product, so this is the normal
    // steady state: stocking happens on the row, not through "Nhập mới".
    final dio = _fakeDio(
      products: [
        _product(id: 'prod-uuid-1', sku: 'GAS12', name: 'Gas 12kg'),
      ],
      stock: [
        _stockRow(productId: 'prod-uuid-1', sku: 'GAS12', name: 'Gas 12kg'),
      ],
    );

    await tester.pumpWidget(_hostPage(dio));
    await tester.pumpAndSettle();

    expect(find.text('Nhập mới'), findsNothing);
    expect(find.text('Gas 12kg'), findsOneWidget);
  });

  testWidgets('«Nhập mới» stays available while a product has no stock row',
      (tester) async {
    final dio = _fakeDio(
      products: [
        _product(id: 'prod-uuid-1', sku: 'GAS12', name: 'Gas 12kg'),
        _product(id: 'prod-uuid-2', sku: 'GAS45', name: 'Gas 45kg'),
      ],
      stock: [
        _stockRow(productId: 'prod-uuid-1', sku: 'GAS12', name: 'Gas 12kg'),
      ],
    );

    await tester.pumpWidget(_hostPage(dio));
    await tester.pumpAndSettle();

    expect(find.text('Nhập mới'), findsOneWidget);
  });
}
