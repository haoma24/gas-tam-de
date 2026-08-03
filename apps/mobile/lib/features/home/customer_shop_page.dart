import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import '../auth/auth_session.dart';
import '../auth/me_api.dart';
import '../catalog/catalog_api.dart';
import '../catalog/catalog_models.dart';

/// Post-OTP brand shop — hero + catalogue cards + bottom nav.
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
    final session = ref.watch(authSessionProvider);
    final profileAsync = ref.watch(customerProfileProvider);
    final name = profileAsync.maybeWhen(
      data: (p) => p?.fullName?.trim().isNotEmpty == true
          ? p!.fullName!
          : null,
      orElse: () => null,
    );
    final phone = profileAsync.maybeWhen(
          data: (p) => p?.phoneMasked,
          orElse: () => null,
        ) ??
        session?.user.phoneMasked ??
        '';

    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle.light,
      child: Scaffold(
        backgroundColor: AppColors.surface0,
        body: RefreshIndicator(
          onRefresh: _load,
          color: AppColors.fire,
          child: CustomScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            slivers: [
              _ShopHeroSliver(
                name: name,
                phone: phone,
                onStartOrder: widget.onStartOrder,
                onProfile: widget.onProfile,
              ),
              SliverToBoxAdapter(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(20, 24, 20, 4),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          'Sản phẩm',
                          style: AppTextStyles.sectionTitle(context),
                        ),
                      ),
                      if (_loading && _products != null)
                        const SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: AppColors.fire,
                          ),
                        ),
                    ],
                  ),
                ),
              ),
              ..._productSlivers(context),
              const SliverToBoxAdapter(child: SizedBox(height: 120)),
            ],
          ),
        ),
        bottomNavigationBar: _ShopBottomNav(onProfile: widget.onProfile),
        floatingActionButtonLocation:
            FloatingActionButtonLocation.centerFloat,
        floatingActionButton: _OrderFAB(onTap: widget.onStartOrder),
      ),
    );
  }

  List<Widget> _productSlivers(BuildContext context) {
    if (_loading && _products == null) {
      return [
        const SliverFillRemaining(
          hasScrollBody: false,
          child: Center(
            child: CircularProgressIndicator(color: AppColors.fire),
          ),
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
                const Icon(Icons.wifi_off_rounded,
                    size: 48, color: AppColors.ash),
                const SizedBox(height: 12),
                Text(
                  _error!,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                      fontSize: 15),
                ),
                const SizedBox(height: 16),
                OutlinedButton(
                    onPressed: _load, child: const Text('Thử lại')),
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
            padding: const EdgeInsets.fromLTRB(24, 48, 24, 24),
            child: Column(
              children: [
                const Icon(Icons.inventory_2_outlined,
                    size: 52, color: AppColors.ash),
                const SizedBox(height: 12),
                const Text(
                  'Cửa hàng chưa mở bán sản phẩm.',
                  style: TextStyle(fontSize: 15, color: AppColors.ash),
                ),
              ],
            ),
          ),
        ),
      ];
    }
    return [
      SliverPadding(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 0),
        sliver: SliverList.separated(
          itemCount: items.length,
          separatorBuilder: (_, __) => const SizedBox(height: 12),
          itemBuilder: (context, i) =>
              _ProductCard(product: items[i], onOrder: widget.onStartOrder),
        ),
      ),
    ];
  }
}

// ─────────────────────────────────────────────
// Hero sliver
// ─────────────────────────────────────────────
class _ShopHeroSliver extends StatelessWidget {
  const _ShopHeroSliver({
    required this.name,
    required this.phone,
    required this.onStartOrder,
    required this.onProfile,
  });

  final String? name;
  final String phone;
  final VoidCallback onStartOrder;
  final VoidCallback onProfile;

  @override
  Widget build(BuildContext context) {
    final greeting = name ?? (phone.isNotEmpty ? phone : 'bạn');

    return SliverToBoxAdapter(
      child: Container(
        decoration: const BoxDecoration(gradient: AppColors.heroGradient),
        child: Stack(
          children: [
            // Ambient glow
            const Positioned.fill(
              child: CustomPaint(painter: FlameAmbientPainter()),
            ),
            SafeArea(
              bottom: false,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(20, 12, 12, 28),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Top bar
                    Row(
                      children: [
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Xin chào 👋',
                              style: TextStyle(
                                color: AppColors.onDark
                                    .withValues(alpha: 0.55),
                                fontSize: 13,
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                            Text(
                              greeting,
                              style: const TextStyle(
                                color: AppColors.onDark,
                                fontSize: 16,
                                fontWeight: FontWeight.w700,
                              ),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
                        ),
                        const Spacer(),
                        GestureDetector(
                          onTap: onProfile,
                          child: Container(
                            padding: const EdgeInsets.all(10),
                            decoration: BoxDecoration(
                              color:
                                  AppColors.ash.withValues(alpha: 0.45),
                              shape: BoxShape.circle,
                            ),
                            child: const Icon(
                              Icons.person_outline_rounded,
                              color: AppColors.onDark,
                              size: 20,
                            ),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 32),
                    // Brand
                    const Text(
                      'Gas Tam Đệ',
                      style: TextStyle(
                        color: AppColors.onDark,
                        fontSize: 36,
                        fontWeight: FontWeight.w900,
                        letterSpacing: -1.2,
                        height: 1.0,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Giao gas tận nơi — nhanh, rõ phí.',
                      style: TextStyle(
                        color: AppColors.onDark.withValues(alpha: 0.72),
                        fontSize: 15,
                        fontWeight: FontWeight.w400,
                      ),
                    ),
                    const SizedBox(height: 24),
                    // Stats chips
                    Wrap(
                      spacing: 8,
                      runSpacing: 6,
                      children: [
                        _HeroChip(
                          icon: Icons.bolt_rounded,
                          label: 'Giao nhanh',
                          color: AppColors.gold,
                        ),
                        _HeroChip(
                          icon: Icons.shield_outlined,
                          label: 'An toàn',
                          color: Colors.greenAccent.shade200,
                        ),
                        _HeroChip(
                          icon: Icons.receipt_long_outlined,
                          label: 'Rõ phí',
                          color: Colors.lightBlueAccent.shade100,
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HeroChip extends StatelessWidget {
  const _HeroChip({
    required this.icon,
    required this.label,
    required this.color,
  });

  final IconData icon;
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: AppRadius.pill,
        border: Border.all(color: color.withValues(alpha: 0.3), width: 1),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: color),
          const SizedBox(width: 4),
          Text(
            label,
            style: TextStyle(
              color: color,
              fontSize: 11,
              fontWeight: FontWeight.w600,
              letterSpacing: 0.2,
            ),
          ),
        ],
      ),
    );
  }
}

// ─────────────────────────────────────────────
// Product card
// ─────────────────────────────────────────────
class _ProductCard extends StatelessWidget {
  const _ProductCard({required this.product, required this.onOrder});

  final Product product;
  final VoidCallback onOrder;

  @override
  Widget build(BuildContext context) {
    final desc = product.description?.trim();

    return GestureDetector(
      onTap: onOrder,
      child: Container(
        decoration: BoxDecoration(
          color: AppColors.surface1,
          borderRadius: AppRadius.md,
          boxShadow: AppShadow.card,
        ),
        child: Row(
          children: [
            // Icon panel
            Container(
              width: 96,
              height: 96,
              decoration: const BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [AppColors.ash, AppColors.coal],
                ),
                borderRadius: BorderRadius.only(
                  topLeft: Radius.circular(16),
                  bottomLeft: Radius.circular(16),
                ),
              ),
              child: const Center(
                child: Icon(
                  Icons.propane_tank_rounded,
                  color: AppColors.amber,
                  size: 40,
                ),
              ),
            ),
            // Info
            Expanded(
              child: Padding(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      product.name,
                      style: const TextStyle(
                        fontWeight: FontWeight.w800,
                        fontSize: 15,
                        letterSpacing: -0.2,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    if (desc != null && desc.isNotEmpty) ...[
                      const SizedBox(height: 3),
                      Text(
                        desc,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: const Color(0xFF78716C),
                          fontSize: 12.5,
                          height: 1.4,
                        ),
                      ),
                    ],
                    const SizedBox(height: 8),
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.center,
                      children: [
                        Expanded(
                          child: Text(
                            '${formatVnd(product.salePrice)} / ${product.unit}',
                            style: const TextStyle(
                              fontWeight: FontWeight.w800,
                              fontSize: 14,
                              color: AppColors.fire,
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 12, vertical: 5),
                          decoration: BoxDecoration(
                            gradient: const LinearGradient(
                              colors: [AppColors.amber, AppColors.fire],
                            ),
                            borderRadius: AppRadius.pill,
                          ),
                          child: const Text(
                            'Đặt',
                            style: TextStyle(
                              color: AppColors.obsidian,
                              fontWeight: FontWeight.w800,
                              fontSize: 12,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─────────────────────────────────────────────
// Bottom nav + FAB
// ─────────────────────────────────────────────
class _ShopBottomNav extends StatelessWidget {
  const _ShopBottomNav({required this.onProfile});
  final VoidCallback onProfile;

  @override
  Widget build(BuildContext context) {
    return NavigationBar(
      selectedIndex: 0,
      backgroundColor: AppColors.surface0,
      indicatorColor: AppColors.amber.withValues(alpha: 0.18),
      onDestinationSelected: (i) {
        if (i == 1) onProfile();
      },
      destinations: const [
        NavigationDestination(
          icon: Icon(Icons.storefront_outlined),
          selectedIcon: Icon(Icons.storefront_rounded,
              color: AppColors.fire),
          label: 'Cửa hàng',
        ),
        NavigationDestination(
          icon: Icon(Icons.person_outline_rounded),
          selectedIcon: Icon(Icons.person_rounded, color: AppColors.fire),
          label: 'Hồ sơ',
        ),
      ],
    );
  }
}

class _OrderFAB extends StatefulWidget {
  const _OrderFAB({required this.onTap});
  final VoidCallback onTap;

  @override
  State<_OrderFAB> createState() => _OrderFABState();
}

class _OrderFABState extends State<_OrderFAB>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double> _scale;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 100));
    _scale = Tween(begin: 1.0, end: 0.94)
        .animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut));
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: GestureDetector(
        onTapDown: (_) => _ctrl.forward(),
        onTapUp: (_) {
          _ctrl.reverse();
          widget.onTap();
        },
        onTapCancel: () => _ctrl.reverse(),
        child: ScaleTransition(
          scale: _scale,
          child: Container(
            height: 52,
            width: double.infinity,
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [AppColors.amber, AppColors.fire],
                begin: Alignment.centerLeft,
                end: Alignment.centerRight,
              ),
              borderRadius: AppRadius.pill,
              boxShadow: [
                BoxShadow(
                  color: AppColors.fire.withValues(alpha: 0.5),
                  blurRadius: 20,
                  offset: const Offset(0, 6),
                ),
              ],
            ),
            child: const Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.add_rounded, color: AppColors.obsidian, size: 20),
                SizedBox(width: 8),
                Text(
                  'Đặt giao gas ngay',
                  style: TextStyle(
                    color: AppColors.obsidian,
                    fontWeight: FontWeight.w800,
                    fontSize: 15,
                    letterSpacing: 0.2,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
