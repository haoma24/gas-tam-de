import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_session.dart';
import '../auth/me_api.dart';
import '../catalog/catalog_api.dart';
import '../catalog/catalog_models.dart';

/// Post-OTP brand shop: hero + catalogue + CTAs (not the raw order form).
class CustomerShopPage extends ConsumerStatefulWidget {
  const CustomerShopPage({
    super.key,
    required this.onStartOrder,
    required this.onProfile,
  });

  final VoidCallback onStartOrder;
  final VoidCallback onProfile;

  @override
  ConsumerState<CustomerShopPage> createState() => _CustomerShopPageState();
}

class _CustomerShopPageState extends ConsumerState<CustomerShopPage> {
  List<Product>? _products;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final items = await ref.read(catalogApiProvider).listActiveProducts();
      if (!mounted) return;
      setState(() {
        _products = items;
        _loading = false;
      });
    } on CatalogApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.displayMessage;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = 'Không tải được danh mục.';
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final session = ref.watch(authSessionProvider);
    final profileAsync = ref.watch(customerProfileProvider);
    final greeting = profileAsync.maybeWhen(
      data: (p) {
        if (p != null && p.hasName) return p.fullName!.trim();
        return null;
      },
      orElse: () => null,
    );
    final phone = profileAsync.maybeWhen(
          data: (p) => p?.phoneMasked,
          orElse: () => null,
        ) ??
        session?.user.phoneMasked ??
        '';

    return Scaffold(
      body: RefreshIndicator(
        onRefresh: _load,
        child: CustomScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          slivers: [
            SliverToBoxAdapter(child: _ShopHero(
              greeting: greeting,
              phoneMasked: phone,
              onStartOrder: widget.onStartOrder,
              onProfile: widget.onProfile,
            )),
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(24, 28, 24, 12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Sản phẩm đang bán',
                      style: theme.textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.w800,
                        letterSpacing: -0.4,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      'Chọn bình gas phù hợp — giao tận nơi, phí rõ ràng.',
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            ..._productSlivers(theme),
            const SliverToBoxAdapter(child: SizedBox(height: 24)),
          ],
        ),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: 0,
        onDestinationSelected: (i) {
          if (i == 1) widget.onProfile();
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.storefront_outlined),
            selectedIcon: Icon(Icons.storefront),
            label: 'Cửa hàng',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person),
            label: 'Hồ sơ',
          ),
        ],
      ),
    );
  }

  List<Widget> _productSlivers(ThemeData theme) {
    if (_loading && _products == null) {
      return [
        const SliverFillRemaining(
          hasScrollBody: false,
          child: Center(child: CircularProgressIndicator()),
        ),
      ];
    }
    if (_error != null && _products == null) {
      return [
        SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              children: [
                Text(
                  _error!,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyLarge?.copyWith(
                    color: theme.colorScheme.error,
                  ),
                ),
                const SizedBox(height: 12),
                OutlinedButton(onPressed: _load, child: const Text('Thử lại')),
              ],
            ),
          ),
        ),
      ];
    }
    final items = _products ?? const <Product>[];
    if (items.isEmpty) {
      return [
        SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text(
              'Cửa hàng chưa mở bán sản phẩm. Quay lại sau nhé.',
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ),
        ),
      ];
    }
    return [
      SliverPadding(
        padding: const EdgeInsets.fromLTRB(24, 0, 24, 16),
        sliver: SliverList.separated(
          itemCount: items.length,
          separatorBuilder: (_, __) => const SizedBox(height: 12),
          itemBuilder: (context, index) {
            final p = items[index];
            return _CatalogRow(
              product: p,
              onOrder: widget.onStartOrder,
            );
          },
        ),
      ),
    ];
  }
}

class _ShopHero extends StatelessWidget {
  const _ShopHero({
    required this.greeting,
    required this.phoneMasked,
    required this.onStartOrder,
    required this.onProfile,
  });

  final String? greeting;
  final String phoneMasked;
  final VoidCallback onStartOrder;
  final VoidCallback onProfile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final onHero = Colors.white;

    return Stack(
      children: [
        // Full-bleed brand atmosphere (flame / amber industrial).
        Container(
          width: double.infinity,
          constraints: const BoxConstraints(minHeight: 320),
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                Color(0xFF1C1917),
                Color(0xFF44403C),
                Color(0xFF9A3412),
              ],
              stops: [0.0, 0.55, 1.0],
            ),
          ),
          child: CustomPaint(
            painter: _FlameMotifPainter(),
            child: SafeArea(
              bottom: false,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(24, 12, 16, 36),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            greeting != null
                                ? 'Xin chào, $greeting'
                                : (phoneMasked.isNotEmpty
                                    ? 'Xin chào, $phoneMasked'
                                    : 'Xin chào'),
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: onHero.withValues(alpha: 0.78),
                            ),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        IconButton(
                          tooltip: 'Hồ sơ',
                          onPressed: onProfile,
                          icon: Icon(Icons.person_outline, color: onHero),
                        ),
                      ],
                    ),
                    const SizedBox(height: 28),
                    Text(
                      'Gas Tam Đệ',
                      style: theme.textTheme.displaySmall?.copyWith(
                        color: onHero,
                        fontWeight: FontWeight.w800,
                        letterSpacing: -1.2,
                        height: 1.05,
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      'Giao gas tận nơi — nhanh, rõ phí, đúng địa chỉ.',
                      style: theme.textTheme.titleMedium?.copyWith(
                        color: onHero.withValues(alpha: 0.88),
                        fontWeight: FontWeight.w500,
                        height: 1.35,
                      ),
                    ),
                    const SizedBox(height: 28),
                    FilledButton(
                      onPressed: onStartOrder,
                      style: FilledButton.styleFrom(
                        backgroundColor: const Color(0xFFF59E0B),
                        foregroundColor: const Color(0xFF1C1917),
                        minimumSize: const Size(200, 48),
                        textStyle: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w800,
                        ),
                      ),
                      child: const Text('Đặt giao gas'),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

class _CatalogRow extends StatelessWidget {
  const _CatalogRow({
    required this.product,
    required this.onOrder,
  });

  final Product product;
  final VoidCallback onOrder;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final desc = product.description?.trim();

    return InkWell(
      onTap: onOrder,
      borderRadius: BorderRadius.circular(4),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 64,
              height: 64,
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(4),
              ),
              child: Icon(
                Icons.propane_tank_outlined,
                color: theme.colorScheme.primary,
                size: 32,
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    product.name,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  if (desc != null && desc.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      desc,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                  const SizedBox(height: 6),
                  Text(
                    '${formatVnd(product.salePrice)} / ${product.unit}',
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w800,
                      color: theme.colorScheme.primary,
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

/// Soft diagonal flame streaks — atmosphere without floating badges.
class _FlameMotifPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..style = PaintingStyle.fill
      ..color = const Color(0xFFF59E0B).withValues(alpha: 0.08);
    final path = Path()
      ..moveTo(size.width * 0.55, 0)
      ..lineTo(size.width, 0)
      ..lineTo(size.width, size.height * 0.55)
      ..close();
    canvas.drawPath(path, paint);

    final paint2 = Paint()
      ..style = PaintingStyle.fill
      ..color = const Color(0xFFEA580C).withValues(alpha: 0.06);
    final path2 = Path()
      ..moveTo(size.width * 0.35, size.height)
      ..lineTo(size.width, size.height * 0.4)
      ..lineTo(size.width, size.height)
      ..close();
    canvas.drawPath(path2, paint2);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
