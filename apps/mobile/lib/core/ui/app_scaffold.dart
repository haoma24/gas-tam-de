import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'app_breakpoints.dart';
import 'app_tokens.dart';

/// Standard page chrome: title, optional back arrow, actions, a body that is
/// width-capped on desktop, and an optional sticky bottom bar.
///
/// Replaces the `Scaffold + AppBar + IconButton(arrow_back)` boilerplate that
/// was copy-pasted into 18 pages, along with their hand-wired `onBack`
/// callbacks — back now goes through go_router directly.
class AppScaffold extends StatelessWidget {
  const AppScaffold({
    super.key,
    this.title,
    required this.body,
    this.actions,
    this.bottomBar,
    this.floatingActionButton,
    this.showBack = true,
    this.onBack,
    this.backFallback = '/',
    this.padBody = true,
    this.constrainWidth = true,
    this.appBar,
  });

  final String? title;
  final Widget body;
  final List<Widget>? actions;

  /// Pinned above the keyboard/safe area — the place for a primary CTA.
  final Widget? bottomBar;

  final Widget? floatingActionButton;
  final bool showBack;

  /// Overrides the default pop-or-go behaviour.
  final VoidCallback? onBack;

  /// Where to go when there is nothing on the stack to pop (deep link, reload).
  final String backFallback;

  final bool padBody;

  /// Caps the content column on wide screens so text does not run edge-to-edge.
  final bool constrainWidth;

  /// Fully custom app bar; [title]/[actions]/[showBack] are ignored when set.
  final PreferredSizeWidget? appBar;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;

    Widget content = body;
    if (padBody) {
      content = Padding(padding: AppSpacing.pageH, child: content);
    }
    if (constrainWidth && context.isExpanded) {
      content = Align(
        alignment: Alignment.topCenter,
        child: ConstrainedBox(
          constraints: BoxConstraints(maxWidth: context.contentMaxWidth),
          child: content,
        ),
      );
    }

    return Scaffold(
      backgroundColor: p.bg,
      appBar: appBar ??
          (title == null && actions == null && !showBack
              ? null
              : AppBar(
                  title: title == null ? null : Text(title!),
                  automaticallyImplyLeading: false,
                  leading: showBack
                      ? IconButton(
                          icon: const Icon(Icons.arrow_back_rounded),
                          tooltip: 'Quay lại',
                          onPressed:
                              onBack ?? () => popOrGo(context, backFallback),
                        )
                      : null,
                  actions: actions,
                )),
      body: SafeArea(top: false, child: content),
      bottomNavigationBar:
          bottomBar == null ? null : AppBottomBar(child: bottomBar!),
      floatingActionButton: floatingActionButton,
    );
  }
}

/// Sticky bottom action bar with a hairline top edge.
class AppBottomBar extends StatelessWidget {
  const AppBottomBar({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Container(
      decoration: BoxDecoration(
        color: p.bg,
        border: Border(top: BorderSide(color: p.border)),
      ),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.lg,
            AppSpacing.md,
            AppSpacing.lg,
            AppSpacing.md,
          ),
          child: child,
        ),
      ),
    );
  }
}

/// Pops if there is something to pop, otherwise navigates to [fallback].
///
/// Flutter Web deep links and hard reloads land with an empty stack, so a bare
/// `context.pop()` would throw.
void popOrGo(BuildContext context, String fallback) {
  final router = GoRouter.of(context);
  if (router.canPop()) {
    router.pop();
  } else {
    router.go(fallback);
  }
}
