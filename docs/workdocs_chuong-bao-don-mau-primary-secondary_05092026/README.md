# Chuông báo đơn thay giọng đọc, màu primary/secondary, giao diện mặc định Sáng

**Ngày:** 05/09/2026
**Loại:** feat + fix
**Phạm vi:** `apps/mobile` (Order Desk, cấu hình desk, theme), `deploy/Dockerfile.web`, `docs/codemap.md`

## Bối cảnh

Bốn việc chủ shop báo về menu admin/khách:

1. Thông báo đơn mới **đọc bằng giọng nói** — nghe không rõ trong tiếng ồn cửa hàng, và
   máy nào chưa cài voice pack tiếng Việt thì đọc bằng giọng Anh. Muốn **chuông báo
   thức**: kêu liên tục kèm popup có «Không hiển thị lại» và «Snooze».
2. Icon tab **Báo cáo** và **Cài đặt** vẫn trống.
3. Giao diện phải **mặc định Sáng**.
4. Giao diện **đơn sắc** khó chịu — cần primary color cho nút, secondary cho thông tin.

## 1. Chuông báo thay giọng đọc

`new_order_voice.dart` (+ `flutter_tts`) bị xoá, thay bằng `new_order_alarm.dart`.

- **Âm thanh** sinh tại chỗ thay vì commit một file nhị phân không ai review được:
  `buildAlarmWav()` dựng 1 giây PCM 16-bit mono trong header WAV 44 byte — hai tiếng
  «ting» (C6 rồi E6, mỗi tiếng 130 ms) rồi im. Đuôi im chính là chỗ vòng lặp nối lại,
  nên phát `ReleaseMode.loop` không bị «pop» ở mối nối; mỗi tiếng có fade 6 ms hai đầu
  vì cạnh vuông của sóng sin kêu lụp bụp.
- **Phát** bằng `audioplayers` (thay `flutter_tts`) — chạy cùng một đường trên Web,
  Android, iOS. Trình duyệt chặn audio trước khi có tương tác, nhưng vào được Order Desk
  là đã bấm qua màn hình đăng nhập nên điều kiện đó luôn thoả.
- **Popup** `showNewOrderAlarmDialog()` khoá cả barrier lẫn nút back (`PopScope`), trả về:
  - `Duration` = snooze **5 / 10 / 15 / 30 phút**
  - `null` = «Không hiển thị lại» — tắt hẳn cho phiên làm việc này
  Không có kết quả thứ ba, nên chuông không thể tắt do bấm nhầm ra ngoài.
- **Order Desk** gom mute/snooze vào một chỗ duy nhất là `_raiseAlarm()`: cả nhắc theo
  chu kỳ (`_restartAlertTimer`) lẫn đơn mới về (`_notifyNewOrders`) đều đi qua đây, và
  `_alarmOpen` chặn đơn thứ hai chồng thêm một popup nữa lên popup đang mở.
- **Cấu hình Order Desk**: «Thông báo giọng nói» → «Chuông báo đơn chờ», nút «Nghe thử
  chuông báo» phát ~2,5 giây để chỉnh âm lượng máy trước khi có đơn thật. Phần hướng dẫn
  cài voice pack tiếng Việt không còn lý do tồn tại nên bỏ.

`alertEnabled` / `alertIntervalSec` của desk giữ nguyên ngữ nghĩa (bật/tắt + chu kỳ nhắc lại),
không đụng API hay DB.

## 2. Icon tab Báo cáo / Cài đặt

Lần trước (`d9b2a8e`) đã sửa **cache**: thêm `Cache-Control: no-cache` cho `/assets/` để
buộc revalidate. Vẫn trống, nên lần này sửa **nguyên nhân gốc** thay vì cách trình duyệt
giữ file: `deploy/Dockerfile.web` build với `--no-tree-shake-icons`.

Mặc định Flutter cắt `MaterialIcons-Regular.otf` xuống đúng những icon build đó dùng. Font
subset **đổi nội dung mỗi lần thêm icon** nhưng URL thì không đổi (Flutter không hash tên
asset), nên bất kỳ trình duyệt/proxy nào còn giữ bản cũ đều vẽ icon mới thành ô trống. Với
`--no-tree-shake-icons` font là bản đầy đủ, **byte-identical giữa mọi lần build**, nên một
bản cache lại không bao giờ sai được nữa.

Giá phải trả: 1,6 MB font thay vì ~10 KB subset (brotli -q 11 trong Dockerfile kéo xuống
đáng kể, và tải một lần rồi cache vĩnh viễn).

## 3. Giao diện mặc định Sáng

`ThemeModeController._restore()`: giá trị chưa lưu → `ThemeMode.light` thay vì
`ThemeMode.system`. «Hệ thống» vẫn là một lựa chọn, chỉ không còn là mặc định.

## 4. Primary / secondary color

`AppPalette` trước đây chỉ có `ink` (đen/trắng) + `accent` dùng đúng một chỗ là FAB —
đó chính là lý do màn hình đơn sắc. Nay:

| Token | Sáng | Tối | Dùng ở đâu |
|---|---|---|---|
| `primary` / `onPrimary` | `#EA580C` | `#FB923C` | Nút chính, FAB, chip/segment/switch/checkbox/radio khi chọn, tab + nav đang chọn, viền input focus, progress |
| `secondary` / `onSecondary` | `#0284C7` | `#38BDF8` | Icon tiêu đề `AppSection`, icon `ListTile`, `colorScheme.secondary` / `tertiary` |

- `accent` giữ lại làm **getter trỏ vào `primary`** — accent và primary vốn là cùng một màu
  thương hiệu, nên không phải đổi tên 5 call site đang dùng.
- `primaryContainer` / `secondaryContainer` được `_tint()` **flatten trên nền trang** thay vì
  để alpha: một card tô màu trong suốt sẽ lộ thứ nằm dưới nó.
- `ink` lui về đúng vai trò chữ và tiêu đề.

## Kiểm tra

- `flutter test` — 71 test pass. `new_order_voice_test.dart` → `new_order_alarm_test.dart`:
  header WAV khai báo đúng kích thước, hai tiếng chuông có tín hiệu, khoảng giữa và đuôi
  im tuyệt đối, popup trả đúng snooze / `null`, barrier không tắt được chuông.
- `test/app_theme_test.dart` bổ sung: `colorScheme.primary` = `palette.primary`, nền
  FilledButton = `primary`, container **không** trong suốt.
- `test/theme_mode_test.dart`: mặc định là Sáng, «Hệ thống» vẫn khôi phục được.
- `flutter analyze` — 7 info có sẵn từ trước (deprecation `Radio.groupValue`, dangling doc).
- `flutter build web --release --no-tree-shake-icons` — build được, `build/web/assets/fonts/`
  ra font đầy đủ 1.645.184 byte.

## Việc chưa làm

- Chuông chỉ có một tông. Nếu shop muốn chọn âm khác thì thêm `chime()` khác trong
  `buildAlarmWav()` và một field trong desk settings — chưa ai yêu cầu.
- «Không hiển thị lại» tắt theo **phiên** (biến trong state), không lưu xuống
  `SharedPreferences`: mở lại Order Desk là chuông sống lại, đúng ý một cái báo thức.
- `secondary` mới dùng cho icon `AppSection` / `ListTile`. Muốn phủ thêm (badge thông tin,
  link) thì thêm `AppBadgeTone.info` khi có chỗ dùng thật.
