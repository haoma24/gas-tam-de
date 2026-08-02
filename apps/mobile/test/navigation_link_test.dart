import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/order/navigation_link.dart';

void main() {
  const lat = 10.776889;
  const lng = 106.700897;

  test('googleMapsDirectionsUri uses destination + driving, no origin', () {
    final uri = googleMapsDirectionsUri(lat, lng);
    expect(uri.scheme, 'https');
    expect(uri.host, 'www.google.com');
    expect(uri.path, '/maps/dir/');
    expect(uri.queryParameters['api'], '1');
    expect(uri.queryParameters['destination'], '10.776889,106.700897');
    expect(uri.queryParameters['travelmode'], 'driving');
    expect(uri.queryParameters.containsKey('origin'), isFalse);
  });

  test('geoIntentUri encodes lat,lng query', () {
    final uri = geoIntentUri(lat, lng);
    expect(uri.scheme, 'geo');
    expect(uri.path, '10.776889,106.700897');
    expect(uri.queryParameters['q'], '10.776889,106.700897');
  });

  test('googleNavigationUri is Android navigation scheme (opaque path)', () {
    final uri = googleNavigationUri(lat, lng);
    expect(uri.scheme, 'google.navigation');
    expect(
      uri.toString(),
      'google.navigation:q=10.776889,106.700897&mode=d',
    );
  });

  test('googleMapsAppDirectionsUri uses comgooglemaps daddr', () {
    final uri = googleMapsAppDirectionsUri(lat, lng);
    expect(uri.scheme, 'comgooglemaps');
    expect(uri.queryParameters['daddr'], '10.776889,106.700897');
    expect(uri.queryParameters['directionsmode'], 'driving');
    expect(uri.toString(), startsWith('comgooglemaps://?'));
  });

  test('appleMapsDirectionsUri uses maps daddr', () {
    final uri = appleMapsDirectionsUri(lat, lng);
    expect(uri.scheme, 'maps');
    expect(uri.queryParameters['daddr'], '10.776889,106.700897');
    expect(uri.queryParameters['dirflg'], 'd');
    expect(uri.toString(), startsWith('maps://?'));
  });
  test('navigationCandidateUris always ends with HTTPS fallback', () {
    final uris = navigationCandidateUris(lat, lng);
    expect(uris, isNotEmpty);
    expect(uris.last, googleMapsDirectionsUri(lat, lng));
  });

  test('navigationCandidateUris rejects non-finite coords', () {
    expect(
      () => navigationCandidateUris(double.nan, lng),
      throwsArgumentError,
    );
  });
}
