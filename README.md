# Tripla Technical Test

API manajemen transaksi tiket konser menggunakan Go, Gin, GORM, dan MySQL.

Sistem ini mendukung pembuatan user dan tiket, pembuatan transaksi pembelian tiket, pembayaran, sinkronisasi ke mock accounting software, serta demo sinkronisasi stok ke sistem eksternal.

## Tech Stack

- Go
- Gin framework
- GORM
- MySQL
- Postman

## Fitur Utama

- Membuat user.
- Membuat tiket konser.
- Membuat transaksi pembelian tiket.
- Memproses transaksi pending menggunakan worker-style endpoint.
- Melakukan pembayaran transaksi.
- Mengirim transaksi sukses ke mock accounting software.
- Menerima payment webhook dari pihak ketiga.
- Menangani race condition saat stok terbatas.
- Menangani duplicate payment webhook.
- Menangani out-of-order stock synchronization.

Sistem ini belum menghandle aspek harga tiket.

## Struktur Data

Tabel utama:

- `users`: menyimpan data user/pembeli.
- `tickets`: menyimpan data tiket, total quantity, available quantity, dan stock version.
- `transactions`: menyimpan request pembelian tiket dan status transaksi.
- `payments`: menyimpan data pembayaran, termasuk webhook payment dari pihak ketiga.

Tabel tambahan untuk demo sinkronisasi stok:

- `external_data`: menyimpan mock data stok eksternal dan version terakhir yang berhasil diterapkan.

## Requirement

Pastikan sudah tersedia:

- Go
- MySQL
- Postman, opsional tetapi direkomendasikan

## Setup Dari Awal

### 1. Clone atau buka project

Masuk ke folder project:

```powershell
cd D:\zProjects\Other\tripla-technical-test
```

### 2. Setup database MySQL

Buat database MySQL:

```sql
CREATE DATABASE tripla_technical_test;
```

Pastikan user MySQL yang digunakan memiliki akses ke database tersebut.

### 3. Setup environment

Buat file `.env` berdasarkan `.env.example`.

Contoh isi `.env`:

```env
PORT=8080

DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=tripla_technical_test

ACCOUNTING_MOCK_MODE=random
```

Pilihan `ACCOUNTING_MOCK_MODE`:

- `force_success`: semua sync accounting berhasil.
- `force_fail`: semua sync accounting gagal seperti HTTP 500.
- `random` atau kosong: 90% sukses, 5% known error, 5% internal server error.

### 4. Install dependency

```powershell
go mod tidy
```

### 5. Jalankan server

```powershell
go run ./cmd/server
```

Server berjalan di:

```text
http://localhost:8080
```

Cek health endpoint:

```http
GET /health
```

Atau buka:

- `http://localhost:8080/`
- `http://localhost:8080/health`

## Postman Collection

Project ini menyediakan Postman collection:

```text
tripla-technical-test.postman_collection.json
```

Cara menggunakan:

1. Buka Postman.
2. Import file `tripla-technical-test.postman_collection.json`.
3. Pastikan variable `baseUrl` bernilai:

```text
http://localhost:8080
```

Collection tersebut berisi request untuk users, tickets, transactions, workers, webhooks, dan demo scenario.

## Transaction Lifecycle

Status transaksi:

- `pending`: request transaksi sudah tersimpan, tetapi stok belum di-reserve.
- `processing`: transaksi sedang diklaim dan diproses oleh worker.
- `waiting_for_payment`: stok sudah di-reserve dan user dapat melakukan pembayaran.
- `success`: pembayaran berhasil.
- `failed`: transaksi gagal, misalnya karena stok tidak cukup.
- `expired`: transaksi `waiting_for_payment` sudah melewati batas waktu dan stok dikembalikan.

Status accounting:

- `pending`: transaksi sukses menunggu dikirim ke accounting.
- `synced`: transaksi sukses sudah berhasil dikirim ke accounting.
- `failed`: sync accounting gagal dan dapat dicoba ulang.

## API Summary

### Users

- `POST /users`
- `GET /users`

### Tickets

- `POST /tickets`
- `GET /tickets`
- `GET /tickets/:id`

### Transactions

- `POST /transactions`
- `GET /transactions`
- `GET /transactions/:id`
- `POST /workers/transactions/process-pending`
- `POST /transactions/:id/pay`
- `POST /transactions/:id/sync-accounting`

### Webhooks

- `POST /webhooks/payments`

### Demo Endpoints

- `POST /demo/concurrency`
- `POST /demo/high-traffic`
- `POST /demo/duplicate-payment-webhook`
- `POST /demo/out-of-order-stock`

## Guide Basic Flow

### 1. Buat User

Endpoint:

```http
POST /users
```

Body:

```json
{
  "name": "Demo User",
  "email": "demo@example.com"
}
```

Expected result:

- User baru berhasil dibuat.
- Simpan `id` user untuk request berikutnya.

### 2. Buat Ticket

Endpoint:

```http
POST /tickets
```

Body:

```json
{
  "name": "VIP Concert Ticket",
  "quantity_total": 1
}
```

Expected result:

- Ticket baru berhasil dibuat.
- `quantity_available` otomatis mengikuti `quantity_total`.
- Simpan `id` ticket untuk request berikutnya.

### 3. Buat Transaction

Endpoint:

```http
POST /transactions
```

Body:

```json
{
  "user_id": 1,
  "ticket_id": 1,
  "quantity": 1
}
```

Expected result:

- Transaction dibuat dengan status `pending`.
- Stok belum dikurangi pada tahap ini.
- API mengembalikan `202 Accepted`.

### 4. Process Pending Transaction

Endpoint:

```http
POST /workers/transactions/process-pending
```

Body:

```json
{
  "limit": 1
}
```

Expected result:

- Jika stok tersedia, transaksi berubah menjadi `waiting_for_payment`.
- Stok tiket berkurang.
- Jika stok tidak tersedia, transaksi berubah menjadi `failed`.

### 5. Pay Transaction

Endpoint:

```http
POST /transactions/:id/pay
```

Body:

```json
{
  "payment_method": "credit_card"
}
```

Expected result:

- Transaction berubah menjadi `success`.
- Payment record dibuat.
- Sync ke accounting dicoba.

## Scenario 1: Race Condition

### Problem

Race condition terjadi ketika dua proses membaca stok tiket yang sama sebelum salah satu proses selesai mengurangi stok. Akibatnya, kedua proses dapat menganggap stok masih tersedia.

### Asumsi

- Service ini menangani semua proses transaksi tiket.
- Tidak ada service lain yang mengubah stok tiket tanpa menggunakan mekanisme update stok yang sama.
- User yang tidak mendapatkan tiket akan ditandai `failed`; tidak ada status waitlist.

### Solusi

Sistem menggunakan database locking, tepatnya pessimistic row-level locking dengan `SELECT FOR UPDATE`.

Saat worker memproses transaksi, row tiket dikunci terlebih dahulu sebelum sistem mengecek dan mengurangi stok. Worker lain yang ingin memproses tiket yang sama harus menunggu sampai transaksi pertama selesai. Setelah lock dilepas, worker berikutnya membaca stok terbaru. Jika stok sudah habis, transaksi berikutnya gagal.

Worker juga melakukan atomic claim dari status `pending` ke `processing`, sehingga satu transaksi pending tidak diproses oleh dua worker berbeda.

### Test Case

Endpoint:

```http
POST /demo/concurrency
```

Body:

```json
{
  "attempts": 10
}
```

Expected result:

- Hanya 1 transaksi menjadi `waiting_for_payment`.
- Sisa transaksi menjadi `failed`.
- Final stok tiket adalah `0`.

## Scenario 2: High Traffic Transactions

### Problem

Sistem dapat menerima lebih dari 10.000 request transaksi dalam waktu kurang dari 1 menit. Jika semua transaksi diproses secara synchronous di request utama, sistem berisiko mengalami timeout, tekanan database tinggi, atau lock contention.

### Asumsi

- Stok tiket cukup untuk semua transaksi yang diharapkan berhasil.
- Request transaksi valid.
- Database tersedia dan mampu menerima beban write tinggi.
- API boleh mengembalikan `202 Accepted`, artinya request diterima tetapi belum final.
- Transaksi boleh diproses secara asynchronous oleh worker.
- Kegagalan sementara seperti deadlock atau lock timeout boleh di-retry.

### Solusi

Sistem menggunakan asynchronous worker-style batch processing.

`POST /transactions` menyimpan transaksi sebagai `pending`. API merespon cepat setelah request transaksi tersimpan. Setelah itu worker memproses transaksi pending secara batch. Setiap transaksi diklaim, stok dicek dengan locking, lalu transaksi menjadi `waiting_for_payment` atau `failed`.

Demo mendukung:

- maksimal `10000` request transaksi,
- maksimal `100` worker,
- maksimal `1000` transaksi per batch worker.

### Test Case

Endpoint:

```http
POST /demo/high-traffic
```

Body:

```json
{
  "request_count": 10000,
  "ticket_stock": 10000,
  "worker_count": 10,
  "worker_batch_size": 1000
}
```

Expected result:

- `stored_pending_count = 10000`
- `processed_count = 10000`
- `waiting_for_payment_count = 10000`
- `failed_count = 0`

Jika `ticket_stock` lebih kecil dari `request_count`, sebagian transaksi memang akan menjadi `failed` karena stok tidak cukup.

## Scenario 3: Failed Third-Party Accounting Sync

### Problem

Setiap transaksi yang berhasil harus dikirim ke third-party accounting software. Jika accounting software mengembalikan error seperti HTTP 500, transaksi utama tidak boleh hilang atau berubah menjadi gagal.

### Asumsi

- Transaksi utama tetap valid jika pembayaran berhasil, walaupun sync accounting gagal.
- Sync accounting boleh dilakukan terpisah dari transaksi utama.
- Kegagalan accounting dapat bersifat sementara.
- Pengiriman ulang transaksi yang sama ke accounting aman, atau accounting mendukung idempotency.
- Sistem menyimpan status sinkronisasi accounting.

### Solusi

Sistem memisahkan status transaksi utama dan status sync accounting.

Saat payment berhasil, status transaksi menjadi `success`, dan `accounting_status` menjadi `pending`. Sistem kemudian mencoba mengirim transaksi ke accounting service.

Jika accounting menerima data, `accounting_status` menjadi `synced`. Jika accounting mengembalikan error seperti HTTP 500, `accounting_status` menjadi `failed`. Transaction tetap `success`, dan sync accounting dapat dicoba ulang.

Implementasi saat ini menyediakan retry manual melalui:

```http
POST /transactions/:id/sync-accounting
```

Pada production system, ini dapat dikembangkan menjadi automatic retry worker dengan retry limit, exponential backoff, dan monitoring.

### Test Case

Set `.env`:

```env
ACCOUNTING_MOCK_MODE=force_fail
```

Jalankan basic flow sampai payment berhasil:

1. Buat user.
2. Buat ticket.
3. Buat transaction.
4. Process pending transaction.
5. Pay transaction.

Expected result:

- Transaction status adalah `success`.
- `accounting_status` adalah `failed`.

Kemudian ubah `.env`:

```env
ACCOUNTING_MOCK_MODE=force_success
```

Restart server, lalu panggil:

```http
POST /transactions/:id/sync-accounting
```

Expected result:

- `accounting_status` berubah menjadi `synced`.

## Scenario 4: Duplicate Payment Webhook

### Problem

Third-party payment system dapat melakukan retry webhook jika tidak menerima response yang diharapkan. Karena anomali, third-party dapat mengirim dua request webhook identik pada waktu yang sama. Tanpa proteksi, sistem dapat membuat payment record duplikat.

### Asumsi

- Setiap webhook payment dari pihak ketiga memiliki `external_payment_id` yang stabil.
- Retry untuk payment yang sama menggunakan `external_payment_id` yang sama.
- Sistem boleh menganggap webhook berulang dengan `external_payment_id` yang sama sebagai duplicate.

### Solusi

Sistem menggunakan idempotency berdasarkan `external_payment_id`.

Kolom `payments.external_payment_id` memiliki unique constraint. Saat webhook diterima, sistem mencoba insert payment record. Jika dua request identik datang bersamaan, hanya satu insert yang berhasil. Insert kedua gagal karena duplicate key, lalu sistem mengambil payment existing dan mengembalikannya tanpa membuat data baru.

Dengan cara ini, hanya ada satu payment record untuk satu pembayaran pihak ketiga.

### Test Case

Endpoint:

```http
POST /demo/duplicate-payment-webhook
```

Body:

```json
{
  "external_payment_id": "pay_duplicate_demo_001",
  "concurrent_requests": 2
}
```

Expected result:

- `created_count = 1`
- `duplicate_count = 1`
- `payment_count_in_database = 1`

## Scenario 5: Data Synchronization

### Problem

Update stok tiket dapat diterima oleh sistem lain secara tidak berurutan karena latency jaringan.

Contoh:

- Update 1: `quantity = 5`
- Update 2: `quantity = 2`

Jika update 2 diterima lebih dulu, lalu update 1 diterima belakangan, sistem eksternal dapat salah menampilkan quantity `5`, padahal kondisi terbaru seharusnya `2`.

### Asumsi

- Setiap update stok memiliki `stock_version`.
- `stock_version` selalu meningkat untuk update yang lebih baru.
- Sistem penerima menyimpan version terakhir yang berhasil diterapkan.
- Update lama boleh diabaikan.
- Version terbaru lebih dipercaya daripada urutan kedatangan request.

### Solusi

Sistem menggunakan stock versioning.

Setiap update stok membawa version number. Sistem penerima hanya menerapkan update jika incoming version lebih besar dari version yang sudah tersimpan.

Jika incoming version lebih kecil atau sama dengan stored version, update dianggap stale dan diabaikan.

Contoh:

- Update 1: `quantity = 5`, `version = 1`
- Update 2: `quantity = 2`, `version = 2`

Jika version `2` diterima lebih dulu, sistem menerapkan quantity `2` dan menyimpan current version `2`. Ketika version `1` datang belakangan, update tersebut diabaikan karena lebih lama dari current version.

### Test Case

Endpoint:

```http
POST /demo/out-of-order-stock
```

Body:

```json
[
  {
    "ticket_id": 1,
    "stock": 2,
    "version": 2
  },
  {
    "ticket_id": 1,
    "stock": 5,
    "version": 1
  }
]
```

Expected result:

- Version `2` diterapkan.
- Version `1` diabaikan.
- Final external stock tetap `2`.

## Notes

- `POST /transactions` mengembalikan `202 Accepted` karena request transaksi diterima dulu dan diproses kemudian oleh worker.
- Stok hanya di-reserve ketika transaksi pending diproses oleh worker.
- Transaksi `waiting_for_payment` yang expired akan mengembalikan stok yang sebelumnya di-reserve.
- Retry accounting saat ini masih manual, tetapi dapat dikembangkan menjadi background retry worker.
