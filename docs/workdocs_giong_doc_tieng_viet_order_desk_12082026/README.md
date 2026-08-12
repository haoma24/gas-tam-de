# Order Desk đọc thông báo bằng giọng tiếng Việt

- **Thư mục:** `docs/workdocs_giong_doc_tieng_viet_order_desk_12082026`
- **Ngày:** 12/08/2026
- **Loại:** fix
- **Liên quan:** phản hồi cửa hàng 12/08/2026 — thông báo Order Desk đọc giọng tiếng Anh

## Mục tiêu

Thông báo «Bạn có N đơn chưa giao» ở Order Desk phát ra bằng **giọng tiếng
Anh** dù nội dung là tiếng Việt. Nghe rất khó hiểu khi đang bán hàng.

## Nguyên nhân

`NewOrderVoice._ensureReady()` đã cố chọn giọng tiếng Việt, nhưng:

1. Trình duyệt nạp danh sách giọng **bất đồng bộ** — `speechSynthesis.getVoices()`
   gọi ngay sau khi tải trang trả về **danh sách rỗng**, giọng thật chỉ xuất
   hiện sau event `voiceschanged` (giọng mạng của Chrome càng chậm).
2. Hàm đó đặt `_ready = true` **kể cả khi không tìm thấy giọng nào**, và không
   bao giờ thử lại. Nên lần chạy đầu gặp list rỗng là kẹt vĩnh viễn ở giọng mặc
   định của hệ thống — tiếng Anh.

`setLanguage('vi-VN')` một mình không đủ: nếu engine không có giọng khớp thì nó
vẫn đọc bằng giọng mặc định chứ không báo lỗi.

## Phạm vi

- Trong scope: chọn giọng đúng trên Web/Android/iOS, cách để cửa hàng tự kiểm tra
- Ngoài scope: tự cài giọng tiếng Việt cho máy (app không làm được), đổi giọng
  nam/nữ, đọc nội dung khác ngoài số đơn chờ

## Quyết định chính

- **Không cache kết quả tìm giọng thất bại.** Mỗi lần thông báo lại thử tìm một
  lần (rẻ) — giọng có thể xuất hiện muộn, hoặc admin vừa cài xong voice pack.
- **Poll khi khám phá lần đầu** (10 lần × 200ms ≈ 2s), chỉ ở `prewarm()` và
  «Nghe thử» nên không làm trễ thông báo thật.
- **`prewarm()` khi mở Order Desk** để tiếng chuông đầu tiên đã đúng giọng Việt.
- **Xếp hạng giọng**: locale `vi-VN` chính xác > locale `vi*` > tên có chứa
  "vietnam"/"việt". Mỗi engine ghi nhãn một kiểu (`vi-VN` trên Android, `vi_VN`
  trên vài bản iOS, Windows chỉ ghi trong tên "Microsoft An - Vietnamese").
- **Vẫn `setLanguage('vi-VN')` kể cả khi không thấy giọng nào** — một số engine
  tự khớp giọng theo ngôn ngữ.
- **Nói thẳng khi máy không có giọng Việt.** Đây là giới hạn của thiết bị, app
  không nhúng được giọng; nút «Nghe thử» trong Cấu hình Order Desk cho biết đang
  dùng giọng nào, hoặc hiện hướng dẫn cài đặt màu đỏ nếu chưa có giọng Việt.

## Đã làm

- [x] `NewOrderVoice` viết lại phần chọn giọng: `prewarm()`, poll, không cache thất bại
- [x] `TtsVoiceOption` + `pickVietnameseVoice` tách ra để test được thứ tự ưu tiên
- [x] `speakSample()` trả về giọng đang dùng (null nếu không phải tiếng Việt)
- [x] Order Desk gọi `NewOrderVoice.prewarm()` trong `initState`
- [x] Cấu hình Order Desk: nút «Nghe thử giọng đọc» + dòng trạng thái/hướng dẫn cài
- [x] 7 test cho xếp hạng giọng và nội dung câu đọc

## File đụng tới

| Path | Thao tác | Ghi chú |
|------|----------|---------|
| `apps/mobile/lib/features/order/new_order_voice.dart` | modified | chọn giọng có retry + API kiểm tra |
| `apps/mobile/lib/features/order/admin_orders_page.dart` | modified | `prewarm()` khi mở desk |
| `apps/mobile/lib/features/order/admin_desk_settings_page.dart` | modified | nút nghe thử + trạng thái giọng |
| `apps/mobile/test/new_order_voice_test.dart` | added | 7 test |

## Cách verify

1. Mở **Cấu hình Order Desk** → bấm **«Nghe thử giọng đọc»**.
   - Có giọng Việt: nghe đúng tiếng Việt, hiện «Đang dùng giọng: …».
   - Không có: hiện hướng dẫn cài (chữ đỏ) — cài xong **tải lại trang**.
2. Mở Order Desk khi đang có đơn chờ, chờ tới chu kỳ thông báo.
3. `flutter test test/new_order_voice_test.dart`.

Trình duyệt khuyến nghị cho máy bán hàng: **Chrome trên Windows/Android** (có
giọng Việt sẵn hoặc cài được). Safari/iOS dùng giọng hệ thống — cài trong
Cài đặt › Trợ năng › Nội dung đọc › Giọng nói.

## Ghi chú / blocker

- **App không thể tự cài giọng.** Web chỉ dùng được giọng của trình duyệt/OS;
  `flutter_tts` không có API cài voice pack. Vì vậy phần "không có giọng Việt"
  chỉ có thể hướng dẫn, không tự sửa được.
- Giọng mạng của Chrome (Google Tiếng Việt) cần Internet; mất mạng có thể rơi về
  giọng offline. Lần thông báo sau sẽ tự chọn lại.
- Trình duyệt chặn phát âm thanh trước khi người dùng tương tác với trang. Nếu
  mở Order Desk rồi để yên mà chưa click gì, thông báo đầu có thể bị chặn —
  bấm «Nghe thử» một lần là mở khóa cho cả phiên.
